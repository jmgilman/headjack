package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsClaudeToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "regular text",
			input: "This is just some regular text",
			want:  false,
		},
		{
			name:  "claude oauth token",
			input: "sk-ant-oat01-abc123xyz",
			want:  true,
		},
		{
			name:  "claude token with longer suffix",
			input: "sk-ant-oat01-aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789",
			want:  true,
		},
		{
			name:  "sk-ant prefix only",
			input: "sk-ant-",
			want:  true,
		},
		{
			name:  "similar but wrong prefix",
			input: "sk-anth-oat01-abc123",
			want:  false,
		},
		{
			name:  "openai-style key",
			input: "sk-proj-abc123",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClaudeToken(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "token on its own line",
			output: "Some text\nsk-ant-oat01-abc123\nMore text",
			want:   "sk-ant-oat01-abc123",
		},
		{
			name:   "token with CRLF line endings",
			output: "Some text\r\nsk-ant-oat01-abc123\r\nMore text",
			want:   "sk-ant-oat01-abc123",
		},
		{
			name:   "token with CR line endings",
			output: "Some text\rsk-ant-oat01-abc123\rMore text",
			want:   "sk-ant-oat01-abc123",
		},
		{
			name:   "token with surrounding whitespace",
			output: "Some text\n  sk-ant-oat01-abc123  \nMore text",
			want:   "sk-ant-oat01-abc123",
		},
		{
			name:   "no token in output",
			output: "Some text\nNo token here\nMore text",
			want:   "",
		},
		{
			name:   "token with ANSI codes",
			output: "Some text\n\x1b[32msk-ant-oat01-abc123\x1b[0m\nMore text",
			want:   "sk-ant-oat01-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToken(tt.output)
			assert.Equal(t, tt.want, got)
		})
	}
}
