# mc-backup

Automated Minecraft server backup tool. Connects to the server via RCON to safely flush the world to disk, compresses it to a tar.gz archive, and uploads it to a NAS via NFS with Proxmox-style generational retention.

## Requirements

- Docker (for building)
- NAS mounted via NFS
- Minecraft server with RCON enabled

## Quick Start

**1. Enable RCON in `server.properties`:**

```properties
rcon.enable=true
rcon.port=25575
rcon.password=yourpassword
```

**2. Build:**

```bash
make build
```

**3. Configure:**

```bash
cp .env.example .env
nano .env
```

**4. Install:**

```bash
sudo bash systemd/install.sh
```

**5. Check:**

```bash
systemctl status mc-backup.timer
journalctl -u mc-backup -f
```

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `MINECRAFT_DIR` | yes | — | Path to the world directory |
| `RCON_HOST` | no | `localhost` | RCON host |
| `RCON_PORT` | no | `25575` | RCON port |
| `RCON_PASSWORD` | yes | — | RCON password |
| `NFS_HOST` | yes | — | NAS IP or hostname |
| `NFS_SHARE` | yes | — | NFS share path on the NAS |
| `NFS_MOUNT_POINT` | yes | — | Local mount point |
| `RETENTION_HOURLY` | no | `24` | Hourly backups to keep |
| `RETENTION_DAILY` | no | `7` | Daily backups to keep |
| `RETENTION_WEEKLY` | no | `4` | Weekly backups to keep |
| `RETENTION_MONTHLY` | no | `12` | Monthly backups to keep |
| `BACKUP_TMP_DIR` | no | `/tmp/mc-backup` | Temporary local directory |

## Retention

Uses Proxmox-style generational retention. After each backup, the oldest archives that don't fit any generation slot are deleted. With default settings and a ~50 MB compressed world, steady-state NAS usage is ~2.4 GB.

## Uninstall

```bash
sudo bash systemd/uninstall.sh
```

Config and NAS backups are not removed automatically.
