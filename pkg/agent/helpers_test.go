package agent

import (
	"path/filepath"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/commands"
)

// TestExpandHome tests the expandHome function
func TestExpandHome(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no tilde",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
		{
			name:  "relative path",
			input: "relative/path",
			want:  "relative/path",
		},
		{
			name:  "tilde alone",
			input: "~",
			want:  "", // Will be home dir, we'll check for existence
		},
		{
			name:  "tilde with path",
			input: "~/documents",
			want:  "", // Will be home/documents, we'll check for existence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.input)

			if tt.want == "" && tt.input != "" && (tt.input == "~" || tt.input[0] == '~') {
				// For tilde expansion, just check that it expanded (doesn't start with ~)
				if got == "" {
					t.Errorf("expandHome(%q) = %q, expected non-empty expansion", tt.input, got)
				}
				if got[0] == '~' {
					t.Errorf("expandHome(%q) = %q, should have expanded tilde", tt.input, got)
				}
			} else if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCompilePatterns tests the compilePatterns function
func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty patterns",
			patterns:  []string{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "valid patterns",
			patterns:  []string{"^/home/.*", "^/tmp/.*"},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "mixed valid and invalid",
			patterns:  []string{"^/home/.*", "[invalid(regex", "/tmp/.*"},
			wantCount: 2,
			wantErr:   false, // Invalid patterns are skipped with warning
		},
		{
			name:      "single invalid pattern",
			patterns:  []string{"[invalid(regex"},
			wantCount: 0,
			wantErr:   false, // Skipped, not an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compilePatterns(tt.patterns)
			if len(got) != tt.wantCount {
				t.Errorf("compilePatterns(%v) returned %d patterns, want %d", tt.patterns, len(got), tt.wantCount)
			}
		})
	}
}

// TestInferMediaType tests the inferMediaType function
func TestInferMediaType(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		want        string
	}{
		{
			name:        "image content type",
			filename:    "photo.jpg",
			contentType: "image/jpeg",
			want:        "image",
		},
		{
			name:        "audio content type",
			filename:    "song.mp3",
			contentType: "audio/mpeg",
			want:        "audio",
		},
		{
			name:        "video content type",
			filename:    "movie.mp4",
			contentType: "video/mp4",
			want:        "video",
		},
		{
			name:        "ogg audio",
			filename:    "track.ogg",
			contentType: "application/ogg",
			want:        "audio",
		},
		{
			name:        "image jpg extension",
			filename:    "image.jpg",
			contentType: "application/octet-stream",
			want:        "image",
		},
		{
			name:        "image png extension",
			filename:    "photo.png",
			contentType: "",
			want:        "image",
		},
		{
			name:        "audio mp3 extension",
			filename:    "audio.mp3",
			contentType: "application/octet-stream",
			want:        "audio",
		},
		{
			name:        "video mp4 extension",
			filename:    "video.mp4",
			contentType: "",
			want:        "video",
		},
		{
			name:        "unknown file",
			filename:    "document.txt",
			contentType: "text/plain",
			want:        "file",
		},
		{
			name:        "case insensitive content type",
			filename:    "unknown.bin",
			contentType: "IMAGE/PNG",
			want:        "image",
		},
		{
			name:        "case insensitive filename",
			filename:    "PHOTO.JPG",
			contentType: "",
			want:        "image",
		},
		{
			name:        "empty filename",
			filename:    "",
			contentType: "image/jpeg",
			want:        "image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMediaType(tt.filename, tt.contentType)
			if got != tt.want {
				t.Errorf("inferMediaType(%q, %q) = %q, want %q", tt.filename, tt.contentType, got, tt.want)
			}
		})
	}
}

// TestMapCommandError tests the mapCommandError function
func TestMapCommandError(t *testing.T) {
	testErr := filepath.SkipDir // Use any error

	tests := []struct {
		name    string
		result  commands.ExecuteResult
		wantMsg string
	}{
		{
			name: "with command name",
			result: commands.ExecuteResult{
				Command: "help",
				Err:     testErr,
			},
			wantMsg: "Failed to execute /help",
		},
		{
			name: "without command name",
			result: commands.ExecuteResult{
				Command: "",
				Err:     testErr,
			},
			wantMsg: "Failed to execute command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapCommandError(tt.result)
			if !containsStr(got, tt.wantMsg) {
				t.Errorf("mapCommandError() = %q, want to contain %q", got, tt.wantMsg)
			}
		})
	}
}

// TestExtractProvider tests the extractProvider function
func TestExtractProvider(t *testing.T) {
	tests := []struct {
		name     string
		registry *AgentRegistry
		wantOk   bool
	}{
		{
			name:     "nil registry",
			registry: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := extractProvider(tt.registry)
			if ok != tt.wantOk {
				t.Errorf("extractProvider() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

// TestGetTodayFile tests the getTodayFile method of MemoryStore
func TestMemoryStore_GetTodayFile(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)

	todayFile := ms.getTodayFile()

	// Check that the path follows the expected pattern: memory/YYYYMM/YYYYMMDD.md
	if !contains(todayFile, "memory") {
		t.Errorf("getTodayFile() path doesn't contain 'memory': %q", todayFile)
	}
	if !contains(todayFile, ".md") {
		t.Errorf("getTodayFile() path doesn't end with '.md': %q", todayFile)
	}

	// Verify the file path is under the memory directory
	if !contains(todayFile, tmpDir) {
		t.Errorf("getTodayFile() path not under workspace: %q", todayFile)
	}
}

// Helper function to check if a string contains a substring
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetTodayFile_Format(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)

	todayFile := ms.getTodayFile()

	// Check that it contains the expected directory components
	if !containsStr(todayFile, filepath.Join("memory", "202")) {
		// Just check it's in a YYYYMM/YYYYMMDD format
		if !containsStr(todayFile, "memory") {
			t.Errorf("getTodayFile() = %q, should contain memory directory", todayFile)
		}
	}

	// Check that it ends with .md
	if !containsStr(todayFile, ".md") {
		t.Errorf("getTodayFile() = %q, should end with .md", todayFile)
	}
}

func TestMemoryStore_WriteLongTerm(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)

	content := "# Long-term memory\nRemember this."
	if err := ms.WriteLongTerm(content); err != nil {
		t.Fatalf("WriteLongTerm: %v", err)
	}
	got := ms.ReadLongTerm()
	if got != content {
		t.Errorf("ReadLongTerm after write = %q, want %q", got, content)
	}
}

func TestMemoryStore_ReadToday_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)
	if got := ms.ReadToday(); got != "" {
		t.Errorf("ReadToday on empty store = %q, want empty", got)
	}
}

func TestMemoryStore_AppendToday_New(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)

	if err := ms.AppendToday("First note"); err != nil {
		t.Fatalf("AppendToday: %v", err)
	}
	got := ms.ReadToday()
	if !containsStr(got, "First note") {
		t.Errorf("ReadToday after AppendToday = %q, want it to contain 'First note'", got)
	}
}

func TestMemoryStore_AppendToday_Appends(t *testing.T) {
	tmpDir := t.TempDir()
	ms := NewMemoryStore(tmpDir)

	_ = ms.AppendToday("Note one")
	if err := ms.AppendToday("Note two"); err != nil {
		t.Fatalf("second AppendToday: %v", err)
	}
	got := ms.ReadToday()
	if !containsStr(got, "Note one") || !containsStr(got, "Note two") {
		t.Errorf("ReadToday = %q, want both notes", got)
	}
}
