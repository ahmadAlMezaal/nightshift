package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxDiscordLen = 2000

var discordBoldRe = regexp.MustCompile(`\*([^*\n]+)\*`)

func toDiscordMarkdown(s string) string {
	return discordBoldRe.ReplaceAllString(s, "**$1**")
}

type Discord struct {
	Enabled    bool
	WebhookURL string
	HTTP       *http.Client
}

func NewDiscord(webhookURL string) *Discord {
	return &Discord{
		Enabled:    webhookURL != "",
		WebhookURL: webhookURL,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *Discord) Send(ctx context.Context, message string) {
	if d == nil || !d.Enabled {
		return
	}
	go func() {
		if err := d.post(context.Background(), message); err != nil {
			slog.Warn("notify: discord send failed", "err", err)
		}
	}()
}

func (d *Discord) SendSync(ctx context.Context, message string) error {
	if d == nil {
		return fmt.Errorf("discord client is nil")
	}
	if d.WebhookURL == "" {
		return fmt.Errorf("missing webhook URL")
	}
	return d.post(ctx, message)
}

func (d *Discord) post(ctx context.Context, message string) error {
	if d.HTTP == nil {
		return fmt.Errorf("discord HTTP client is nil")
	}
	message = toDiscordMarkdown(message)

	runes := []rune(message)
	if len(runes) > maxDiscordLen {
		message = string(runes[:maxDiscordLen-3]) + "..."
	}

	body := struct {
		Content         string `json:"content"`
		AllowedMentions struct {
			Parse []string `json:"parse"`
		} `json:"allowed_mentions"`
	}{Content: message}
	body.AllowedMentions.Parse = []string{}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("discord returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
