package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the interface for posting Slack messages.
type Client interface {
	PostMessage(channel, text, threadTS string) (ts string, err error)
}

type client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// New creates a Slack client with the given bot token.
func New(token string) Client {
	return &client{
		token:      token,
		httpClient: http.DefaultClient,
		baseURL:    "https://slack.com/api",
	}
}

// NewWithHTTPClient creates a Slack client with a custom http.Client and base URL.
// Intended for testing.
func NewWithHTTPClient(token string, httpClient *http.Client, baseURL string) Client {
	return &client{
		token:      token,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

type payload struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

type response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	TS    string `json:"ts"`
}

func (c *client) PostMessage(channel, text, threadTS string) (string, error) {
	p := payload{
		Channel:  channel,
		Text:     text,
		ThreadTS: threadTS,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode slack response: %v", err)
	}
	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}
	return result.TS, nil
}
