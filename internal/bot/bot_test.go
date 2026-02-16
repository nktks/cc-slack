package bot

import "testing"

func TestStripMention(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<@U12345> hello world", "hello world"},
		{"<@U12345>hello", "hello"},
		{"<@U12345>  multiple spaces", "multiple spaces"},
		{"no mention here", "no mention here"},
		{"<@U12345> ", ""},
		{"<@U12345>", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StripMention(tt.input)
			if got != tt.want {
				t.Errorf("StripMention(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
