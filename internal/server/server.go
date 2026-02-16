package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nktks/cc-slack/internal/hook"
	"github.com/nktks/cc-slack/internal/slack"
)

// Handler handles HTTP requests from Claude Code hooks.
type Handler struct {
	Slack         slack.Client
	Channel       string
	MentionUserID string
	Threads       *ThreadStore
}

// HandleHook processes a hook event sent via POST.
func (h *Handler) HandleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input hook.Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Wait briefly for the transcript file to be fully written.
	time.Sleep(500 * time.Millisecond)

	prompt, response := hook.ScanTranscript(input.TranscriptPath)
	threadTS := h.Threads.Get(input.SessionID)
	isReply := threadTS != ""
	text := hook.BuildMessage(input, prompt, response, isReply)
	if uid := h.mentionTarget(); uid != "" {
		text = fmt.Sprintf("<@%s> %s", uid, text)
	}

	responseTS, err := h.Slack.PostMessage(h.Channel, text, threadTS)
	if err != nil {
		log.Printf("failed to send slack message: %v", err)
		http.Error(w, "slack post failed", http.StatusInternalServerError)
		return
	}

	if input.SessionID != "" && threadTS == "" && responseTS != "" {
		h.Threads.Set(input.SessionID, responseTS)
	}

	w.WriteHeader(http.StatusOK)
}

// mentionTarget returns the user ID to mention, or empty string if none.
func (h *Handler) mentionTarget() string {
	if h.MentionUserID != "" {
		return h.MentionUserID
	}
	if strings.HasPrefix(h.Channel, "U") {
		return h.Channel
	}
	return ""
}
