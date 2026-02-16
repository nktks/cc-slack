package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Input struct {
	HookEventName  string          `json:"hook_event_name"`
	TranscriptPath string          `json:"transcript_path"`
	SessionID      string          `json:"session_id"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
}

type transcriptEntry struct {
	Type    string             `json:"type"`
	Message *transcriptMessage `json:"message"`
}

type transcriptMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ScanTranscript reads a JSONL transcript file and returns the last user
// prompt and the last assistant text response.
func ScanTranscript(path string) (prompt, response string) {
	if path == "" {
		return "(unknown)", ""
	}
	f, err := os.Open(path)
	if err != nil {
		return "(unknown)", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		var entry transcriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Message == nil {
			continue
		}

		switch {
		case entry.Type == "user" && entry.Message.Role == "user":
			// only string content (arrays are tool_results)
			var s string
			if err := json.Unmarshal(entry.Message.Content, &s); err == nil {
				prompt = s
			}
		case entry.Type == "assistant" && entry.Message.Role == "assistant":
			var blocks []contentBlock
			if err := json.Unmarshal(entry.Message.Content, &blocks); err == nil {
				for _, b := range blocks {
					if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
						response = b.Text
					}
				}
			}
		}
	}

	if prompt == "" {
		prompt = "(unknown)"
	}
	return prompt, response
}

// BuildMessage formats a Slack notification message from hook input.
// When isReply is true, the Prompt line is omitted (it's already in the parent thread).
func BuildMessage(input Input, prompt, response string, isReply bool) string {
	var b strings.Builder

	switch input.HookEventName {
	case "PermissionRequest":
		b.WriteString(fmt.Sprintf("[PermissionRequest] %s", input.ToolName))
		if detail := FormatToolInput(input.ToolName, input.ToolInput); detail != "" {
			b.WriteString(fmt.Sprintf("\n> %s", Truncate(detail, 200)))
		}
		if choices := PermissionChoices(input.ToolName); choices != "" {
			b.WriteString(fmt.Sprintf("\n> %s", strings.ReplaceAll(choices, "\n", "\n> ")))
		}
	default:
		b.WriteString(fmt.Sprintf("[%s]", input.HookEventName))
	}

	if !isReply {
		b.WriteString(fmt.Sprintf("\nPrompt: %q", Truncate(prompt, 100)))
	}

	// AskUserQuestion already shows the question and options; skip redundant response.
	if response != "" && !(input.HookEventName == "PermissionRequest" && input.ToolName == "AskUserQuestion") {
		b.WriteString(fmt.Sprintf("\nResponse: %s", strings.ReplaceAll(response, "\n", "\n> ")))
	}

	return b.String()
}

// FormatToolInput extracts the most relevant field from a tool's input.
func FormatToolInput(toolName string, toolInput json.RawMessage) string {
	if len(toolInput) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(toolInput, &m); err != nil {
		return ""
	}

	switch toolName {
	case "Bash":
		if cmd, ok := m["command"].(string); ok {
			return cmd
		}
	case "Write", "Edit", "Read":
		if fp, ok := m["file_path"].(string); ok {
			return fp
		}
	case "AskUserQuestion":
		return formatAskUserQuestion(m)
	}

	return ""
}

// formatAskUserQuestion extracts question text and option labels from AskUserQuestion input.
func formatAskUserQuestion(m map[string]any) string {
	questions, ok := m["questions"].([]any)
	if !ok || len(questions) == 0 {
		return ""
	}

	var parts []string
	for _, q := range questions {
		qm, ok := q.(map[string]any)
		if !ok {
			continue
		}
		text, _ := qm["question"].(string)
		if text == "" {
			continue
		}

		options, _ := qm["options"].([]any)
		var labels []string
		for _, o := range options {
			om, ok := o.(map[string]any)
			if !ok {
				continue
			}
			if label, ok := om["label"].(string); ok {
				labels = append(labels, label)
			}
		}

		// Claude Code UI always appends these fixed options.
		labels = append(labels, "Type something.", "Chat about this")

		text += "\n" + formatNumberedOptions(labels)
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n")
}

// formatNumberedOptions formats options as a numbered list.
func formatNumberedOptions(labels []string) string {
	var b strings.Builder
	for i, l := range labels {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(fmt.Sprintf("%d. %s", i+1, l))
	}
	return b.String()
}

// PermissionChoices returns the standard Claude Code permission dialog options
// for a given tool name. These are hardcoded in the Claude Code UI.
func PermissionChoices(toolName string) string {
	switch toolName {
	case "AskUserQuestion":
		return "" // options are already in the tool input
	case "Bash":
		return "1. Yes\n2. Yes, and don't ask again for this session\n3. No"
	default:
		return "1. Yes\n2. Yes, allow all edits during this session\n3. No"
	}
}

// Truncate shortens a string to n runes, replacing newlines with spaces.
func Truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
