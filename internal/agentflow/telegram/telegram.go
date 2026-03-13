package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiBase = "https://api.telegram.org"

type Notifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends a text message to the configured chat (channel/group).
// If bot token or chat ID is empty, Send is a no-op and returns nil.
func (n *Notifier) Send(ctx context.Context, text string) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}
	u := fmt.Sprintf("%s/bot%s/sendMessage", apiBase, n.botToken)
	form := url.Values{}
	form.Set("chat_id", n.chatID)
	form.Set("text", text)
	form.Set("parse_mode", "HTML")
	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(body))
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage: %s %s", resp.Status, string(body))
	}
	return nil
}
