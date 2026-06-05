# mc-backup

Automated Minecraft server backup tool written in Go.

## Project Standards

- All code, logs, comments, and documentation must be in **English**
- No comments unless the WHY is non-obvious
- TDD: write tests before implementation — every package has tests
- Do not add abstractions beyond what is needed

## Architecture

Plugin-style with a `Storage` interface (`internal/storage/storage.go`) that decouples the backup pipeline from the destination. Today's implementation is `NFSStorage`. Future implementations (S3, SFTP) only need to satisfy the interface.

```
cmd/mc-backup/main.go         # entrypoint and pipeline
internal/config/              # .env loading and validation
internal/rcon/                # RCON client (PrepareBackup, RestoreSave, Announce)
internal/archive/             # tar.gz creation
internal/retention/           # Proxmox-style generational retention algorithm (pure, no I/O)
internal/storage/storage.go   # Storage interface
internal/storage/nfs/         # NFSStorage + OSMounter
systemd/                      # install.sh, uninstall.sh, .service, .timer
```

## Backup Pipeline

1. Load config from `.env`
2. Connect via RCON → announce backup start to players
3. `save-off` + `save-all` + 5s wait for I/O flush
4. Create tar.gz of `MINECRAFT_DIR` in `BACKUP_TMP_DIR`
5. `save-on` → close RCON (server resumes autosave before slow NFS transfer)
6. Mount NFS → upload archive → unmount
7. Mount NFS → apply generational retention → unmount
8. Delete local tmp archive
9. Reconnect RCON → announce backup complete

## Running Tests

```bash
make test
```

## Building

```bash
make build
```

Produces `./mc-backup` binary compiled for Linux via Docker. No Go installation required.

## Deploying

```bash
make build
scp mc-backup user@server:~/mc-backup/
ssh user@server
cd mc-backup && sudo bash systemd/install.sh
sudo nano /etc/mc-backup/.env
```

## Config (.env)

See `.env.example` for all available options. Required fields:
- `MINECRAFT_DIR`
- `RCON_PASSWORD`
- `NFS_HOST`, `NFS_SHARE`, `NFS_MOUNT_POINT`

## Logs

Runs as JSON (via systemd). View with:

```bash
journalctl -u mc-backup -f
```

## Retention

Proxmox-style generational retention. Defaults: 24 hourly, 7 daily, 4 weekly, 12 monthly.
Estimated steady-state NAS usage: ~2.4 GB (based on 136 MB world / ~50 MB compressed).
