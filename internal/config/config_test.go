package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gustavogebhardt/mc-backup/internal/config"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestLoad_ValidConfig(t *testing.T) {
	env := writeEnv(t, `
MINECRAFT_DIR=/srv/minecraft/world
RCON_HOST=localhost
RCON_PORT=25575
RCON_PASSWORD=secret
NFS_HOST=192.168.1.10
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup
RETENTION_HOURLY=24
RETENTION_DAILY=7
RETENTION_WEEKLY=4
RETENTION_MONTHLY=12
BACKUP_TMP_DIR=/tmp/mc-backup
`)
	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.MinecraftDir != "/srv/minecraft/world" {
		t.Errorf("MinecraftDir: got %q", cfg.MinecraftDir)
	}
	if cfg.RCON.Host != "localhost" {
		t.Errorf("RCON.Host: got %q", cfg.RCON.Host)
	}
	if cfg.RCON.Port != 25575 {
		t.Errorf("RCON.Port: got %d", cfg.RCON.Port)
	}
	if cfg.RCON.Password != "secret" {
		t.Errorf("RCON.Password: got %q", cfg.RCON.Password)
	}
	if cfg.NFS.Host != "192.168.1.10" {
		t.Errorf("NFS.Host: got %q", cfg.NFS.Host)
	}
	if cfg.NFS.Share != "/volume1/mc" {
		t.Errorf("NFS.Share: got %q", cfg.NFS.Share)
	}
	if cfg.NFS.MountPoint != "/mnt/mc-backup" {
		t.Errorf("NFS.MountPoint: got %q", cfg.NFS.MountPoint)
	}
	if cfg.Retention.Hourly != 24 {
		t.Errorf("Retention.Hourly: got %d", cfg.Retention.Hourly)
	}
	if cfg.Retention.Daily != 7 {
		t.Errorf("Retention.Daily: got %d", cfg.Retention.Daily)
	}
	if cfg.Retention.Weekly != 4 {
		t.Errorf("Retention.Weekly: got %d", cfg.Retention.Weekly)
	}
	if cfg.Retention.Monthly != 12 {
		t.Errorf("Retention.Monthly: got %d", cfg.Retention.Monthly)
	}
	if cfg.BackupTmpDir != "/tmp/mc-backup" {
		t.Errorf("BackupTmpDir: got %q", cfg.BackupTmpDir)
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing MINECRAFT_DIR",
			content: `RCON_HOST=localhost
RCON_PORT=25575
RCON_PASSWORD=secret
NFS_HOST=192.168.1.10
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup`,
		},
		{
			name: "missing RCON_PASSWORD",
			content: `MINECRAFT_DIR=/srv/minecraft/world
RCON_HOST=localhost
RCON_PORT=25575
NFS_HOST=192.168.1.10
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup`,
		},
		{
			name: "missing NFS_HOST",
			content: `MINECRAFT_DIR=/srv/minecraft/world
RCON_HOST=localhost
RCON_PORT=25575
RCON_PASSWORD=secret
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := writeEnv(t, tc.content)
			_, err := config.Load(env)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestLoad_InvalidRCONPort(t *testing.T) {
	env := writeEnv(t, `
MINECRAFT_DIR=/srv/minecraft/world
RCON_HOST=localhost
RCON_PORT=not-a-number
RCON_PASSWORD=secret
NFS_HOST=192.168.1.10
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup
`)
	_, err := config.Load(env)
	if err == nil {
		t.Fatal("expected error for invalid RCON_PORT, got nil")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	env := writeEnv(t, `
MINECRAFT_DIR=/srv/minecraft/world
RCON_HOST=localhost
RCON_PORT=25575
RCON_PASSWORD=secret
NFS_HOST=192.168.1.10
NFS_SHARE=/volume1/mc
NFS_MOUNT_POINT=/mnt/mc-backup
`)
	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Retention.Hourly != 24 {
		t.Errorf("default Retention.Hourly: got %d, want 24", cfg.Retention.Hourly)
	}
	if cfg.Retention.Daily != 7 {
		t.Errorf("default Retention.Daily: got %d, want 7", cfg.Retention.Daily)
	}
	if cfg.Retention.Weekly != 4 {
		t.Errorf("default Retention.Weekly: got %d, want 4", cfg.Retention.Weekly)
	}
	if cfg.Retention.Monthly != 12 {
		t.Errorf("default Retention.Monthly: got %d, want 12", cfg.Retention.Monthly)
	}
	if cfg.BackupTmpDir != "/tmp/mc-backup" {
		t.Errorf("default BackupTmpDir: got %q, want /tmp/mc-backup", cfg.BackupTmpDir)
	}
	if cfg.RCON.Host != "localhost" {
		t.Errorf("default RCON.Host: got %q, want localhost", cfg.RCON.Host)
	}
}

func TestLoad_EnvFileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for missing .env file, got nil")
	}
}
