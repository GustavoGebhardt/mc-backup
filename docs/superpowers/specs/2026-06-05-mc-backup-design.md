# mc-backup — Design Spec

**Date:** 2026-06-05  
**Status:** Approved

---

## Overview

A Go CLI binary that runs on the Minecraft server machine, sends RCON commands to safely flush the world to disk, compresses the world directory to a tar.gz archive, uploads it to a NAS via NFS (mounted on-demand), and applies Proxmox-style generational retention. Runs unattended via a systemd timer.

---

## Architecture

Plugin-style: a `Storage` interface decouples the backup core from the destination. Today's implementation is `NFSStorage`; future implementations (S3, SFTP) only need to satisfy the interface.

```
mc-backup/
  cmd/
    mc-backup/
      main.go               # entrypoint: wire dependencies, run pipeline
  internal/
    config/
      config.go             # Config struct + Load() via godotenv + validation
      config_test.go
    rcon/
      client.go             # RCON client wrapping gorcon/rcon
      client_test.go
    archive/
      archive.go            # tar.gz creation into a tmp dir
      archive_test.go
    storage/
      storage.go            # Storage interface
      nfs/
        nfs.go              # NFSStorage: mount → upload → prune → unmount
        nfs_test.go
    retention/
      retention.go          # generational selection algorithm (Proxmox-style)
      retention_test.go
  systemd/
    mc-backup.service
    mc-backup.timer
  .env.example
  .gitignore
  go.mod
```

---

## Execution Flow

```
1. Load config from .env
2. Connect to RCON
3. Send: save-off → save-all → sleep(5s) for I/O flush
4. Create tar.gz of MINECRAFT_DIR into BACKUP_TMP_DIR
5. Send: save-on → disconnect RCON
6. storage.Upload(ctx, archivePath)
   └── NFSStorage: mount NFS → copy file → unmount
7. storage.Prune(ctx, retentionPolicy)
   └── NFSStorage: mount NFS → apply generational retention → unmount
8. Delete local tmp archive
9. Log summary (kept, deleted, space freed)
```

RCON is disconnected and `save-on` is re-sent **before** the upload, so the server resumes normal autosave while the (potentially slow) NFS transfer happens.

On any error: re-send `save-on`, delete tmp archive, exit with code 1. The systemd unit captures the exit code; failures appear in `journalctl -u mc-backup`.

---

## Storage Interface

```go
type Storage interface {
    Upload(ctx context.Context, path string) error
    Prune(ctx context.Context, policy retention.Policy) error
}
```

---

## Configuration (`.env`)

```bash
# Minecraft
MINECRAFT_DIR=/home/ubuntu/minecraft-server/world

# RCON
RCON_HOST=localhost
RCON_PORT=25575
RCON_PASSWORD=changeme

# NFS
NFS_HOST=192.168.1.100
NFS_SHARE=/volume1/minecraft
NFS_MOUNT_POINT=/mnt/mc-backup

# Retention (Proxmox-style generations)
RETENTION_HOURLY=24
RETENTION_DAILY=7
RETENTION_WEEKLY=4
RETENTION_MONTHLY=12

# Misc
BACKUP_TMP_DIR=/tmp/mc-backup
```

---

## Generational Retention

Archive filenames embed a UTC timestamp: `mc_backup_20260605_030000.tar.gz`.

The retention algorithm:
1. Lists all archives on the NAS sorted by time descending.
2. Walks each generation bucket (hourly, daily, weekly, monthly).
3. For each bucket, keeps the **most recent** backup that falls within each slot (e.g. one per calendar hour, one per calendar day, etc.).
4. Any archive not selected by any bucket is deleted.

After prune, logs:

```
[INFO] Retention: kept 24 hourly, 7 daily, 4 weekly, 3 monthly (38 total, 1.9 GB)
[INFO] Pruned 5 backups (250 MB freed)
```

Estimated steady-state NAS usage after 1 year: ~2.4 GB (based on 136 MB world, ~50 MB compressed).

---

## Logging

Uses `log/slog` (Go 1.21 stdlib) with structured JSON output in production and human-readable text in TTY. Rich detail at each step:

- Archive created: path, size, duration
- RCON commands sent and responses received
- NFS mount/unmount with timing
- Upload: destination path, bytes transferred, duration
- Retention plan: each kept/deleted file with reason
- Final summary: total kept, total deleted, space freed

---

## Testing Strategy (TDD)

Every package has tests written **before** implementation:

| Package | Approach |
|---|---|
| `config` | Table-driven tests for valid/invalid `.env` inputs |
| `rcon` | Interface + mock; integration test gated by `RCON_INTEGRATION=1` |
| `archive` | Creates real tar.gz of a temp dir, verifies contents |
| `retention` | Pure function — extensive table-driven tests covering edge cases (empty list, fewer backups than slots, ties) |
| `storage/nfs` | Interface tested via mock; real NFS tested by `NFS_INTEGRATION=1` |

The retention package is pure (no I/O), making it the most thoroughly tested component — it is the most complex logic in the system.

---

## Systemd Integration

`mc-backup.service` — runs the binary, loads env from `/etc/mc-backup/.env`.  
`mc-backup.timer` — triggers the service on a configurable schedule (default: hourly).

```ini
# mc-backup.timer
[Timer]
OnCalendar=hourly
Persistent=true
```

`Persistent=true` ensures that if the machine was off at the scheduled time, the backup runs immediately on next boot.

---

## Dependencies

| Module | Purpose |
|---|---|
| `github.com/joho/godotenv` | Load `.env` file |
| `github.com/gorcon/rcon` | RCON client |

No other external dependencies.
