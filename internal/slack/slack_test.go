package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	slackapi "github.com/slack-go/slack"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	api := slackapi.New("xoxb-test", slackapi.OptionAPIURL(srv.URL+"/"))
	return &client{api: api}
}

func TestPostMessage(t *testing.T) {
	t.Run("successful post returns ts", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("method = %s, want POST", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			params, _ := url.ParseQuery(string(body))
			if params.Get("channel") != "C123" {
				t.Errorf("channel = %q, want %q", params.Get("channel"), "C123")
			}
			if params.Get("text") != "hello" {
				t.Errorf("text = %q, want %q", params.Get("text"), "hello")
			}
			if params.Get("thread_ts") != "" {
				t.Errorf("thread_ts = %q, want empty", params.Get("thread_ts"))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"ts": "1234567890.123456",
			})
		})

		ts, err := c.PostMessage("C123", "hello", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts != "1234567890.123456" {
			t.Errorf("ts = %q, want %q", ts, "1234567890.123456")
		}
	})

	t.Run("sends thread_ts when provided", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			params, _ := url.ParseQuery(string(body))
			if params.Get("thread_ts") != "1111111111.111111" {
				t.Errorf("thread_ts = %q, want %q", params.Get("thread_ts"), "1111111111.111111")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"ts": "2222222222.222222",
			})
		})

		_, err := c.PostMessage("C123", "reply", "1111111111.111111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on slack API error", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok":    false,
				"error": "channel_not_found",
			})
		})

		_, err := c.PostMessage("C123", "hello", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
