package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type RCONConfig struct {
	Host     string
	Port     int
	Password string
}

type NFSConfig struct {
	Host      string
	Share     string
	MountPoint string
	BackupDir  string
}

type RetentionConfig struct {
	Hourly  int
	Daily   int
	Weekly  int
	Monthly int
}

type TelegramConfig struct {
	Token  string
	ChatID string
}

type Config struct {
	MinecraftDir  string
	RCON          RCONConfig
	NFS           NFSConfig
	Retention     RetentionConfig
	BackupTmpDir  string
	BackupTimeout time.Duration
	Telegram      TelegramConfig
}

func Load(envFile string) (*Config, error) {
	env, err := godotenv.Read(envFile)
	if err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}

	get := func(key string) string { return env[key] }

	rconPort := 25575
	if raw := get("RCON_PORT"); raw != "" {
		rconPort, err = strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid RCON_PORT %q: %w", raw, err)
		}
	}

	cfg := &Config{
		MinecraftDir: get("MINECRAFT_DIR"),
		RCON: RCONConfig{
			Host:     stringDefault(get("RCON_HOST"), "localhost"),
			Port:     rconPort,
			Password: get("RCON_PASSWORD"),
		},
		NFS: NFSConfig{
			Host:       get("NFS_HOST"),
			Share:      get("NFS_SHARE"),
			MountPoint: get("NFS_MOUNT_POINT"),
			BackupDir:  stringDefault(get("NFS_BACKUP_DIR"), "backups"),
		},
		Retention: RetentionConfig{
			Hourly:  intDefault(get("RETENTION_HOURLY"), 24),
			Daily:   intDefault(get("RETENTION_DAILY"), 7),
			Weekly:  intDefault(get("RETENTION_WEEKLY"), 4),
			Monthly: intDefault(get("RETENTION_MONTHLY"), 12),
		},
		BackupTmpDir:  stringDefault(get("BACKUP_TMP_DIR"), "/tmp/mc-backup"),
		BackupTimeout: durationDefault(get("BACKUP_TIMEOUT"), 45*60) * time.Second,
		Telegram: TelegramConfig{
			Token:  get("TELEGRAM_BOT_TOKEN"),
			ChatID: get("TELEGRAM_CHAT_ID"),
		},
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	var errs []error
	if c.MinecraftDir == "" {
		errs = append(errs, errors.New("MINECRAFT_DIR is required"))
	}
	if c.RCON.Password == "" {
		errs = append(errs, errors.New("RCON_PASSWORD is required"))
	}
	if c.NFS.Host == "" {
		errs = append(errs, errors.New("NFS_HOST is required"))
	}
	if c.NFS.Share == "" {
		errs = append(errs, errors.New("NFS_SHARE is required"))
	}
	if c.NFS.MountPoint == "" {
		errs = append(errs, errors.New("NFS_MOUNT_POINT is required"))
	}
	return errors.Join(errs...)
}

func stringDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func intDefault(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func durationDefault(v string, defSeconds int) time.Duration {
	if v == "" {
		return time.Duration(defSeconds)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return time.Duration(defSeconds)
	}
	return time.Duration(n)
}
