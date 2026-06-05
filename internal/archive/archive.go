package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Result holds metadata about a created archive.
type Result struct {
	Path     string
	Size     int64
	Duration time.Duration
}

// Create compresses srcDir into a timestamped tar.gz file inside outDir.
func Create(srcDir, outDir string) (*Result, error) {
	if _, err := os.Stat(srcDir); err != nil {
		return nil, fmt.Errorf("source directory: %w", err)
	}

	if err := os.MkdirAll(outDir, 0750); err != nil {
		return nil, fmt.Errorf("creating tmp dir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102_150405")
	archiveName := fmt.Sprintf("mc_backup_%s.tar.gz", ts)
	archivePath := filepath.Join(outDir, archiveName)

	start := time.Now()
	if err := createTarGz(archivePath, srcDir); err != nil {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}

	return &Result{
		Path:     archivePath,
		Size:     info.Size(),
		Duration: time.Since(start),
	}, nil
}

func createTarGz(dest, src string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(filepath.Dir(src), path)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if d.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})
}
