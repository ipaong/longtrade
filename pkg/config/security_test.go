package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSensitiveDataReplacer_Basic(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     10,
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("sk-ant-1234567890"),
			},
		},
	}

	replacer := cfg.SensitiveDataReplacer()
	if replacer == nil {
		t.Errorf("Expected replacer, got nil")
	}
}

func TestSensitiveDataReplacer_Caching(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     10,
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("sk-ant-test"),
			},
		},
	}

	// Call twice, should get same instance (cached)
	replacer1 := cfg.SensitiveDataReplacer()
	replacer2 := cfg.SensitiveDataReplacer()

	if replacer1 != replacer2 {
		t.Errorf("Expected same replacer instance (caching), got different instances")
	}
}

func TestFilterSensitiveData_Disabled(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: false,
		},
	}

	content := "my secret key is sk-ant-1234"
	result := cfg.FilterSensitiveData(content)

	if result != content {
		t.Errorf("Expected unchanged content when filtering disabled")
	}
}

func TestFilterSensitiveData_TooShort(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     100,
		},
	}

	content := "short"
	result := cfg.FilterSensitiveData(content)

	if result != content {
		t.Errorf("Expected unchanged content when below min length")
	}
}

func TestFilterSensitiveData_WithSecret(t *testing.T) {
	secret := "sk-ant-secretkey1234567890"
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     10,
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString(secret),
			},
		},
	}

	content := "The API key is " + secret + " please keep it safe"
	result := cfg.FilterSensitiveData(content)

	if strings.Contains(result, secret) {
		t.Errorf("Expected secret to be filtered")
	}
	if !strings.Contains(result, "[FILTERED]") {
		t.Errorf("Expected [FILTERED] marker in result")
	}
}

func TestSecurityCopyFrom_FileNotExists(t *testing.T) {
	cfg := &Config{}
	err := cfg.SecurityCopyFrom("/nonexistent/path/config.yml")

	// Should not error for missing file
	if err != nil {
		t.Errorf("Expected no error for missing security config: %v", err)
	}
}

func TestSecurityCopyFrom_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create config file
	if err := os.WriteFile(configPath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create security file
	securityPath := filepath.Join(tmpDir, ".security.yml")
	securityContent := `
channels:
  telegram:
    token: "secret-token-123"
`
	if err := os.WriteFile(securityPath, []byte(securityContent), 0600); err != nil {
		t.Fatalf("Failed to create security file: %v", err)
	}

	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{},
		},
	}

	err := cfg.SecurityCopyFrom(configPath)
	if err != nil {
		t.Fatalf("SecurityCopyFrom failed: %v", err)
	}
}

func TestInitSensitiveCache_LazyInitialization(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     10,
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("sk-ant-test"),
			},
		},
	}

	// Cache should be nil initially
	if cfg.sensitiveCache != nil {
		t.Errorf("Expected nil sensitive cache initially")
	}

	// Initialize cache
	cfg.initSensitiveCache()

	// Now it should be initialized
	if cfg.sensitiveCache == nil {
		t.Errorf("Expected sensitive cache to be initialized")
	}
	if cfg.sensitiveCache.replacer == nil {
		t.Errorf("Expected replacer to be initialized")
	}
}

func TestInitSensitiveCache_NoSecrets(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     10,
		},
	}

	cfg.initSensitiveCache()

	if cfg.sensitiveCache == nil {
		t.Errorf("Expected cache to be created even without secrets")
	}
	if cfg.sensitiveCache.replacer == nil {
		t.Errorf("Expected replacer to exist")
	}
}

func TestCollectSensitiveValues_SingleSecret(t *testing.T) {
	secret := "my-secret-token"
	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString(secret),
			},
		},
	}

	values := cfg.collectSensitiveValues()
	if len(values) == 0 {
		t.Fatalf("Expected at least one secret, got %d", len(values))
	}

	found := false
	for _, v := range values {
		if v == secret {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find secret %q in collected values", secret)
	}
}

func TestCollectSensitiveValues_MultipleSecrets(t *testing.T) {
	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("token1"),
			},
			Feishu: FeishuConfig{
				AppSecret: *NewSecureString("secret1"),
			},
		},
	}

	values := cfg.collectSensitiveValues()
	if len(values) < 2 {
		t.Errorf("Expected at least 2 secrets, got %d", len(values))
	}
}

func TestCollectSensitiveValues_EmptySecrets(t *testing.T) {
	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString(""),
			},
		},
	}

	values := cfg.collectSensitiveValues()
	// Empty secrets should not be collected
	for _, v := range values {
		if v == "" {
			t.Errorf("Expected empty strings to be filtered out")
		}
	}
}

func TestCollectSensitive_WithSecureStrings(t *testing.T) {
	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("token123"),
			},
		},
	}

	values := cfg.collectSensitiveValues()
	found := false
	for _, v := range values {
		if v == "token123" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find telegram token in collected values")
	}
}

func TestLoadSecurityConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	secPath := filepath.Join(tmpDir, ".security.yml")

	// Write invalid YAML
	if err := os.WriteFile(secPath, []byte("invalid: [yaml:"), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &Config{}
	err := loadSecurityConfig(cfg, secPath)

	if err == nil {
		t.Errorf("Expected error for invalid YAML")
	}
}

func TestLoadSecurityConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	secPath := filepath.Join(tmpDir, ".security.yml")

	// Write empty file
	if err := os.WriteFile(secPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &Config{}
	err := loadSecurityConfig(cfg, secPath)

	// Empty file should be okay
	if err != nil {
		t.Errorf("Expected no error for empty file: %v", err)
	}
}

func TestSaveSecurityConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	secPath := filepath.Join(tmpDir, ".security.yml")

	cfg := &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("sk-ant-test"),
			},
		},
	}

	err := saveSecurityConfig(secPath, cfg)
	if err != nil {
		t.Fatalf("saveSecurityConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(secPath); err != nil {
		t.Errorf("Expected security file to be created: %v", err)
	}

	// Check permissions
	info, err := os.Stat(secPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("Expected 0o600 permissions, got %o", info.Mode().Perm())
	}
}

func TestSecurityCopyFrom_EmptyPath(t *testing.T) {
	cfg := &Config{}
	// Empty path resolves to ./.security.yml which likely doesn't exist — no error
	err := cfg.SecurityCopyFrom("")
	if err != nil {
		t.Errorf("Expected no error for empty path (file missing is not an error): %v", err)
	}
}

func TestCollectSensitive_NestedStructure(t *testing.T) {
	cfg := &Config{
		Channels: ChannelsConfig{
			Feishu: FeishuConfig{
				AppSecret:  *NewSecureString("secret1"),
				EncryptKey: *NewSecureString("key1"),
			},
		},
	}

	values := cfg.collectSensitiveValues()
	if len(values) < 2 {
		t.Errorf("Expected to collect from nested structures")
	}
}

func TestSensitiveDataReplacer_ShortValues(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     5,
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Token: *NewSecureString("x"), // Very short
			},
		},
	}

	// "x" is too short to filter (< 3 chars)
	content := "key is x test"
	result := cfg.FilterSensitiveData(content)

	if result != content {
		t.Errorf("Expected unchanged for short secrets")
	}
}
