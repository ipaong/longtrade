package providers

import (
	"os"
	"strings"
	"testing"
)

func TestExpandTilde_NoTilde(t *testing.T) {
	if got := expandTilde("/absolute/path"); got != "/absolute/path" {
		t.Errorf("expandTilde no tilde = %q, want unchanged", got)
	}
}

func TestExpandTilde_Empty(t *testing.T) {
	if got := expandTilde(""); got != "" {
		t.Errorf("expandTilde empty = %q, want empty", got)
	}
}

func TestExpandTilde_TildePrefix(t *testing.T) {
	got := expandTilde("~/models/llama")
	if got == "~/models/llama" {
		t.Error("expandTilde should expand ~, got unchanged")
	}
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(got, home) {
		t.Errorf("expandTilde = %q, should start with home dir %q", got, home)
	}
	if !strings.HasSuffix(got, "models/llama") {
		t.Errorf("expandTilde = %q, should end with models/llama", got)
	}
}

func TestExpandTilde_TildeOnly(t *testing.T) {
	got := expandTilde("~")
	if got == "~" {
		t.Error("expandTilde '~' should expand to home dir")
	}
}
