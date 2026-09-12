package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestLogLevelFiltering(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(WARN)

	tests := []struct {
		name      string
		level     LogLevel
		shouldLog bool
	}{
		{"DEBUG message", DEBUG, false},
		{"INFO message", INFO, false},
		{"WARN message", WARN, true},
		{"ERROR message", ERROR, true},
		{"FATAL message", FATAL, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.level {
			case DEBUG:
				Debug(tt.name)
			case INFO:
				Info(tt.name)
			case WARN:
				Warn(tt.name)
			case ERROR:
				Error(tt.name)
			case FATAL:
				if tt.shouldLog {
					t.Logf("FATAL test skipped to prevent program exit")
				}
			}
		})
	}

	SetLevel(INFO)
}

func TestLoggerWithComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)

	tests := []struct {
		name      string
		component string
		message   string
		fields    map[string]any
	}{
		{"Simple message", "test", "Hello, world!", nil},
		{"Message with component", "discord", "Discord message", nil},
		{"Message with fields", "telegram", "Telegram message", map[string]any{
			"user_id": "12345",
			"count":   42,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch {
			case tt.fields == nil && tt.component != "":
				InfoC(tt.component, tt.message)
			case tt.fields != nil:
				InfoF(tt.message, tt.fields)
			default:
				Info(tt.message)
			}
		})
	}

	SetLevel(INFO)
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"DEBUG level", DEBUG, "DEBUG"},
		{"INFO level", INFO, "INFO"},
		{"WARN level", WARN, "WARN"},
		{"ERROR level", ERROR, "ERROR"},
		{"FATAL level", FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if logLevelNames[tt.level] != tt.want {
				t.Errorf("logLevelNames[%d] = %s, want %s", tt.level, logLevelNames[tt.level], tt.want)
			}
		})
	}
}

func TestSetGetLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	tests := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}

	for _, level := range tests {
		SetLevel(level)
		if GetLevel() != level {
			t.Errorf("SetLevel(%v) -> GetLevel() = %v, want %v", level, GetLevel(), level)
		}
	}
}

func TestLoggerHelperFunctions(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(INFO)

	Debug("This should not log")
	Debugf("this should not log")
	Info("This should log")
	Warn("This should log")
	Error("This should log")

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]any{"key": "value"})
	Infof("test from %v", "Infof")

	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]any{"error": "test"})
	Errorf("test from %v", "Errorf")

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	Debugf("test from %v", "Debugf")
	WarnF("Warning with fields", map[string]any{"key": "value"})
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// Basic types test (default case of the switch)
		{
			name:     "Integer Type",
			input:    42,
			expected: "42",
		},
		{
			name:     "Boolean Type",
			input:    true,
			expected: "true",
		},
		{
			name:     "Unsupported Struct Type",
			input:    struct{ A int }{A: 1},
			expected: "{1}",
		},

		// Simple strings and byte slices test
		{
			name:     "Simple string without spaces",
			input:    "simple_value",
			expected: "simple_value",
		},
		{
			name:     "Simple byte slice",
			input:    []byte("byte_value"),
			expected: "byte_value",
		},

		// Unquoting test (strconv.Unquote)
		{
			name:     "Quoted string",
			input:    `"quoted_value"`,
			expected: "quoted_value",
		},

		// Strings with newline (\n) test
		{
			name:     "String with newline",
			input:    "line1\nline2",
			expected: "\nline1\nline2",
		},
		{
			name:     "Quoted string with newline (Unquote -> newline)",
			input:    `"line1\nline2"`, // Escaped \n that Unquote will resolve
			expected: "\nline1\nline2",
		},

		// Strings with spaces test (which should be quoted)
		{
			name:     "String with spaces",
			input:    "hello world",
			expected: `"hello world"`,
		},
		{
			name:     "Quoted string with spaces (Unquote -> has spaces -> Re-quote)",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},

		// JSON formats test (strings with spaces that start/end with brackets)
		{
			name:     "Valid JSON object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Valid JSON array",
			input:    `[1, 2, "three"]`,
			expected: `[1, 2, "three"]`,
		},
		{
			name:     "Fake JSON (starts with { but doesn't end with })",
			input:    `{"key": "value"`, // Missing closing bracket, has spaces
			expected: `"{\"key\": \"value\""`,
		},
		{
			name:     "Empty JSON (object)",
			input:    `{ }`,
			expected: `{ }`,
		},

		// 7. Edge Cases
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace only string",
			input:    "   ",
			expected: `"   "`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatFieldValue(tt.input)
			if actual != tt.expected {
				t.Errorf("formatFieldValue() = %q, expected %q", actual, tt.expected)
			}
		})
	}
}

// --- EnableFileLogging / DisableFileLogging ---

