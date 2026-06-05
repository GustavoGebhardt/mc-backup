package nfs

import (
	"fmt"
	"os"
	"os/exec"
)

type OSMounter struct {
	host       string
	share      string
	mountPoint string
}

func NewOSMounter(host, share, mountPoint string) *OSMounter {
	return &OSMounter{host: host, share: share, mountPoint: mountPoint}
}

func (m *OSMounter) Mount() error {
	if m.IsMounted() {
		return nil
	}
	if err := os.MkdirAll(m.mountPoint, 0750); err != nil {
		return fmt.Errorf("creating mount point %s: %w", m.mountPoint, err)
	}
	src := fmt.Sprintf("%s:%s", m.host, m.share)
	cmd := exec.Command("mount", "-t", "nfs", src, m.mountPoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount nfs %s -> %s: %w\n%s", src, m.mountPoint, err, out)
	}
	return nil
}

func (m *OSMounter) Unmount() error {
	if !m.IsMounted() {
		return nil
	}
	cmd := exec.Command("umount", m.mountPoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		cmd2 := exec.Command("umount", "-l", m.mountPoint)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("umount %s: %w\n%s\n%s", m.mountPoint, err, out, out2)
		}
	}
	return nil
}

func (m *OSMounter) IsMounted() bool {
	cmd := exec.Command("mountpoint", "-q", m.mountPoint)
	return cmd.Run() == nil
}
