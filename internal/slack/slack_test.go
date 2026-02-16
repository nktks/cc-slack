package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostMessage(t *testing.T) {
	t.Run("successful post returns ts", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
				t.Errorf("Authorization = %q, want %q", got, "Bearer xoxb-test")
			}
			if got := r.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", got)
			}

			body, _ := io.ReadAll(r.Body)
			var p payload
			json.Unmarshal(body, &p)
			if p.Channel != "C123" {
				t.Errorf("channel = %q, want %q", p.Channel, "C123")
			}
			if p.Text != "hello" {
				t.Errorf("text = %q, want %q", p.Text, "hello")
			}
			if p.ThreadTS != "" {
				t.Errorf("thread_ts = %q, want empty", p.ThreadTS)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response{OK: true, TS: "1234567890.123456"})
		}))
		defer srv.Close()

		c := NewWithHTTPClient("xoxb-test", srv.Client(), srv.URL)
		ts, err := c.PostMessage("C123", "hello", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts != "1234567890.123456" {
			t.Errorf("ts = %q, want %q", ts, "1234567890.123456")
		}
	})

	t.Run("sends thread_ts when provided", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var p payload
			json.Unmarshal(body, &p)
			if p.ThreadTS != "1111111111.111111" {
				t.Errorf("thread_ts = %q, want %q", p.ThreadTS, "1111111111.111111")
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response{OK: true, TS: "2222222222.222222"})
		}))
		defer srv.Close()

		c := NewWithHTTPClient("xoxb-test", srv.Client(), srv.URL)
		_, err := c.PostMessage("C123", "reply", "1111111111.111111")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on slack API error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response{OK: false, Error: "channel_not_found"})
		}))
		defer srv.Close()

		c := NewWithHTTPClient("xoxb-test", srv.Client(), srv.URL)
		_, err := c.PostMessage("C123", "hello", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := NewWithHTTPClient("xoxb-test", srv.Client(), srv.URL)
		_, err := c.PostMessage("C123", "hello", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
