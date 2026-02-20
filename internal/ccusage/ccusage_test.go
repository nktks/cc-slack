package ccusage

import (
	"strings"
	"testing"
)

func TestFormatSlackTable(t *testing.T) {
	input := `{
  "weekly": [
    {
      "week": "2026-02-08",
      "inputTokens": 28959264,
      "outputTokens": 48997,
      "cacheCreationTokens": 20576,
      "cacheReadTokens": 2505048,
      "totalTokens": 31533885,
      "totalCost": 122.32,
      "modelsUsed": ["claude-opus-4-6", "claude-haiku-4-5-20251001"],
      "modelBreakdowns": []
    },
    {
      "week": "2026-02-15",
      "inputTokens": 102745410,
      "outputTokens": 208918,
      "cacheCreationTokens": 126024,
      "cacheReadTokens": 6063148,
      "totalTokens": 109143500,
      "totalCost": 494.28,
      "modelsUsed": ["claude-opus-4-6"],
      "modelBreakdowns": []
    }
  ],
  "totals": {
    "inputTokens": 131704674,
    "outputTokens": 257915,
    "cacheCreationTokens": 146600,
    "cacheReadTokens": 8568196,
    "totalTokens": 140677385,
    "totalCost": 616.60
  }
}`

	got, err := FormatSlackTable([]byte(input))
	if err != nil {
		t.Fatalf("FormatSlackTable: %v", err)
	}

	if !strings.Contains(got, "*Weekly Usage Report*") {
		t.Error("missing header")
	}
	if !strings.Contains(got, "2026-02-08") {
		t.Error("missing week 2026-02-08")
	}
	if !strings.Contains(got, "2026-02-15") {
		t.Error("missing week 2026-02-15")
	}
	if !strings.Contains(got, "Total") {
		t.Error("missing total row")
	}
	if !strings.Contains(got, "$616.60") {
		t.Error("missing total cost")
	}
	if !strings.Contains(got, "opus-4-6") {
		t.Error("missing model name")
	}
	if !strings.Contains(got, "haiku-4-5") {
		t.Error("missing model name haiku-4-5")
	}
}

func TestFmtTokens(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1500, "1.5K"},
		{1500000, "1.5M"},
		{1500000000, "1.5B"},
	}
	for _, tt := range tests {
		got := fmtTokens(tt.input)
		if got != tt.want {
			t.Errorf("fmtTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortModels(t *testing.T) {
	models := []string{"claude-opus-4-5-20251101", "claude-haiku-4-5-20251001", "claude-sonnet-4-5-20250929"}
	got := shortModels(models)
	if got != "opus-4-5, haiku-4-5, sonnet-4-5" {
		t.Errorf("shortModels = %q", got)
	}
}

func TestFormatSlackTableInvalidJSON(t *testing.T) {
	_, err := FormatSlackTable([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
