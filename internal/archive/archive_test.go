package archive_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gustavogebhardt/mc-backup/internal/archive"
)

func makeWorldDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"level.dat":              "leveldata",
		"region/r.0.0.mca":      "regiondata",
		"playerdata/abc.dat":     "playerdata",
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func listTarContents(t *testing.T, archivePath string) []string {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

func TestCreate_ProducesValidTarGz(t *testing.T) {
	worldDir := makeWorldDir(t)
	outDir := t.TempDir()

	result, err := archive.Create(worldDir, outDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("archive file not found: %v", err)
	}
	if !strings.HasSuffix(result.Path, ".tar.gz") {
		t.Errorf("expected .tar.gz extension, got %q", result.Path)
	}
}

func TestCreate_ArchiveContainsAllFiles(t *testing.T) {
	worldDir := makeWorldDir(t)
	outDir := t.TempDir()

	result, err := archive.Create(worldDir, outDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	entries := listTarContents(t, result.Path)
	entrySet := map[string]bool{}
	for _, e := range entries {
		entrySet[e] = true
	}

	// Walk the source and verify every file appears in the archive
	err = filepath.WalkDir(worldDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(worldDir, path)
		found := false
		for e := range entrySet {
			if strings.HasSuffix(e, rel) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("file %q not found in archive (entries: %v)", rel, entries)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreate_ReturnsCorrectSize(t *testing.T) {
	worldDir := makeWorldDir(t)
	outDir := t.TempDir()

	result, err := archive.Create(worldDir, outDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Size != info.Size() {
		t.Errorf("Size: reported %d, actual %d", result.Size, info.Size())
	}
	if result.Size == 0 {
		t.Error("archive size should be greater than 0")
	}
}

func TestCreate_NameContainsTimestamp(t *testing.T) {
	worldDir := makeWorldDir(t)
	outDir := t.TempDir()

	result, err := archive.Create(worldDir, outDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	base := filepath.Base(result.Path)
	if !strings.HasPrefix(base, "mc_backup_") {
		t.Errorf("expected name to start with mc_backup_, got %q", base)
	}
}

func TestCreate_SourceDirNotFound(t *testing.T) {
	_, err := archive.Create("/nonexistent/world", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing source dir, got nil")
	}
}
