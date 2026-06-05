package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gustavogebhardt/mc-backup/internal/archive"
	"github.com/gustavogebhardt/mc-backup/internal/config"
	"github.com/gustavogebhardt/mc-backup/internal/notify"
	rconnotify "github.com/gustavogebhardt/mc-backup/internal/notify/rcon"
	telegramnotify "github.com/gustavogebhardt/mc-backup/internal/notify/telegram"
	"github.com/gustavogebhardt/mc-backup/internal/rcon"
	"github.com/gustavogebhardt/mc-backup/internal/retention"
	"github.com/gustavogebhardt/mc-backup/internal/storage/nfs"
)

var version = "dev"

func main() {
	envFile := flag.String("config", ".env", "path to .env config file")
	flag.Parse()

	log := newLogger()
	log.Info("mc-backup", "version", version)

	cfg, err := config.Load(*envFile)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	notifiers := buildNotifiers(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, cfg.BackupTimeout)
	defer cancel()

	if err := run(ctx, log, cfg, notifiers); err != nil {
		log.Error("backup failed", "error", err)
		notify.All(context.Background(), log, notifiers, "[Backup] Backup failed! Check the server logs.")
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, cfg *config.Config, notifiers []notify.Notifier) error {
	log.Info("backup started",
		"minecraft_dir", cfg.MinecraftDir,
		"nfs_host", cfg.NFS.Host,
		"timeout", cfg.BackupTimeout.String(),
		"retention", fmt.Sprintf("hourly=%d daily=%d weekly=%d monthly=%d",
			cfg.Retention.Hourly, cfg.Retention.Daily,
			cfg.Retention.Weekly, cfg.Retention.Monthly),
	)

	notify.All(ctx, log, notifiers, "[Backup] Starting backup...")

	log.Info("connecting to RCON", "host", cfg.RCON.Host, "port", cfg.RCON.Port)
	client, err := rcon.Dial(cfg.RCON.Host, cfg.RCON.Port, cfg.RCON.Password)
	if err != nil {
		return fmt.Errorf("rcon connect: %w", err)
	}
	defer client.Close()

	log.Info("sending save-off + save-all to server")
	if err := client.PrepareBackup(ctx); err != nil {
		return fmt.Errorf("preparing backup: %w", err)
	}

	backupPrepared := true
	defer func() {
		if !backupPrepared {
			return
		}
		if err := client.RestoreSave(context.Background()); err != nil {
			log.Warn("failed to send save-on", "error", err)
		}
	}()

	log.Info("waiting for I/O flush", "duration", "5s")
	select {
	case <-ctx.Done():
		return fmt.Errorf("cancelled during I/O flush: %w", ctx.Err())
	case <-time.After(5 * time.Second):
	}

	log.Info("creating archive", "source", cfg.MinecraftDir, "tmp_dir", cfg.BackupTmpDir)
	archiveStart := time.Now()
	result, err := archive.Create(cfg.MinecraftDir, cfg.BackupTmpDir)
	if err != nil {
		return fmt.Errorf("creating archive: %w", err)
	}
	defer os.Remove(result.Path)

	log.Info("archive created",
		"path", result.Path,
		"size_mb", fmt.Sprintf("%.1f", float64(result.Size)/1024/1024),
		"duration", time.Since(archiveStart).Round(time.Second),
	)

	log.Info("sending save-on to server")
	if err := client.RestoreSave(ctx); err != nil {
		log.Warn("failed to send save-on, server autosave may be disabled", "error", err)
	}
	backupPrepared = false

	mounter := nfs.NewOSMounter(cfg.NFS.Host, cfg.NFS.Share, cfg.NFS.MountPoint)
	store := nfs.New(filepath.Join(cfg.NFS.MountPoint, cfg.NFS.BackupDir), mounter)

	log.Info("uploading archive to NFS", "host", cfg.NFS.Host, "share", cfg.NFS.Share)
	uploadStart := time.Now()
	remotePath, err := store.Upload(ctx, result.Path)
	if err != nil {
		return fmt.Errorf("uploading to NFS: %w", err)
	}
	log.Info("upload complete",
		"remote_path", remotePath,
		"duration", time.Since(uploadStart).Round(time.Second),
	)

	policy := retention.Policy{
		Hourly:  cfg.Retention.Hourly,
		Daily:   cfg.Retention.Daily,
		Weekly:  cfg.Retention.Weekly,
		Monthly: cfg.Retention.Monthly,
	}

	log.Info("applying retention policy", "policy", fmt.Sprintf("%+v", policy))
	pruned, bytesFreed, err := store.Prune(ctx, policy)
	if err != nil {
		log.Warn("retention prune failed (backup was saved)", "error", err)
	} else {
		log.Info("retention applied",
			"pruned_count", len(pruned),
			"freed_mb", fmt.Sprintf("%.1f", float64(bytesFreed)/1024/1024),
		)
		for _, b := range pruned {
			log.Debug("pruned backup", "name", b.Name, "time", b.Time.Format(time.RFC3339))
		}
	}

	log.Info("backup completed successfully",
		"archive", filepath.Base(remotePath),
		"total_duration", time.Since(archiveStart).Round(time.Second),
	)

	notify.All(ctx, log, notifiers, "[Backup] Backup completed successfully!")

	return nil
}

func buildNotifiers(cfg *config.Config) []notify.Notifier {
	var notifiers []notify.Notifier

	dialer := rconnotify.NewOSDialer(cfg.RCON.Host, cfg.RCON.Port, cfg.RCON.Password)
	notifiers = append(notifiers, rconnotify.New(dialer))

	if cfg.Telegram.Token != "" && cfg.Telegram.ChatID != "" {
		notifiers = append(notifiers, telegramnotify.New(cfg.Telegram.Token, cfg.Telegram.ChatID))
	}

	return notifiers
}

func newLogger() *slog.Logger {
	if isTerminal() {
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
