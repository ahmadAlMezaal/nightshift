package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Slack struct {
	Enabled    bool
	WebhookURL string
	HTTP       *http.Client
}

func NewSlack(webhookURL string) *Slack {
	return &Slack{
		Enabled:    webhookURL != "",
		WebhookURL: webhookURL,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Slack) Send(ctx context.Context, message string) {
	if s == nil || !s.Enabled {
		return
	}
	go func() {
		if err := s.post(context.Background(), message); err != nil {
			slog.Warn("notify: slack send failed", "err", err)
		}
	}()
}

func (s *Slack) SendSync(ctx context.Context, message string) error {
	if s == nil {
		return fmt.Errorf("slack client is nil")
	}
	if s.WebhookURL == "" {
		return fmt.Errorf("missing webhook URL")
	}
	return s.post(ctx, message)
}

func (s *Slack) post(ctx context.Context, message string) error {
	if s.HTTP == nil {
		return fmt.Errorf("slack HTTP client is nil")
	}
	payload, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: message})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("slack returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
