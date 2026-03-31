// Package threadfin provides an HTTP API client for Threadfin.
// This replaces the Python WebSocket-based implementation with the simpler REST API.
package threadfin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client wraps Threadfin API interactions.
type Client struct {
	BaseURL    string
	User       string
	Pass       string
	HTTPClient *http.Client
	token      string
}

// New creates a Threadfin API client. Returns nil if required fields are missing.
func New(host, port, user, pass string) *Client {
	if user == "" || pass == "" {
		return nil
	}

	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "34400"
	}

	return &Client{
		BaseURL: fmt.Sprintf("http://%s:%s", host, port),
		User:    user,
		Pass:    pass,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with Threadfin and stores the session token.
func (c *Client) Login() error {
	payload := map[string]string{
		"cmd":      "login",
		"username": c.User,
		"password": c.Pass,
	}

	resp, err := c.post(payload)
	if err != nil {
		return fmt.Errorf("threadfin login: %w", err)
	}

	status, _ := resp["status"].(bool)
	if !status {
		return fmt.Errorf("threadfin login failed: check credentials and API access settings")
	}

	token, ok := resp["token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("threadfin login: no token in response")
	}

	c.token = token
	slog.Info("threadfin login successful")
	return nil
}

// UpdateM3U triggers an M3U file update in Threadfin.
func (c *Client) UpdateM3U() error {
	slog.Info("triggering threadfin M3U update")

	resp, err := c.post(map[string]string{
		"cmd":   "update.m3u",
		"token": c.token,
	})
	if err != nil {
		return fmt.Errorf("update m3u: %w", err)
	}

	// Update token for subsequent requests.
	if t, ok := resp["token"].(string); ok && t != "" {
		c.token = t
	}

	slog.Info("threadfin M3U update complete")
	return nil
}

// UpdateXMLTV triggers an XMLTV EPG update in Threadfin.
func (c *Client) UpdateXMLTV() error {
	slog.Info("triggering threadfin XMLTV update")

	resp, err := c.post(map[string]string{
		"cmd":   "update.xmltv",
		"token": c.token,
	})
	if err != nil {
		return fmt.Errorf("update xmltv: %w", err)
	}

	if t, ok := resp["token"].(string); ok && t != "" {
		c.token = t
	}

	slog.Info("threadfin XMLTV update complete")
	return nil
}

// UpdateXEPG triggers an xEPG update in Threadfin.
func (c *Client) UpdateXEPG() error {
	slog.Info("triggering threadfin xEPG update")

	_, err := c.post(map[string]string{
		"cmd":   "update.xepg",
		"token": c.token,
	})
	if err != nil {
		return fmt.Errorf("update xepg: %w", err)
	}

	slog.Info("threadfin xEPG update complete")
	return nil
}

// RunFullUpdate logs in and runs all three update steps with pauses between them.
func (c *Client) RunFullUpdate() error {
	if err := c.Login(); err != nil {
		return err
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"M3U", c.UpdateM3U},
		{"XMLTV", c.UpdateXMLTV},
		{"xEPG", c.UpdateXEPG},
	}

	for _, step := range steps {
		time.Sleep(10 * time.Second)
		if err := step.fn(); err != nil {
			slog.Error("threadfin update step failed", "step", step.name, "error", err)
			return err
		}
	}

	return nil
}

func (c *Client) post(payload map[string]string) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	url := c.BaseURL + "/api/"
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusForbidden ||
		resp.StatusCode == 423 { // Locked
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}
