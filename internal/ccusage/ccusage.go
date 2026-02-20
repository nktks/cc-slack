package ccusage

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type report struct {
	Weekly []weekEntry `json:"weekly"`
	Totals totals      `json:"totals"`
}

type weekEntry struct {
	Week               string           `json:"week"`
	InputTokens        int64            `json:"inputTokens"`
	OutputTokens       int64            `json:"outputTokens"`
	CacheCreationTokens int64           `json:"cacheCreationTokens"`
	CacheReadTokens    int64            `json:"cacheReadTokens"`
	TotalTokens        int64            `json:"totalTokens"`
	TotalCost          float64          `json:"totalCost"`
	ModelsUsed         []string         `json:"modelsUsed"`
	ModelBreakdowns    []modelBreakdown `json:"modelBreakdowns"`
}

type modelBreakdown struct {
	ModelName string  `json:"modelName"`
	Cost      float64 `json:"cost"`
}

type totals struct {
	InputTokens        int64   `json:"inputTokens"`
	OutputTokens       int64   `json:"outputTokens"`
	CacheCreationTokens int64  `json:"cacheCreationTokens"`
	CacheReadTokens    int64   `json:"cacheReadTokens"`
	TotalTokens        int64   `json:"totalTokens"`
	TotalCost          float64 `json:"totalCost"`
}

// Run executes ccusage weekly --json and returns the raw JSON output.
func Run() ([]byte, error) {
	out, err := exec.Command("ccusage", "weekly", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("ccusage command: %w", err)
	}
	return out, nil
}

// FormatSlackTable parses ccusage weekly JSON and returns a Slack mrkdwn table.
func FormatSlackTable(data []byte) (string, error) {
	var r report
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("parse ccusage json: %w", err)
	}

	var b strings.Builder
	b.WriteString("*Weekly Usage Report*\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("| %-10s | %-20s | %8s | %8s | %8s | %8s | %8s | %9s |\n",
		"Week", "Models", "Input", "Output", "CacheWr", "CacheRd", "Total", "Cost"))
	b.WriteString(fmt.Sprintf("|%s|%s|%s|%s|%s|%s|%s|%s|\n",
		dash(12), dash(22), dash(10), dash(10), dash(10), dash(10), dash(10), dash(11)))

	for _, w := range r.Weekly {
		models := shortModels(w.ModelsUsed)
		b.WriteString(fmt.Sprintf("| %-10s | %-20s | %8s | %8s | %8s | %8s | %8s | %9s |\n",
			w.Week,
			truncate(models, 20),
			fmtTokens(w.InputTokens),
			fmtTokens(w.OutputTokens),
			fmtTokens(w.CacheCreationTokens),
			fmtTokens(w.CacheReadTokens),
			fmtTokens(w.TotalTokens),
			fmtCost(w.TotalCost)))
	}

	b.WriteString(fmt.Sprintf("|%s|%s|%s|%s|%s|%s|%s|%s|\n",
		dash(12), dash(22), dash(10), dash(10), dash(10), dash(10), dash(10), dash(11)))
	b.WriteString(fmt.Sprintf("| %-10s | %-20s | %8s | %8s | %8s | %8s | %8s | %9s |\n",
		"Total", "",
		fmtTokens(r.Totals.InputTokens),
		fmtTokens(r.Totals.OutputTokens),
		fmtTokens(r.Totals.CacheCreationTokens),
		fmtTokens(r.Totals.CacheReadTokens),
		fmtTokens(r.Totals.TotalTokens),
		fmtCost(r.Totals.TotalCost)))
	b.WriteString("```")

	return b.String(), nil
}

func fmtTokens(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func fmtCost(c float64) string {
	return fmt.Sprintf("$%.2f", c)
}

func shortModels(models []string) string {
	short := make([]string, len(models))
	for i, m := range models {
		// "claude-opus-4-6" -> "opus-4-6"
		s := strings.TrimPrefix(m, "claude-")
		// "opus-4-5-20251101" -> "opus-4-5"
		parts := strings.Split(s, "-")
		if len(parts) > 3 {
			s = strings.Join(parts[:3], "-")
		}
		short[i] = s
	}
	return strings.Join(short, ", ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "~"
}

func dash(n int) string {
	return strings.Repeat("-", n)
}
