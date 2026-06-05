package rcon_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gustavogebhardt/mc-backup/internal/rcon"
)

type mockConn struct {
	responses map[string]string
	closed    bool
	execErr   error
}

func (m *mockConn) Execute(cmd string) (string, error) {
	if m.execErr != nil {
		return "", m.execErr
	}
	return m.responses[cmd], nil
}

func (m *mockConn) Close() {
	m.closed = true
}

type recordingConn struct {
	inner    rcon.Conn
	recorded *[]string
}

func (r *recordingConn) Execute(cmd string) (string, error) {
	*r.recorded = append(*r.recorded, cmd)
	return r.inner.Execute(cmd)
}

func (r *recordingConn) Close() {
	r.inner.Close()
}

func TestClient_PrepareBackup_SendsCommandsInOrder(t *testing.T) {
	var cmds []string
	conn := &mockConn{responses: map[string]string{
		"save-off": "Automatic saving is now disabled",
		"save-all": "Saved the game",
	}}
	client := rcon.NewClientWithConn(&recordingConn{inner: conn, recorded: &cmds})

	if err := client.PrepareBackup(context.Background()); err != nil {
		t.Fatalf("PrepareBackup: %v", err)
	}

	want := []string{"save-off", "save-all"}
	if len(cmds) != len(want) {
		t.Fatalf("expected %d commands, got %d: %v", len(want), len(cmds), cmds)
	}
	for i, cmd := range want {
		if cmds[i] != cmd {
			t.Errorf("command[%d]: got %q, want %q", i, cmds[i], cmd)
		}
	}
}

func TestClient_RestoreSave_SendsSaveOn(t *testing.T) {
	var cmds []string
	conn := &mockConn{responses: map[string]string{"save-on": "Automatic saving is now enabled"}}
	client := rcon.NewClientWithConn(&recordingConn{inner: conn, recorded: &cmds})

	if err := client.RestoreSave(context.Background()); err != nil {
		t.Fatalf("RestoreSave: %v", err)
	}

	if len(cmds) != 1 || cmds[0] != "save-on" {
		t.Errorf("expected [save-on], got %v", cmds)
	}
}

func TestClient_PrepareBackup_ReturnsErrorOnFailure(t *testing.T) {
	conn := &mockConn{execErr: errors.New("connection refused")}
	client := rcon.NewClientWithConn(conn)

	if err := client.PrepareBackup(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_Close_ClosesConnection(t *testing.T) {
	conn := &mockConn{}
	client := rcon.NewClientWithConn(conn)
	client.Close()

	if !conn.closed {
		t.Error("expected connection to be closed")
	}
}
