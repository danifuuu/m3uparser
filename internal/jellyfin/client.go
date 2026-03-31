// Package jellyfin provides a lightweight HTTP client for the Jellyfin API.
// It uses API key authentication (no user/password login flow needed).
package jellyfin

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Client wraps Jellyfin API interactions using an API key.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New creates a Jellyfin API client. Returns nil if url or apiKey is empty.
func New(url, apiKey string) *Client {
	if url == "" || apiKey == "" {
		return nil
	}
	return &Client{
		BaseURL: url,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ping checks if the Jellyfin server is reachable.
func (c *Client) Ping(maxRetries int, interval time.Duration) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/System/Ping", nil)
		if err != nil {
			return fmt.Errorf("create ping request: %w", err)
		}
		c.setHeaders(req)

		resp, err := c.HTTPClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			slog.Info("jellyfin server is reachable")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		slog.Warn("ping attempt failed", "attempt", attempt, "max", maxRetries, "error", err)
		if attempt < maxRetries {
			time.Sleep(interval)
		}
	}

	return fmt.Errorf("jellyfin server unreachable after %d attempts", maxRetries)
}

// RefreshLibrary triggers a library scan.
func (c *Client) RefreshLibrary() error {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/Library/Refresh", nil)
	if err != nil {
		return fmt.Errorf("create refresh request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("library refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		slog.Info("library refresh triggered")
		return nil
	}

	return fmt.Errorf("library refresh failed with status %d", resp.StatusCode)
}

// RefreshGuide triggers the live TV guide refresh scheduled task.
func (c *Client) RefreshGuide() error {
	taskID := "bea9b218c97bbf98c5dc1303bdb9a0ca"
	url := fmt.Sprintf("%s/ScheduledTasks/Running/%s", c.BaseURL, taskID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create guide refresh request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("guide refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		slog.Info("guide refresh triggered")
		return nil
	}

	return fmt.Errorf("guide refresh failed with status %d", resp.StatusCode)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf(`MediaBrowser Token="%s"`, c.APIKey))
	req.Header.Set("Content-Type", "application/json")
}
