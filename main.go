package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type hookInput struct {
	HookEventName  string `json:"hook_event_name"`
	TranscriptPath string `json:"transcript_path"`
}

type transcriptEntry struct {
	Type    string           `json:"type"`
	Message *transcriptMessage `json:"message"`
}

type transcriptMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type slackPayload struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func main() {
	token := envWithFallback("CC_NOTIFY_SLACK_TOKEN", "SLACK_TOKEN")
	channel := envWithFallback("CC_NOTIFY_SLACK_CHANNEL", "SLACK_CHANNEL")
	if token == "" || channel == "" {
		log.Fatal("CC_NOTIFY_SLACK_TOKEN and CC_NOTIFY_SLACK_CHANNEL must be set")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		log.Fatalf("failed to parse hook input: %v", err)
	}

	prompt := lastUserPrompt(input.TranscriptPath)
	text := fmt.Sprintf("[%s] %q", input.HookEventName, truncate(prompt, 100))

	if err := postSlack(token, channel, text); err != nil {
		log.Fatalf("failed to send slack message: %v", err)
	}
}

func envWithFallback(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

func lastUserPrompt(path string) string {
	if path == "" {
		return "(unknown)"
	}
	f, err := os.Open(path)
	if err != nil {
		return "(unknown)"
	}
	defer f.Close()

	var last string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		var entry transcriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Type != "user" || entry.Message == nil || entry.Message.Role != "user" {
			continue
		}
		// only string content (arrays are tool_results)
		var s string
		if err := json.Unmarshal(entry.Message.Content, &s); err != nil {
			continue
		}
		last = s
	}
	if last == "" {
		return "(unknown)"
	}
	return last
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func postSlack(token, channel, text string) error {
	payload := slackPayload{
		Channel: channel,
		Text:    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode slack response: %v", err)
	}
	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}
