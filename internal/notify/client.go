// Package notify sends a structured notification to the homelab Telegram bot
// webhook after each m3uparser run.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MediaItem represents a single added or removed media title.
type MediaItem struct {
	Title string `json:"title"`
	Type  string `json:"type"` // "movie" | "tv" | "unsorted"
}

// Payload is the JSON body POSTed to the Telegram bot webhook.
type Payload struct {
	Added   []MediaItem `json:"added"`
	Removed []MediaItem `json:"removed"`
	Errors  int         `json:"errors"`
}

// Client posts notifications to the bot webhook.
type Client struct {
	url    string
	secret string
}

// New creates a Client. secret may be empty.
func New(webhookURL, secret string) *Client {
	return &Client{url: webhookURL, secret: secret}
}

// Send POSTs the payload to the webhook.  Non-2xx responses are returned as errors.
func (c *Client) Send(p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("notify: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}

	hc := &http.Client{Timeout: 10 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("notify: POST %s: %w", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notify: webhook returned %d", resp.StatusCode)
	}

	return nil
}
