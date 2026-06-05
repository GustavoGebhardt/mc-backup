package rcon_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gustavogebhardt/mc-backup/internal/rcon"
	rconnotify "github.com/gustavogebhardt/mc-backup/internal/notify/rcon"
)

type mockConn struct {
	executed []string
	execErr  error
}

func (c *mockConn) Execute(cmd string) (string, error) {
	c.executed = append(c.executed, cmd)
	return "", c.execErr
}

func (c *mockConn) Close() {}

type mockDialer struct {
	conn    *mockConn
	dialErr error
}

func (d *mockDialer) Dial() (rcon.Conn, error) {
	if d.dialErr != nil {
		return nil, d.dialErr
	}
	return d.conn, nil
}

func TestNotify_SendsSayCommand(t *testing.T) {
	conn := &mockConn{}
	notifier := rconnotify.New(&mockDialer{conn: conn})

	if err := notifier.Notify(context.Background(), "[Backup] Starting backup..."); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if len(conn.executed) != 1 || conn.executed[0] != "say [Backup] Starting backup..." {
		t.Errorf("expected say command, got %v", conn.executed)
	}
}

func TestNotify_ReturnsErrorOnDialFailure(t *testing.T) {
	notifier := rconnotify.New(&mockDialer{dialErr: errors.New("connection refused")})

	if err := notifier.Notify(context.Background(), "hello"); err == nil {
		t.Fatal("expected error on dial failure, got nil")
	}
}

func TestNotify_ReturnsErrorOnExecuteFailure(t *testing.T) {
	conn := &mockConn{execErr: errors.New("execute failed")}
	notifier := rconnotify.New(&mockDialer{conn: conn})

	if err := notifier.Notify(context.Background(), "hello"); err == nil {
		t.Fatal("expected error on execute failure, got nil")
	}
}

func TestNotify_ClosesConnectionAfterNotify(t *testing.T) {
	conn := &mockConn{}
	notifier := rconnotify.New(&mockDialer{conn: conn})

	_ = notifier.Notify(context.Background(), "hello")

	if len(conn.executed) == 0 {
		t.Error("expected command to be executed before close")
	}
}
