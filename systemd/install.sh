#!/usr/bin/env bash
# Run as root from the project root on the Minecraft server.
# Usage: sudo bash systemd/install.sh
set -euo pipefail

BINARY=mc-backup
CONFIG_DIR=/etc/mc-backup

echo "==> Installing binary to /usr/local/bin/$BINARY"
install -m 755 "$BINARY" /usr/local/bin/

echo "==> Creating config directory $CONFIG_DIR"
install -d -m 750 "$CONFIG_DIR"

if [[ ! -f "$CONFIG_DIR/.env" ]]; then
    echo "==> Copying .env.example → $CONFIG_DIR/.env"
    install -m 600 .env.example "$CONFIG_DIR/.env"
    echo "    !! Edit $CONFIG_DIR/.env before starting the timer !!"
fi

echo "==> Installing systemd units"
install -m 644 systemd/mc-backup.service /etc/systemd/system/
install -m 644 systemd/mc-backup.timer   /etc/systemd/system/

echo "==> Reloading systemd"
systemctl daemon-reload

echo "==> Enabling and starting timer"
systemctl enable --now mc-backup.timer

echo ""
echo "Done. Check status with:"
echo "  systemctl status mc-backup.timer"
echo "  journalctl -u mc-backup -f"
