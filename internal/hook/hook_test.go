package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScanTranscript(t *testing.T) {
	t.Run("extracts last user prompt and assistant response", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "transcript.jsonl")
		lines := []string{
			`{"type":"user","message":{"role":"user","content":"first prompt"}}`,
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"first response"}]}}`,
			`{"type":"user","message":{"role":"user","content":"second prompt"}}`,
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"second response"}]}}`,
		}
		writeLines(t, path, lines)

		prompt, response := ScanTranscript(path)
		if prompt != "second prompt" {
			t.Errorf("prompt = %q, want %q", prompt, "second prompt")
		}
		if response != "second response" {
			t.Errorf("response = %q, want %q", response, "second response")
		}
	})

	t.Run("skips tool_result arrays in user messages", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "transcript.jsonl")
		lines := []string{
			`{"type":"user","message":{"role":"user","content":"real prompt"}}`,
			`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"result"}]}}`,
		}
		writeLines(t, path, lines)

		prompt, _ := ScanTranscript(path)
		if prompt != "real prompt" {
			t.Errorf("prompt = %q, want %q", prompt, "real prompt")
		}
	})

	t.Run("returns unknown for empty path", func(t *testing.T) {
		prompt, response := ScanTranscript("")
		if prompt != "(unknown)" {
			t.Errorf("prompt = %q, want %q", prompt, "(unknown)")
		}
		if response != "" {
			t.Errorf("response = %q, want empty", response)
		}
	})

	t.Run("returns unknown for missing file", func(t *testing.T) {
		prompt, response := ScanTranscript("/nonexistent/file.jsonl")
		if prompt != "(unknown)" {
			t.Errorf("prompt = %q, want %q", prompt, "(unknown)")
		}
		if response != "" {
			t.Errorf("response = %q, want empty", response)
		}
	})
}

func TestBuildMessage(t *testing.T) {
	t.Run("Stop event with prompt and response", func(t *testing.T) {
		input := Input{HookEventName: "Stop"}
		msg := BuildMessage(input, "my prompt", "my response", false)
		if !contains(msg, "Prompt:") {
			t.Errorf("should contain Prompt, got:\n%s", msg)
		}
		if !contains(msg, "Response: my response") {
			t.Errorf("should contain Response, got:\n%s", msg)
		}
	})

	t.Run("reply omits prompt", func(t *testing.T) {
		input := Input{HookEventName: "Stop"}
		msg := BuildMessage(input, "my prompt", "my response", true)
		if contains(msg, "Prompt:") {
			t.Errorf("reply should not contain Prompt, got:\n%s", msg)
		}
		if !contains(msg, "Response: my response") {
			t.Errorf("reply should contain Response, got:\n%s", msg)
		}
	})

	t.Run("PermissionRequest Bash includes choices", func(t *testing.T) {
		input := Input{
			HookEventName: "PermissionRequest",
			ToolName:      "Bash",
			ToolInput:     json.RawMessage(`{"command":"npm test"}`),
		}
		msg := BuildMessage(input, "my prompt", "some response", false)
		if !contains(msg, "[PermissionRequest] Bash") {
			t.Errorf("should contain tool name, got:\n%s", msg)
		}
		if !contains(msg, "> npm test") {
			t.Errorf("should contain tool input, got:\n%s", msg)
		}
		if !contains(msg, "> 1. Yes") {
			t.Errorf("should contain permission choices, got:\n%s", msg)
		}
		if !contains(msg, "don't ask again") {
			t.Errorf("Bash should have 'don't ask again' choice, got:\n%s", msg)
		}
	})

	t.Run("PermissionRequest Write includes choices", func(t *testing.T) {
		input := Input{
			HookEventName: "PermissionRequest",
			ToolName:      "Write",
			ToolInput:     json.RawMessage(`{"file_path":"/tmp/foo.go","content":"x"}`),
		}
		msg := BuildMessage(input, "my prompt", "", false)
		if !contains(msg, "> /tmp/foo.go") {
			t.Errorf("should contain file path, got:\n%s", msg)
		}
		if !contains(msg, "allow all edits") {
			t.Errorf("Write should have 'allow all edits' choice, got:\n%s", msg)
		}
	})

	t.Run("PermissionRequest AskUserQuestion no extra choices and no response", func(t *testing.T) {
		input := Input{
			HookEventName: "PermissionRequest",
			ToolName:      "AskUserQuestion",
			ToolInput:     json.RawMessage(`{"questions":[{"question":"Pick one","options":[{"label":"A"}]}]}`),
		}
		msg := BuildMessage(input, "my prompt", "some preceding text", false)
		if contains(msg, "1. Yes") {
			t.Errorf("AskUserQuestion should not have Yes/No choices, got:\n%s", msg)
		}
		if !contains(msg, "Chat about this") {
			t.Errorf("should contain Chat about this, got:\n%s", msg)
		}
		if contains(msg, "Response:") {
			t.Errorf("AskUserQuestion should not show Response, got:\n%s", msg)
		}
	})

	t.Run("TaskCompleted event", func(t *testing.T) {
		input := Input{HookEventName: "TaskCompleted"}
		msg := BuildMessage(input, "do the thing", "done", false)
		if !contains(msg, "[TaskCompleted]") {
			t.Errorf("should contain event name, got:\n%s", msg)
		}
		if !contains(msg, "Response:") {
			t.Errorf("should contain response, got:\n%s", msg)
		}
	})
}

func TestFormatToolInput(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    string
		want     string
	}{
		{"Bash command", "Bash", `{"command":"go test ./..."}`, "go test ./..."},
		{"Write file_path", "Write", `{"file_path":"/tmp/foo.go","content":"x"}`, "/tmp/foo.go"},
		{"Edit file_path", "Edit", `{"file_path":"/tmp/bar.go"}`, "/tmp/bar.go"},
		{"Read file_path", "Read", `{"file_path":"/tmp/baz.go"}`, "/tmp/baz.go"},
		{"AskUserQuestion", "AskUserQuestion", `{"questions":[{"question":"何を出しますか？","header":"じゃんけん","options":[{"label":"グー","description":"✊"},{"label":"チョキ","description":"✌️"},{"label":"パー","description":"🖐️"}],"multiSelect":false}]}`, "何を出しますか？\n1. グー\n2. チョキ\n3. パー\n4. Type something.\n5. Chat about this"},
		{"AskUserQuestion multiple", "AskUserQuestion", `{"questions":[{"question":"Q1","options":[{"label":"A"},{"label":"B"}]},{"question":"Q2","options":[{"label":"X"},{"label":"Y"}]}]}`, "Q1\n1. A\n2. B\n3. Type something.\n4. Chat about this\nQ2\n1. X\n2. Y\n3. Type something.\n4. Chat about this"},
		{"AskUserQuestion no options", "AskUserQuestion", `{"questions":[{"question":"Free text?"}]}`, "Free text?\n1. Type something.\n2. Chat about this"},
		{"unknown tool", "WebFetch", `{"url":"https://example.com"}`, ""},
		{"empty input", "Bash", ``, ""},
		{"invalid json", "Bash", `{invalid`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatToolInput(tt.toolName, json.RawMessage(tt.input))
			if got != tt.want {
				t.Errorf("FormatToolInput(%q, %q) = %q, want %q", tt.toolName, tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"long string", "hello world", 5, "hello..."},
		{"newlines replaced", "hello\nworld", 20, "hello world"},
		{"multibyte runes", "こんにちは世界", 3, "こんに..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func writeLines(t *testing.T, path string, lines []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
}