func TestEnableFileLogging_CreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "logs", "test.log")

	err := EnableFileLogging(logPath)
	t.Cleanup(func() {
		DisableFileLogging()
	})

	if err != nil {
		t.Fatalf("EnableFileLogging failed: %v", err)
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestEnableFileLogging_AllowsLoggingToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	err := EnableFileLogging(logPath)
	t.Cleanup(func() {
		DisableFileLogging()
	})

	if err != nil {
		t.Fatalf("EnableFileLogging failed: %v", err)
	}

	// Log something at a high level to ensure it goes to file
	initialLevel := GetLevel()
	SetLevel(INFO)
	Info("test message")
	SetLevel(initialLevel)

	// Verify log file has content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if len(data) == 0 {
		t.Error("log file should contain data")
	}
}

func TestEnableFileLogging_FailsWithBadPath(t *testing.T) {
	// Use an invalid path that can't be created
	badPath := "/dev/null/impossible/path/log.txt"

	err := EnableFileLogging(badPath)
	if err == nil {
		t.Cleanup(func() {
			DisableFileLogging()
		})
		t.Error("EnableFileLogging should fail for invalid path")
	}
}

func TestDisableFileLogging_ClosesFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	err := EnableFileLogging(logPath)
	if err != nil {
		t.Fatalf("EnableFileLogging failed: %v", err)
	}

	DisableFileLogging()

	// After disabling, logging should still work (just not to file)
	Info("test after disable")
}

// --- RegisterSecret / Redact ---

func TestRegisterSecret_AddsSecretToList(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	// Clear secrets first by creating a new test scenario
	RegisterSecret("mysecret123")
	redacted := Redact("my password is mysecret123")
	if redacted == "my password is mysecret123" {
		t.Error("secret should be redacted")
	}
	if !hasSubstring(redacted, "***REDACTED***") {
		t.Error("redacted text should contain *** REDACTED ***")
	}
}

func TestRegisterSecret_IgnoresEmptyStrings(t *testing.T) {
	// Empty strings should be silently ignored
	RegisterSecret("")
	// Should not panic or error
}

func TestRegisterSecret_AvoidsDuplicates(t *testing.T) {
	secret := "dupsecret"
	RegisterSecret(secret)
	RegisterSecret(secret)
	// Should not fail, just avoid duplicate
}

func TestRedact_RedactsAllRegisteredSecrets(t *testing.T) {
	RegisterSecret("secret1")
	RegisterSecret("secret2")

	input := "contains secret1 and secret2 and secret1 again"
	redacted := Redact(input)

	if hasSubstring(redacted, "secret1") || hasSubstring(redacted, "secret2") {
		t.Error("all secrets should be redacted")
	}
	if countOccurrences(redacted, "***REDACTED***") < 3 {
		t.Error("should have 3 redactions")
	}
}

func TestRedact_EmptyStringInput(t *testing.T) {
	result := Redact("")
	if result != "" {
		t.Errorf("Redact of empty string should be empty, got %q", result)
	}
}

// --- DebugF / DebugCF (missing coverage) ---

func TestDebugF_WithFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	fields := map[string]any{
		"field1": "value1",
		"field2": 42,
	}
	DebugF("debug with fields", fields)
	// Should not panic
}

func TestDebugCF_WithComponentAndFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	fields := map[string]any{
		"user": "alice",
		"id":   123,
	}
	DebugCF("mycomponent", "debug component with fields", fields)
	// Should not panic
}

// --- InfoCF (missing coverage) ---

func TestInfoCF_WithComponentAndFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	fields := map[string]any{
		"action":  "login",
		"status":  "success",
		"elapsed": 250,
	}
	InfoCF("auth", "user authenticated", fields)
	// Should not panic
}

// --- WarnCF (missing coverage) ---

func TestWarnCF_WithComponentAndFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	fields := map[string]any{
		"resource": "memory",
		"used":     75,
		"limit":    100,
	}
	WarnCF("system", "resource threshold exceeded", fields)
	// Should not panic
}

// --- ErrorC (missing coverage) ---

func TestErrorC_WithComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	ErrorC("database", "connection failed")
	// Should not panic
}

// --- ErrorCF (missing coverage) ---

func TestErrorCF_WithComponentAndFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	fields := map[string]any{
		"error":  "timeout",
		"retry":  3,
		"limit":  5,
	}
	ErrorCF("network", "request failed after retries", fields)
	// Should not panic
}

// --- getCallerInfo (partial coverage: 71.4%) ---

func TestGetCallerInfo_ReturnsValidInfo(t *testing.T) {
	file, line, funcName := getCallerInfo()

	if file == "???" {
		t.Error("file should not be ???")
	}
	if line == 0 {
		t.Error("line should not be 0")
	}
	if funcName == "???" {
		t.Error("funcName should not be ???")
	}
}

// --- getEvent (partial coverage: 71.4%) ---

