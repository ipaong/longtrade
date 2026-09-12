package utils

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hi",
			maxLen: 10,
			want:   "hi",
		},
		{
			name:   "exact length unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated with ellipsis",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "maxLen equals 4 leaves 1 char plus ellipsis",
			input:  "abcdef",
			maxLen: 4,
			want:   "a...",
		},
		{
			name:   "maxLen 3 returns first 3 chars without ellipsis",
			input:  "abcdef",
			maxLen: 3,
			want:   "abc",
		},
		{
			name:   "maxLen 2 returns first 2 chars",
			input:  "abcdef",
			maxLen: 2,
			want:   "ab",
		},
		{
			name:   "maxLen 1 returns first char",
			input:  "abcdef",
			maxLen: 1,
			want:   "a",
		},
		{
			name:   "maxLen 0 returns empty",
			input:  "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "negative maxLen returns empty",
			input:  "hello",
			maxLen: -1,
			want:   "",
		},
		{
			name:   "empty string unchanged",
			input:  "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "empty string with zero maxLen",
			input:  "",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "unicode truncated correctly",
			input:  "\U0001f600\U0001f601\U0001f602\U0001f603\U0001f604",
			maxLen: 4,
			want:   "\U0001f600...",
		},
		{
			name:   "unicode short enough",
			input:  "\u00e9\u00e8",
			maxLen: 5,
			want:   "\u00e9\u00e8",
		},
		{
			name:   "mixed ascii and unicode",
			input:  "Go\U0001f680\U0001f525\U0001f4a5\U0001f30d",
			maxLen: 5,
			want:   "Go...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSanitizeMessageContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text unchanged", "Hello world", "Hello world"},
		{"strip ZWSP", "Hello\u200bworld", "Helloworld"},
		{"strip RTL override", "Hi\u202eevil", "Hievil"},
		{"strip BOM", "\uFEFFcontent", "content"},
		{"strip multiple", "a\u200c\u202ab\u202cc", "abc"},
		{"unicode letters preserved", "café \u65e5\u672c\u8a9e", "café \u65e5\u672c\u8a9e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeMessageContent(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeMessageContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetDisableTruncation_Disables(t *testing.T) {
	SetDisableTruncation(true)
	t.Cleanup(func() { SetDisableTruncation(false) })

	long := "hello world this is a long string"
	got := Truncate(long, 5)
	if got != long {
		t.Errorf("Truncate with DisableTruncation=true should return full string, got %q", got)
	}
}

func TestSetDisableTruncation_Enables(t *testing.T) {
	SetDisableTruncation(false)
	long := "hello world this is long"
	got := Truncate(long, 8)
	if got == long {
		t.Errorf("Truncate with DisableTruncation=false should truncate")
	}
}

func TestDerefStr_Nil(t *testing.T) {
	if got := DerefStr(nil, "fallback"); got != "fallback" {
		t.Errorf("DerefStr nil = %q, want fallback", got)
	}
}

func TestDerefStr_NonNil(t *testing.T) {
	s := "hello"
	if got := DerefStr(&s, "fallback"); got != "hello" {
		t.Errorf("DerefStr non-nil = %q, want hello", got)
	}
}

func TestDerefStr_EmptyFallback(t *testing.T) {
	if got := DerefStr(nil, ""); got != "" {
		t.Errorf("DerefStr nil empty fallback = %q, want empty", got)
	}
}
