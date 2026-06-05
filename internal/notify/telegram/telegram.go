package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramNotifier struct {
	token   string
	chatID  string
	baseURL string
}

func New(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{token: token, chatID: chatID, baseURL: "https://api.telegram.org"}
}

func NewWithBaseURL(token, chatID, baseURL string) *TelegramNotifier {
	return &TelegramNotifier{token: token, chatID: chatID, baseURL: baseURL}
}

func (n *TelegramNotifier) Notify(ctx context.Context, msg string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id": n.chatID,
		"text":    msg,
	})
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", n.baseURL, n.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return fmt.Errorf("telegram API error %d: %s", resp.StatusCode, body.Description)
	}
	return nil
}
