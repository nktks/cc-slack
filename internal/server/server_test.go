package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

type mockSlack struct {
	lastChannel  string
	lastText     string
	lastThreadTS string
	returnTS     string
	returnErr    error
}

func (m *mockSlack) PostMessage(channel, text, threadTS string) (string, error) {
	m.lastChannel = channel
	m.lastText = text
	m.lastThreadTS = threadTS
	return m.returnTS, m.returnErr
}

func TestHandleHook(t *testing.T) {
	t.Run("posts message and stores thread_ts", func(t *testing.T) {
		dir := t.TempDir()
		transcript := filepath.Join(dir, "transcript.jsonl")
		os.WriteFile(transcript, []byte(
			`{"type":"user","message":{"role":"user","content":"hello"}}`+"\n"+
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]}}`+"\n",
		), 0644)

		mock := &mockSlack{returnTS: "111.222"}
		h := &Handler{
			Slack:   mock,
			Channel: "C123",
			Threads: NewThreadStore(),
		}

		body, _ := json.Marshal(map[string]string{
			"hook_event_name": "Stop",
			"session_id":      "sess-1",
			"transcript_path": transcript,
		})
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleHook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if mock.lastChannel != "C123" {
			t.Errorf("channel = %q, want %q", mock.lastChannel, "C123")
		}
		if mock.lastThreadTS != "" {
			t.Errorf("thread_ts = %q, want empty for first message", mock.lastThreadTS)
		}
		if ts := h.Threads.Get("sess-1"); ts != "111.222" {
			t.Errorf("stored thread_ts = %q, want %q", ts, "111.222")
		}
	})

	t.Run("uses existing thread_ts for same session", func(t *testing.T) {
		dir := t.TempDir()
		transcript := filepath.Join(dir, "transcript.jsonl")
		os.WriteFile(transcript, []byte(
			`{"type":"user","message":{"role":"user","content":"test"}}`+"\n",
		), 0644)

		mock := &mockSlack{returnTS: "333.444"}
		threads := NewThreadStore()
		threads.Set("sess-2", "111.222", "")

		h := &Handler{
			Slack:   mock,
			Channel: "C123",
			Threads: threads,
		}

		body, _ := json.Marshal(map[string]string{
			"hook_event_name": "PermissionRequest",
			"session_id":      "sess-2",
			"transcript_path": transcript,
		})
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleHook(w, req)

		if mock.lastThreadTS != "111.222" {
			t.Errorf("thread_ts = %q, want %q", mock.lastThreadTS, "111.222")
		}
		// thread_ts should not be overwritten
		if ts := h.Threads.Get("sess-2"); ts != "111.222" {
			t.Errorf("stored thread_ts = %q, want %q (should not change)", ts, "111.222")
		}
	})

	t.Run("mentions explicit user ID", func(t *testing.T) {
		dir := t.TempDir()
		transcript := filepath.Join(dir, "transcript.jsonl")
		os.WriteFile(transcript, []byte(
			`{"type":"user","message":{"role":"user","content":"test"}}`+"\n",
		), 0644)

		mock := &mockSlack{returnTS: "555.666"}
		h := &Handler{
			Slack:         mock,
			Channel:       "C123",
			UserID: "U9999",
			Threads:       NewThreadStore(),
		}

		body, _ := json.Marshal(map[string]string{
			"hook_event_name": "Stop",
			"session_id":      "sess-3",
			"transcript_path": transcript,
		})
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleHook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if !contains(mock.lastText, "<@U9999>") {
			t.Errorf("should mention U9999, got:\n%s", mock.lastText)
		}
	})

	t.Run("auto-mentions for DM channel", func(t *testing.T) {
		dir := t.TempDir()
		transcript := filepath.Join(dir, "transcript.jsonl")
		os.WriteFile(transcript, []byte(
			`{"type":"user","message":{"role":"user","content":"test"}}`+"\n",
		), 0644)

		mock := &mockSlack{returnTS: "777.888"}
		h := &Handler{
			Slack:   mock,
			Channel: "U5555",
			Threads: NewThreadStore(),
		}

		body, _ := json.Marshal(map[string]string{
			"hook_event_name": "Stop",
			"session_id":      "sess-4",
			"transcript_path": transcript,
		})
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleHook(w, req)

		if !contains(mock.lastText, "<@U5555>") {
			t.Errorf("should auto-mention DM user, got:\n%s", mock.lastText)
		}
	})

	t.Run("no mention for channel without explicit user", func(t *testing.T) {
		dir := t.TempDir()
		transcript := filepath.Join(dir, "transcript.jsonl")
		os.WriteFile(transcript, []byte(
			`{"type":"user","message":{"role":"user","content":"test"}}`+"\n",
		), 0644)

		mock := &mockSlack{returnTS: "999.000"}
		h := &Handler{
			Slack:   mock,
			Channel: "C123",
			Threads: NewThreadStore(),
		}

		body, _ := json.Marshal(map[string]string{
			"hook_event_name": "Stop",
			"session_id":      "sess-5",
			"transcript_path": transcript,
		})
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleHook(w, req)

		if contains(mock.lastText, "<@") {
			t.Errorf("should not mention anyone, got:\n%s", mock.lastText)
		}
	})

	t.Run("rejects non-POST", func(t *testing.T) {
		h := &Handler{Threads: NewThreadStore()}
		req := httptest.NewRequest("GET", "/hook", nil)
		w := httptest.NewRecorder()
		h.HandleHook(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		h := &Handler{Threads: NewThreadStore()}
		req := httptest.NewRequest("POST", "/hook", bytes.NewReader([]byte("not json")))
		w := httptest.NewRecorder()
		h.HandleHook(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}
