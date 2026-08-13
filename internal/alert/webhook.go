package alert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookNotifier delivers alerts by POSTing the formatter's payload to a URL.
//
//	// Feishu custom bot webhook
//	feishu := alert.NewWebhookNotifier("https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
//	    alert.NewLarkFormatter())
//
//	// Gotify push
//	gotify := alert.NewWebhookNotifier("https://gotify.example.com/message",
//	    alert.NewGotifyFormatter("app-token"))
type WebhookNotifier struct {
	url       string
	formatter Formatter
	client    *http.Client
}

// WebhookOption configures a WebhookNotifier.
type WebhookOption func(*WebhookNotifier)

// WithTimeout sets the HTTP request timeout (default 5s).
func WithTimeout(d time.Duration) WebhookOption {
	return func(n *WebhookNotifier) { n.client.Timeout = d }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) WebhookOption {
	return func(n *WebhookNotifier) { n.client = c }
}

// NewWebhookNotifier creates a notifier that posts formatter output to url.
func NewWebhookNotifier(url string, f Formatter, opts ...WebhookOption) *WebhookNotifier {
	n := &WebhookNotifier{
		url:       url,
		formatter: f,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// Name returns the notifier name.
func (n *WebhookNotifier) Name() string { return "webhook:" + n.url }

// Notify POSTs the alert payload to the webhook URL.
func (n *WebhookNotifier) Notify(ctx context.Context, a Alert) error {
	a = a.normalize()

	body, err := n.formatter.Format(a)
	if err != nil {
		return fmt.Errorf("alert: format payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("alert: build request: %w", err)
	}
	req.Header.Set("Content-Type", n.formatter.ContentType())
	for k, v := range n.formatter.Headers() {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("alert: send to %s: %w", n.url, err)
	}
	defer resp.Body.Close()
	// Drain a small amount to allow connection reuse.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alert: webhook %s returned %s", n.url, resp.Status)
	}
	return nil
}
