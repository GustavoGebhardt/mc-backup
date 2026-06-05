package storage

import (
	"context"

	"github.com/gustavogebhardt/mc-backup/internal/retention"
)

// Storage is the interface for backup destinations.
// Implementations must be safe to call sequentially (Upload then Prune).
type Storage interface {
	// Upload copies the archive at localPath to the destination.
	Upload(ctx context.Context, localPath string) (remotePath string, err error)

	// Prune removes old backups according to the retention policy.
	// It returns the list of deleted backups and total bytes freed.
	Prune(ctx context.Context, policy retention.Policy) (pruned []retention.Backup, bytesFreed int64, err error)
}
