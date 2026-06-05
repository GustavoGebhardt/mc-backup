#!/usr/bin/env bash
# Run as root on the Minecraft server to remove mc-backup.
set -euo pipefail

echo "==> Stopping and disabling timer"
systemctl disable --now mc-backup.timer 2>/dev/null || true
systemctl stop mc-backup.service 2>/dev/null || true

echo "==> Removing systemd units"
rm -f /etc/systemd/system/mc-backup.service
rm -f /etc/systemd/system/mc-backup.timer

echo "==> Reloading systemd"
systemctl daemon-reload
systemctl reset-failed 2>/dev/null || true

echo "==> Removing binary"
rm -f /usr/local/bin/mc-backup

echo "==> Removing config"
rm -rf /etc/mc-backup

echo ""
echo "Done. NAS backups were NOT removed."
echo "To also remove them: delete the backups/ folder on your NAS manually."
