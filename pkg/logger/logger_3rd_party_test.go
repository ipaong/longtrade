package logger

import (
	"testing"
)

// --- maskSecrets ---

func TestMaskSecrets_TelegramBotToken(t *testing.T) {
	token := "bot123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh"
	input := "my bot token is " + token
	result := maskSecrets(input)

	if result == input {
		t.Error("bot token should be masked")
	}
	// Should have format bot123456789:ABCD****efgh
	if !hasSubstring(result, "bot123456789:") {
		t.Error("should preserve bot ID prefix")
	}
	if !hasSubstring(result, "****") {
		t.Error("should mask middle section with ****")
	}
}

func TestMaskSecrets_MultipleTokens(t *testing.T) {
	token1 := "bot111111111:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh"
	token2 := "bot222222222:XYZabcdefghijklmnopqrstuvwxyzABCDEFGH"
	input := "token1: " + token1 + " token2: " + token2
	result := maskSecrets(input)

	if result == input {
		t.Error("tokens should be masked")
	}
	if countOccurrences(result, "****") < 2 {
		t.Error("should mask both tokens")
	}
}

func TestMaskSecrets_NoTokens(t *testing.T) {
	input := "just a regular message without tokens"
	result := maskSecrets(input)

	if result != input {
		t.Errorf("message without tokens should not change: %q -> %q", input, result)
	}
}

// --- Logger.Debug ---

func TestLogger_Debug(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	l := NewLogger("testcomp")
	l.Debug("debug message", "with", "multiple", "args")
	// Should not panic
}

// --- Logger.Info ---

func TestLogger_Info(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	l := NewLogger("infocomp")
	l.Info("info message")
	// Should not panic
}

// --- Logger.Warn ---

func TestLogger_Warn(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	l := NewLogger("warncomp")
	l.Warn("warning message")
	// Should not panic
}

// --- Logger.Error ---

func TestLogger_Error(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	l := NewLogger("errcomp")
	l.Error("error message", "error detail")
	// Should not panic
}

// --- Logger.Debugf ---

func TestLogger_Debugf(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	l := NewLogger("comp1")
	l.Debugf("debug formatted: %s %d", "message", 42)
	// Should not panic
}

// --- Logger.Infof ---

func TestLogger_Infof(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	l := NewLogger("comp2")
	l.Infof("info formatted: %v", "test")
	// Should not panic
}

// --- Logger.Warnf ---

func TestLogger_Warnf(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	l := NewLogger("comp3")
	l.Warnf("warning formatted: code=%d", 500)
	// Should not panic
}

// --- Logger.Warningf ---

func TestLogger_Warningf(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	l := NewLogger("comp4")
	l.Warningf("warning (alias): %s", "message")
	// Should not panic
}

// --- Logger.Errorf ---

func TestLogger_Errorf(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	l := NewLogger("comp5")
	l.Errorf("error formatted: %v", "failed")
	// Should not panic
}

// --- Logger.Log ---

func TestLogger_Log_DebugLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	l := NewLogger("logcomp")
	l.Log(int(DEBUG), 0, "log at debug: %s", "test")
	// Should not panic
}

func TestLogger_Log_InfoLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	l := NewLogger("logcomp2")
	l.Log(int(INFO), 0, "log at info: %s", "test")
	// Should not panic
}

func TestLogger_Log_ErrorLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	l := NewLogger("logcomp3")
	l.Log(int(ERROR), 0, "log at error: %s", "test")
	// Should not panic
}

func TestLogger_Log_WithCustomLevels(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	l := NewLogger("logcomp4")
	customLevels := map[int]LogLevel{
		int(WARN): INFO, // Remap WARN messages to INFO level
	}
	l = l.WithLevels(customLevels)
	l.Log(int(WARN), 0, "mapped warn to info: %s", "test")
	// Should not panic
}

// --- Logger.Sync ---

func TestLogger_Sync(t *testing.T) {
	l := NewLogger("synccomp")
	err := l.Sync()
	if err != nil {
		t.Errorf("Sync should not return error: %v", err)
	}
}

// --- Logger.WithLevels ---

func TestLogger_WithLevels_SetsMapping(t *testing.T) {
	l := NewLogger("comp")
	if l.levels != nil {
		t.Error("levels should be nil initially")
	}

	customLevels := map[int]LogLevel{
		int(WARN): DEBUG,
	}
	l2 := l.WithLevels(customLevels)

	if l2.levels == nil {
		t.Error("WithLevels should set levels mapping")
	}
	if l2.levels[int(WARN)] != DEBUG {
		t.Errorf("level mapping not set correctly")
	}
}

// --- NewLogger ---

func TestNewLogger_CreatesWithComponent(t *testing.T) {
	componentName := "mycomponent"
	l := NewLogger(componentName)

	if l.component != componentName {
		t.Errorf("component: got %q, want %q", l.component, componentName)
	}
	if l.levels != nil {
		t.Error("levels should be nil initially")
	}
}

func TestNewLogger_EmptyComponent(t *testing.T) {
	l := NewLogger("")
	if l.component != "" {
		t.Error("empty component should be preserved")
	}
}

// --- maskSecrets edge cases ---

func TestMaskSecrets_ShortToken(t *testing.T) {
	// Token too short to match full pattern
	shortToken := "bot123456789:short"
	input := "token: " + shortToken
	_ = maskSecrets(input)
	// Should not crash, even if pattern doesn't match
}

func TestMaskSecrets_EmptyString(t *testing.T) {
	result := maskSecrets("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

func TestMaskSecrets_OnlyPrefix(t *testing.T) {
	input := "bot123456789:"
	result := maskSecrets(input)
	// Should not crash
	if len(result) == 0 {
		t.Error("should preserve input if pattern doesn't fully match")
	}
}