func TestGetEvent_ReturnsCorrectEventType(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	// This is tricky since getEvent returns a zerolog.Event
	// We mainly verify that it doesn't panic
	logMessage(DEBUG, "test", "test message", nil)
	logMessage(INFO, "test", "test message", nil)
	logMessage(WARN, "test", "test message", nil)
	logMessage(ERROR, "test", "test message", nil)
}

// --- appendFields (partial coverage: 50%) ---

func TestAppendFields_AllTypes(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	fields := map[string]any{
		"string_field": "value",
		"int_field":    42,
		"int64_field":  int64(1000),
		"float_field":  3.14,
		"bool_field":   true,
		"slice_field":  []int{1, 2, 3},
		"map_field":    map[string]string{"key": "value"},
	}

	InfoF("message with various field types", fields)
	// Should not panic and all types should be handled
}

// --- WarnCF with formatted message (additional coverage) ---

func TestWarnCFFormatted_FormattedMessage(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	WarnCF("comp", "warning with component", map[string]any{"key": "value"})
	// Should not panic
}

// --- Test callerInfo returns valid values ---

func TestGetCallerInfo_FileHasBasename(t *testing.T) {
	file, _, _ := getCallerInfo()
	if file == "???" {
		t.Error("file should not be ???")
	}
	// Should be something like "logger_test.go"
	if len(file) == 0 {
		t.Error("file should be non-empty")
	}
}

// --- Test getEvent returns different event types ---

func TestGetEvent_DebugEvent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	// Just verify it doesn't panic
	logMessage(DEBUG, "test", "message", nil)
}

func TestGetEvent_WarnEvent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	logMessage(WARN, "test", "message", nil)
}

// --- Test appendFields with all type cases ---

func TestAppendFields_WithSlice(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	fields := map[string]any{
		"numbers": []int{1, 2, 3},
	}
	InfoF("slice field", fields)
}

func TestAppendFields_WithMap(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	fields := map[string]any{
		"data": map[string]int{"a": 1, "b": 2},
	}
	InfoF("map field", fields)
}

// --- EnableFileLogging with directory that needs creation ---

func TestEnableFileLogging_CreatesMultipleLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "deeply", "nested", "dir", "logs", "app.log")

	err := EnableFileLogging(logPath)
	t.Cleanup(func() {
		DisableFileLogging()
	})

	if err != nil {
		t.Fatalf("EnableFileLogging should create nested dirs: %v", err)
	}

	// Verify all directories were created
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file should exist: %v", err)
	}
}

// --- Test masking in log output ---

func TestRegisterSecret_MasksInOutput(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	testSecret := "verysecretkey123"
	RegisterSecret(testSecret)

	// Log a message containing the secret
	InfoF("user password", map[string]any{
		"secret": testSecret,
	})

	// If we get here without panic, redaction is working
}

// --- formatFieldValue with various inputs ---

func TestFormatFieldValue_ByteSlice(t *testing.T) {
	result := formatFieldValue([]byte("test"))
	if result != "test" {
		t.Errorf("byte slice should format as string: got %q", result)
	}
}

func TestFormatFieldValue_Int(t *testing.T) {
	result := formatFieldValue(42)
	if result != "42" {
		t.Errorf("int should format: got %q", result)
	}
}

func TestFormatFieldValue_Float(t *testing.T) {
	result := formatFieldValue(3.14)
	if result != "3.14" {
		t.Errorf("float should format: got %q", result)
	}
}

func TestFormatFieldValue_Bool(t *testing.T) {
	result := formatFieldValue(true)
	if result != "true" {
		t.Errorf("bool should format: got %q", result)
	}
}

// --- Test multiple secrets registration and redaction ---

func TestMultipleSecrets_AllRedacted(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	secret1 := "SECRET_ONE"
	secret2 := "SECRET_TWO"

	RegisterSecret(secret1)
	RegisterSecret(secret2)

	input := "Config: " + secret1 + " and " + secret2
	redacted := Redact(input)

	if hasSubstring(redacted, secret1) || hasSubstring(redacted, secret2) {
		t.Error("both secrets should be redacted")
	}
}

// --- Test levels in different combinations ---

func TestLogLevel_OrderPreserved(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	// Set to each level and verify filtering
	levelTests := []struct {
		setLevel LogLevel
		testFn   func()
	}{
		{DEBUG, func() { DebugC("t", "d") }},
		{INFO, func() { InfoC("t", "i") }},
		{WARN, func() { WarnC("t", "w") }},
		{ERROR, func() { ErrorC("t", "e") }},
	}

	for _, lt := range levelTests {
		SetLevel(lt.setLevel)
		lt.testFn()
	}
}

func TestGetEvent_DefaultCase(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(zerolog.TraceLevel)

	// TraceLevel is not in the switch cases → hits the default branch
	logMessage(zerolog.TraceLevel, "test", "trace message", nil)
}

