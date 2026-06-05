package nfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gustavogebhardt/mc-backup/internal/retention"
)

type Mounter interface {
	Mount() error
	Unmount() error
	IsMounted() bool
}

type NFSStorage struct {
	backupDir string
	mounter   Mounter
}

func New(backupDir string, mounter Mounter) *NFSStorage {
	return &NFSStorage{backupDir: backupDir, mounter: mounter}
}

func (s *NFSStorage) Upload(ctx context.Context, localPath string) (string, error) {
	if err := s.mounter.Mount(); err != nil {
		return "", fmt.Errorf("nfs mount: %w", err)
	}
	defer s.mounter.Unmount() //nolint:errcheck

	if err := os.MkdirAll(s.backupDir, 0750); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	dest := filepath.Join(s.backupDir, filepath.Base(localPath))
	if err := copyFile(localPath, dest); err != nil {
		return "", fmt.Errorf("copying archive to NFS: %w", err)
	}
	return dest, nil
}

func (s *NFSStorage) Prune(ctx context.Context, policy retention.Policy) ([]retention.Backup, int64, error) {
	if err := s.mounter.Mount(); err != nil {
		return nil, 0, fmt.Errorf("nfs mount: %w", err)
	}
	defer s.mounter.Unmount() //nolint:errcheck

	backups, err := listBackups(s.backupDir)
	if err != nil {
		return nil, 0, fmt.Errorf("listing backups: %w", err)
	}

	_, toDelete := retention.Apply(backups, policy)

	var bytesFreed int64
	for _, b := range toDelete {
		path := filepath.Join(s.backupDir, b.Name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, bytesFreed, fmt.Errorf("deleting %s: %w", b.Name, err)
		}
		bytesFreed += b.Size
	}

	return toDelete, bytesFreed, nil
}

func listBackups(dir string) ([]retention.Backup, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []retention.Backup
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "mc_backup_") || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, retention.Backup{
			Name: e.Name(),
			Size: info.Size(),
			Time: info.ModTime(),
		})
	}
	return backups, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
