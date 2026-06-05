package telegram_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gustavogebhardt/mc-backup/internal/notify/telegram"
)

func TestNotify_SendsMessageToTelegram(t *testing.T) {
	var received struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	notifier := telegram.NewWithBaseURL("token123", "chat456", srv.URL)

	if err := notifier.Notify(context.Background(), "[Backup] Starting backup..."); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if received.ChatID != "chat456" {
		t.Errorf("chat_id: got %q, want %q", received.ChatID, "chat456")
	}
	if !strings.Contains(received.Text, "[Backup] Starting backup...") {
		t.Errorf("text: got %q, expected to contain the message", received.Text)
	}
}

func TestNotify_ReturnsErrorOnNonOKResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer srv.Close()

	notifier := telegram.NewWithBaseURL("badtoken", "chat456", srv.URL)

	err := notifier.Notify(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error on non-OK response, got nil")
	}
}
