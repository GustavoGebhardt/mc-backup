package nfs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gustavogebhardt/mc-backup/internal/retention"
	"github.com/gustavogebhardt/mc-backup/internal/storage/nfs"
)

// fakeMounter records mount/unmount calls without touching the OS.
type fakeMounter struct {
	mounted    bool
	mountCalls int
	unmountCalls int
}

func (f *fakeMounter) Mount() error {
	f.mounted = true
	f.mountCalls++
	return nil
}

func (f *fakeMounter) Unmount() error {
	f.mounted = false
	f.unmountCalls++
	return nil
}

func (f *fakeMounter) IsMounted() bool {
	return f.mounted
}

func makeArchive(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "mc_backup_*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("fake archive content")
	f.Close()
	return f.Name()
}

func makeRemoteDir(t *testing.T, files map[string]time.Time) string {
	t.Helper()
	dir := t.TempDir()
	for name, modTime := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestUpload_CopiesFileToMountPoint(t *testing.T) {
	mounter := &fakeMounter{}
	archivePath := makeArchive(t)
	remoteDir := t.TempDir()

	store := nfs.New(remoteDir, mounter)
	remotePath, err := store.Upload(context.Background(), archivePath)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if _, err := os.Stat(remotePath); err != nil {
		t.Errorf("remote file not found: %v", err)
	}
	if filepath.Dir(remotePath) != remoteDir {
		t.Errorf("expected file in %q, got %q", remoteDir, remotePath)
	}
}

func TestUpload_MountsBeforeAndUnmountsAfter(t *testing.T) {
	mounter := &fakeMounter{}
	archivePath := makeArchive(t)

	store := nfs.New(t.TempDir(), mounter)
	_, err := store.Upload(context.Background(), archivePath)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if mounter.mountCalls != 1 {
		t.Errorf("expected 1 mount call, got %d", mounter.mountCalls)
	}
	if mounter.unmountCalls != 1 {
		t.Errorf("expected 1 unmount call, got %d", mounter.unmountCalls)
	}
	if mounter.mounted {
		t.Error("NFS should be unmounted after upload")
	}
}

func TestPrune_RemovesOldBackups(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)

	remoteDir := makeRemoteDir(t, map[string]time.Time{
		"mc_backup_" + now.Format("20060102_150405") + ".tar.gz": now,
		"mc_backup_" + old.Format("20060102_150405") + ".tar.gz": old,
	})

	mounter := &fakeMounter{}
	store := nfs.New(remoteDir, mounter)

	// Policy: keep 1 hourly only — the old one should be pruned
	policy := retention.Policy{Hourly: 1, Daily: 0, Weekly: 0, Monthly: 0}
	pruned, bytesFreed, err := store.Prune(context.Background(), policy)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if len(pruned) != 1 {
		t.Errorf("expected 1 pruned backup, got %d", len(pruned))
	}
	if bytesFreed == 0 {
		t.Error("bytesFreed should be > 0")
	}

	// Verify old file was actually deleted from disk
	oldName := "mc_backup_" + old.Format("20060102_150405") + ".tar.gz"
	if _, err := os.Stat(filepath.Join(remoteDir, oldName)); !os.IsNotExist(err) {
		t.Error("old backup file should have been deleted from disk")
	}
}

func TestPrune_MountsBeforeAndUnmountsAfter(t *testing.T) {
	mounter := &fakeMounter{}
	store := nfs.New(t.TempDir(), mounter)

	policy := retention.Policy{Hourly: 24, Daily: 7, Weekly: 4, Monthly: 12}
	_, _, err := store.Prune(context.Background(), policy)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if mounter.mountCalls != 1 {
		t.Errorf("expected 1 mount call, got %d", mounter.mountCalls)
	}
	if mounter.unmountCalls != 1 {
		t.Errorf("expected 1 unmount call, got %d", mounter.unmountCalls)
	}
}