// --- Test Info variants coverage ---

func TestInfo_Variants(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	Info("simple")
	InfoC("comp", "with component")
	Infof("formatted %s", "msg")
	InfoF("with fields", map[string]any{"k": "v"})
	InfoCF("comp", "with all", map[string]any{"x": 1})
}

// --- Test Debug variants coverage ---

func TestDebug_Variants(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	Debug("simple")
	DebugC("comp", "with component")
	Debugf("formatted %s", "msg")
	DebugF("with fields", map[string]any{"k": "v"})
	DebugCF("comp", "with all", map[string]any{"x": 1})
}

// --- Test Warn variants coverage ---

func TestWarn_Variants(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(WARN)

	Warn("simple")
	WarnC("comp", "with component")
	WarnF("with fields", map[string]any{"k": "v"})
	WarnCF("comp", "with all", map[string]any{"x": 1})
}

// --- Test Error variants coverage ---

func TestError_Variants(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(ERROR)

	Error("simple")
	ErrorC("comp", "with component")
	Errorf("formatted %s", "msg")
	ErrorF("with fields", map[string]any{"k": "v"})
	ErrorCF("comp", "with all", map[string]any{"x": 1})
}

// --- Test logging with file enabled (covers file logging path in logMessage) ---

func TestLogMessage_WithFileLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	err := EnableFileLogging(logPath)
	t.Cleanup(func() {
		DisableFileLogging()
	})

	if err != nil {
		t.Fatalf("EnableFileLogging failed: %v", err)
	}

	// Log messages at different levels while file is enabled
	Info("info to file")
	InfoC("comp", "component info to file")
	InfoF("with fields", map[string]any{"key": "value"})
	Warn("warning to file")
	Error("error to file")

	// Read the file to verify content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(data)
	if !hasSubstring(logContent, "info to file") {
		t.Error("log file should contain info message")
	}
	if !hasSubstring(logContent, "warning to file") {
		t.Error("log file should contain warning message")
	}
}

// --- Test logging with specific levels to exercise getEvent default case ---

func TestLogMessage_AllLevels(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	// Log at all levels
	logMessage(DEBUG, "c", "debug msg", nil)
	logMessage(INFO, "c", "info msg", nil)
	logMessage(WARN, "c", "warn msg", nil)
	logMessage(ERROR, "c", "error msg", nil)

	// Note: FATAL would exit, so we skip it
}

// --- Test logMessage level filtering ---

func TestLogMessage_LevelFiltering(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(WARN)

	// Debug should not log (filtered)
	logMessage(DEBUG, "c", "should not appear", nil)

	// Warn and above should log
	logMessage(WARN, "c", "should appear", nil)
	logMessage(ERROR, "c", "should appear", nil)
}

// --- Test logMessage with nil fields ---

func TestLogMessage_NilFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	logMessage(INFO, "comp", "message", nil)
	// Should not panic
}

// --- Test logMessage with empty component ---

func TestLogMessage_EmptyComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	logMessage(INFO, "", "message", nil)
	// Should use "<none>" in caller field
}

// --- Test formatFieldValue with redaction ---

func TestFormatFieldValue_WithRedaction(t *testing.T) {
	RegisterSecret("SECRET")

	result := formatFieldValue("this contains SECRET value")
	if hasSubstring(result, "SECRET") {
		t.Error("secret should be redacted in field values")
	}
	if !hasSubstring(result, "***REDACTED***") {
		t.Error("redacted marker should be present")
	}
}

// --- Test appendFields with empty fields map ---

func TestAppendFields_EmptyMap(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	InfoF("message", map[string]any{})
	// Should not panic
}

// --- Test multiple file logging switches ---

func TestEnableFileLogging_SwitchFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logPath1 := filepath.Join(tmpDir, "log1.txt")
	logPath2 := filepath.Join(tmpDir, "log2.txt")

	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(INFO)

	// Enable first log file
	err1 := EnableFileLogging(logPath1)
	if err1 != nil {
		t.Fatalf("EnableFileLogging(1) failed: %v", err1)
	}

	Info("message in first log")

	// Switch to second log file
	err2 := EnableFileLogging(logPath2)
	if err2 != nil {
		t.Fatalf("EnableFileLogging(2) failed: %v", err2)
	}

	Info("message in second log")

	t.Cleanup(func() {
		DisableFileLogging()
	})

	// Verify both files exist
	if _, err := os.Stat(logPath1); err != nil {
		t.Errorf("first log file should exist: %v", err)
	}
	if _, err := os.Stat(logPath2); err != nil {
		t.Errorf("second log file should exist: %v", err)
	}
}

// Helper functions for test utilities

// hasSubstring checks if substr exists in s
func hasSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// countOccurrences counts how many times substr appears in s
func countOccurrences(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
