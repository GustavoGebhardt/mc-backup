package notify

import (
	"context"
	"log/slog"
)

type Notifier interface {
	Notify(ctx context.Context, msg string) error
}

func All(ctx context.Context, log *slog.Logger, notifiers []Notifier, msg string) {
	for _, n := range notifiers {
		if err := n.Notify(ctx, msg); err != nil {
			log.Warn("notifier failed", "error", err, "message", msg)
		}
	}
}
