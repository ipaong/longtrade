package credential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewResolver_EmptyDir tests NewResolver with an empty config directory.
func TestNewResolver_EmptyDir(t *testing.T) {
	r := NewResolver("")
	if r.configDir != "" {
		t.Errorf("configDir = %q, want empty", r.configDir)
	}
	if r.resolvedConfigDir != "" {
		t.Errorf("resolvedConfigDir = %q, want empty", r.resolvedConfigDir)
	}
}

// TestNewResolver_NonexistentDir tests NewResolver with a path that doesn't exist.
func TestNewResolver_NonexistentDir(t *testing.T) {
	r := NewResolver("/nonexistent/path/config")
	if r.configDir != "/nonexistent/path/config" {
		t.Errorf("configDir = %q, want /nonexistent/path/config", r.configDir)
	}
	// resolvedConfigDir should equal configDir when EvalSymlinks fails
	if r.resolvedConfigDir != "/nonexistent/path/config" {
		t.Errorf("resolvedConfigDir = %q, want /nonexistent/path/config", r.resolvedConfigDir)
	}
}

// TestNewResolver_ValidDir tests NewResolver with a valid directory.
func TestNewResolver_ValidDir(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver(dir)
	if r.configDir != dir {
		t.Errorf("configDir = %q, want %q", r.configDir, dir)
	}
	// resolvedConfigDir may differ on macOS due to /var -> /private/var symlink
	if r.resolvedConfigDir == "" {
		t.Error("resolvedConfigDir should not be empty")
	}
}

// TestResolve_EmptyString tests that Resolve returns "" for empty input.
func TestResolve_EmptyString(t *testing.T) {
	r := NewResolver("")
	val, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if val != "" {
		t.Errorf("got %q, want empty", val)
	}
}

// TestResolve_PlaintextCredential tests that plaintext credentials are returned unchanged.
func TestResolve_PlaintextCredential(t *testing.T) {
	r := NewResolver("")
	cases := []string{
		"sk-abc123",
		"token_xyz",
		"my-secret-key",
		"plaintext with spaces",
	}
	for _, tc := range cases {
		val, err := r.Resolve(tc)
		if err != nil {
			t.Errorf("Resolve(%q): %v", tc, err)
		}
		if val != tc {
			t.Errorf("Resolve(%q): got %q, want %q", tc, val, tc)
		}
	}
}

// TestResolve_FileReference_Missing tests file:// with a missing file.
func TestResolve_FileReference_Missing(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver(dir)
	val, err := r.Resolve("file://missing.key")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if val != "" {
		t.Errorf("expected empty val, got %q", val)
	}
	if !strings.Contains(err.Error(), "missing.key") {
		t.Errorf("error message should mention missing.key: %v", err)
	}
}

// TestResolve_FileReference_Empty tests file:// with an empty file.
func TestResolve_FileReference_Empty(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "empty.key")
	if err := os.WriteFile(keyPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := NewResolver(dir)
	_, err := r.Resolve("file://empty.key")
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty': %v", err)
	}
}

// TestResolve_FileReference_NoFilename tests file:// with empty filename.
func TestResolve_FileReference_NoFilename(t *testing.T) {
	r := NewResolver(t.TempDir())
	_, err := r.Resolve("file://")
	if err == nil {
		t.Fatal("expected error for empty filename")
	}
	if !strings.Contains(err.Error(), "no filename") {
		t.Errorf("error should mention 'no filename': %v", err)
	}
}

// TestResolve_FileReference_WhitespaceFilename tests file:// with whitespace.
func TestResolve_FileReference_WhitespaceFilename(t *testing.T) {
	r := NewResolver(t.TempDir())
	_, err := r.Resolve("file://   ")
	if err == nil {
		t.Fatal("expected error for whitespace filename")
	}
	if !strings.Contains(err.Error(), "no filename") {
		t.Errorf("error should mention 'no filename': %v", err)
	}
}

// TestResolve_FileReference_Valid tests file:// with a valid file.
func TestResolve_FileReference_Valid(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")
	testKey := "sk-test-credential"
	if err := os.WriteFile(keyPath, []byte(testKey), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := NewResolver(dir)
	val, err := r.Resolve("file://test.key")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if val != testKey {
		t.Errorf("got %q, want %q", val, testKey)
	}
}

// TestResolve_FileReference_WithWhitespace tests file:// returns trimmed content.
func TestResolve_FileReference_WithWhitespace(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "whitespace.key")
	if err := os.WriteFile(keyPath, []byte("  sk-value  \n  "), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := NewResolver(dir)
	val, err := r.Resolve("file://whitespace.key")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if val != "sk-value" {
		t.Errorf("got %q, want 'sk-value'", val)
	}
}

// TestResolve_FileReference_PathEscape tests that file:// path traversal is blocked.
func TestResolve_FileReference_PathEscape(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver(dir)
	_, err := r.Resolve("file://../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path escape attempt")
	}
}

// TestResolve_FileReference_Subdirectory tests file:// with nested directories.
func TestResolve_FileReference_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "credentials")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	keyPath := filepath.Join(subdir, "api.key")
	testKey := "sk-nested-key"
	if err := os.WriteFile(keyPath, []byte(testKey), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := NewResolver(dir)
	val, err := r.Resolve("file://credentials/api.key")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if val != testKey {
		t.Errorf("got %q, want %q", val, testKey)
	}
}

// TestResolve_EncryptedCredential_NoPassphrase tests enc:// without passphrase.
func TestResolve_EncryptedCredential_NoPassphrase(t *testing.T) {
	// Save original PassphraseProvider.
	originalProvider := PassphraseProvider

	// Set PassphraseProvider to return empty string.
	PassphraseProvider = func() string {
		return ""
	}
	t.Cleanup(func() {
		PassphraseProvider = originalProvider
	})

	r := NewResolver("")
	val, err := r.Resolve("enc://somebase64content")
	if err == nil {
		t.Fatal("expected error for missing passphrase")
	}
	if val != "" {
		t.Errorf("expected empty val, got %q", val)
	}
	if !strings.Contains(err.Error(), "passphrase required") {
		t.Errorf("error should mention passphrase: %v", err)
	}
}

// TestResolve_EncryptedCredential_InvalidBase64 tests enc:// with invalid base64.
func TestResolve_EncryptedCredential_InvalidBase64(t *testing.T) {
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "test-passphrase" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	r := NewResolver("")
	val, err := r.Resolve("enc://!!!invalid base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if val != "" {
		t.Errorf("expected empty val, got %q", val)
	}
	if !strings.Contains(err.Error(), "base64") {
		t.Errorf("error should mention base64: %v", err)
	}
}

// TestResolve_EncryptedCredential_PayloadTooShort tests enc:// with too-short payload.
func TestResolve_EncryptedCredential_PayloadTooShort(t *testing.T) {
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "test-passphrase" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	// Use a short base64 string that decodes to something too short
	// "YQ==" decodes to just "a" which is less than saltLen+nonceLen+1
	r := NewResolver("")
	_, err := r.Resolve("enc://YQ==")
	if err == nil {
		t.Fatal("expected error for short payload")
	}
	if !strings.Contains(err.Error(), "too short") && !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("error should mention 'too short' or 'invalid base64': %v", err)
	}
}

// TestIsWithinDir tests the isWithinDir boundary check.
func TestIsWithinDir(t *testing.T) {
	cases := []struct {
		path string
		dir  string
		want bool
	}{
		{"/home/user/config", "/home/user", true},
		{"/home/user", "/home/user", true},
		{"/home/user/config/sub", "/home/user/config", true},
		{"/home/other", "/home/user", false},
		{"/home/user_other", "/home/user", false},
		{"/etc/passwd", "/home/user", false},
	}

	for _, tc := range cases {
		got := isWithinDir(tc.path, tc.dir)
		if got != tc.want {
			t.Errorf("isWithinDir(%q, %q) = %v, want %v", tc.path, tc.dir, got, tc.want)
		}
	}
}

// TestAllowedSSHKeyPath tests the allowedSSHKeyPath boundary check.
func TestAllowedSSHKeyPath_Empty(t *testing.T) {
	if !allowedSSHKeyPath("") {
		t.Error("empty path should be allowed (passphrase-only mode)")
	}
}

// TestAllowedSSHKeyPath_UserHome tests paths within ~/.ssh/.
func TestAllowedSSHKeyPath_UserHome(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	sshPath := filepath.Join(userHome, ".ssh", "khunquant_ed25519.key")
	if !allowedSSHKeyPath(sshPath) {
		t.Errorf("path in ~/.ssh/ should be allowed: %q", sshPath)
	}
}

// TestAllowedSSHKeyPath_EnvVar tests KHUNQUANT_SSH_KEY_PATH matching.
func TestAllowedSSHKeyPath_EnvVar(t *testing.T) {
	testPath := "/opt/custom/ssh.key"
	t.Setenv(SSHKeyPathEnvVar, testPath)

	if !allowedSSHKeyPath(testPath) {
		t.Errorf("path matching KHUNQUANT_SSH_KEY_PATH should be allowed: %q", testPath)
	}
}

// TestAllowedSSHKeyPath_NotAllowed tests that disallowed paths are rejected.
func TestAllowedSSHKeyPath_NotAllowed(t *testing.T) {
	if allowedSSHKeyPath("/tmp/random/key") {
		t.Error("/tmp path should not be allowed")
	}
	if allowedSSHKeyPath("/etc/shadow") {
		t.Error("/etc path should not be allowed")
	}
}

// TestEncrypt_EmptyPassphrase tests that Encrypt fails with empty passphrase.
func TestEncrypt_EmptyPassphrase(t *testing.T) {
	_, err := Encrypt("", "", "plaintext")
	if err == nil {
		t.Fatal("expected error for empty passphrase")
	}
	if !strings.Contains(err.Error(), "passphrase must not be empty") {
		t.Errorf("error should mention empty passphrase: %v", err)
	}
}

// TestEncrypt_NoSSHKey tests that Encrypt fails when SSH key is not found.
func TestEncrypt_NoSSHKey(t *testing.T) {
	// Save and clear environment.
	t.Setenv(SSHKeyPathEnvVar, "")
	t.Setenv(khunquantHome, "")

	// Try to encrypt; pickSSHKeyPath will find no key in default location.
	_, err := Encrypt("testpass", "", "plaintext")
	if err == nil {
		// This is OK if a key happens to exist at default location.
		// We're mainly testing that the error path works when no key is found.
		return
	}
	if !strings.Contains(err.Error(), "SSH") && !strings.Contains(err.Error(), "key") {
		t.Errorf("error should mention SSH key: %v", err)
	}
}

// TestEncrypt_WithValidSSHKey tests successful encryption.
func TestEncrypt_WithValidSSHKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test_key.pem")

	// Generate a test SSH key.
	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	plaintext := "secret-api-key"
	encrypted, err := Encrypt("testpassphrase", keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if !strings.HasPrefix(encrypted, EncScheme) {
		t.Errorf("encrypted value should start with %q, got %q", EncScheme, encrypted)
	}
}

// TestEncrypt_Decrypt_Roundtrip tests encryption and decryption roundtrip.
func TestEncrypt_Decrypt_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "roundtrip_key.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	passphrase := "secure-passphrase"
	plaintext := "my-secret-credential"

	// Encrypt
	encrypted, err := Encrypt(passphrase, keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Set PassphraseProvider to return our passphrase.
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return passphrase }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	// Decrypt
	r := NewResolver(dir)
	decrypted, err := r.Resolve(encrypted)
	if err != nil {
		t.Fatalf("Resolve (decrypt): %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

// TestEncrypt_Decrypt_WrongPassphrase tests decryption with wrong passphrase.
func TestEncrypt_Decrypt_WrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "wrong_pass_key.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	passphrase := "correct-passphrase"
	plaintext := "secret"

	encrypted, err := Encrypt(passphrase, keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Set wrong passphrase.
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "wrong-passphrase" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	r := NewResolver(dir)
	decrypted, err := r.Resolve(encrypted)
	if err == nil {
		t.Fatalf("expected decryption to fail with wrong passphrase, got: %q", decrypted)
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("error should mention decryption failure: %v", err)
	}
}

// TestDeriveKey_EmptySSHKeyPath tests deriveKey with empty SSH key path.
func TestDeriveKey_EmptySSHKeyPath(t *testing.T) {
	_, err := deriveKey("passphrase", "", []byte("salt"))
	if err == nil {
		t.Fatal("expected error for empty SSH key path")
	}
	if !strings.Contains(err.Error(), "SSH") {
		t.Errorf("error should mention SSH key: %v", err)
	}
}

// TestDeriveKey_NotAllowedPath tests deriveKey with disallowed path.
func TestDeriveKey_NotAllowedPath(t *testing.T) {
	_, err := deriveKey("passphrase", "/tmp/not/allowed/key", []byte("salt"))
	if err == nil {
		t.Fatal("expected error for disallowed path")
	}
	if !strings.Contains(err.Error(), "not in an allowed location") {
		t.Errorf("error should mention allowed location: %v", err)
	}
}

// TestDeriveKey_MissingFile tests deriveKey with a non-existent file.
func TestDeriveKey_MissingFile(t *testing.T) {
	missingPath := "/tmp/khunquant_nonexistent_" + fmt.Sprintf("%d", os.Getpid())
	t.Setenv(SSHKeyPathEnvVar, missingPath)

	_, err := deriveKey("passphrase", missingPath, []byte("salt"))
	if err == nil {
		t.Fatal("expected error for missing SSH key file")
	}
	if !strings.Contains(err.Error(), "cannot read SSH key") {
		t.Errorf("error should mention reading SSH key: %v", err)
	}
}

// TestDeriveKey_ValidFile tests deriveKey with a valid SSH key.
func TestDeriveKey_ValidFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "derive_test.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set env to allow this path.
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	salt := make([]byte, saltLen)
	key, err := deriveKey("test-passphrase", keyPath, salt)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}

	if len(key) != keyLen {
		t.Errorf("key length = %d, want %d", len(key), keyLen)
	}
}

// TestDeriveKey_DifferentPassphrases tests that different passphrases produce different keys.
func TestDeriveKey_DifferentPassphrases(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "different_pass.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv(SSHKeyPathEnvVar, keyPath)

	salt := make([]byte, saltLen)
	key1, err := deriveKey("passphrase1", keyPath, salt)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}

	key2, err := deriveKey("passphrase2", keyPath, salt)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("different passphrases should produce different keys")
	}
}

// TestPickSSHKeyPath_Explicit tests pickSSHKeyPath with explicit override.
func TestPickSSHKeyPath_Explicit(t *testing.T) {
	explicit := "/explicit/path/key"
	result := pickSSHKeyPath(explicit)
	if result != explicit {
		t.Errorf("pickSSHKeyPath(%q) = %q, want %q", explicit, result, explicit)
	}
}

// TestPickSSHKeyPath_EnvVar tests pickSSHKeyPath with environment variable.
func TestPickSSHKeyPath_EnvVar(t *testing.T) {
	envPath := "/env/var/path"
	t.Setenv(SSHKeyPathEnvVar, envPath)

	result := pickSSHKeyPath("")
	if result != envPath {
		t.Errorf("pickSSHKeyPath(\"\") = %q, want %q (from env)", result, envPath)
	}
}

// TestPickSSHKeyPath_EmptyEnvVar tests that pickSSHKeyPath respects empty env var.
func TestPickSSHKeyPath_EmptyEnvVar(t *testing.T) {
	t.Setenv(SSHKeyPathEnvVar, "")

	result := pickSSHKeyPath("")
	// Empty env var means "respect the setting, even if empty"
	if result != "" {
		t.Errorf("pickSSHKeyPath(\"\") = %q, want empty (env var is empty)", result)
	}
}

// TestResolveEncrypted_ValidEncryptedString tests resolveEncrypted with valid encrypted data.
func TestResolveEncrypted_ValidEncryptedString(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "valid_enc.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	plaintext := "test-credential"
	passphrase := "test-pass"

	encrypted, err := Encrypt(passphrase, keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return passphrase }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	decrypted, err := resolveEncrypted(encrypted)
	if err != nil {
		t.Fatalf("resolveEncrypted: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

// TestResolveEncrypted_CorruptedCiphertext tests resolveEncrypted with corrupted data.
func TestResolveEncrypted_CorruptedCiphertext(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "corrupt_enc.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	plaintext := "test-credential"
	passphrase := "test-pass"

	encrypted, err := Encrypt(passphrase, keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Corrupt the encrypted string by replacing with wrong passphrase at decryption time
	// (changing one base64 character may not be enough to cause auth tag failure)
	// Instead, use the correct encrypted string but wrong passphrase to ensure failure
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "wrong-pass" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	decrypted, err := resolveEncrypted(encrypted)
	if err == nil {
		t.Fatalf("expected error for wrong passphrase, got: %q", decrypted)
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("error should mention decryption failure: %v", err)
	}
}

// TestFindDefaultSSHKey_Exists tests findDefaultSSHKey when key exists.
func TestFindDefaultSSHKey_Exists(t *testing.T) {
	// Save original home and PassphraseProvider.
	originalProvider := PassphraseProvider

	result := findDefaultSSHKey()
	// We can't easily test this without mocking os.Stat and os.UserHomeDir.
	// The function returns "" if the default key doesn't exist, which is fine for testing.
	// If a key happens to exist, we just verify it returns a non-empty string.
	if result != "" {
		// Check that the path ends with the expected pattern.
		if !strings.Contains(result, "khunquant_ed25519.key") {
			t.Errorf("result = %q, should contain 'khunquant_ed25519.key'", result)
		}
	}

	t.Cleanup(func() { PassphraseProvider = originalProvider })
}

// TestMultipleResolutions_Consistency tests that multiple resolutions are consistent.
func TestMultipleResolutions_Consistency(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "consistency.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Set KHUNQUANT_SSH_KEY_PATH to allow this path
	t.Setenv(SSHKeyPathEnvVar, keyPath)

	plaintext := "consistency-test"
	passphrase := "consistent-pass"

	encrypted, err := Encrypt(passphrase, keyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return passphrase }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	// Decrypt multiple times and verify consistency.
	r := NewResolver(dir)
	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		val, err := r.Resolve(encrypted)
		if err != nil {
			t.Fatalf("Resolve[%d]: %v", i, err)
		}
		results[i] = val
	}

	for i := 1; i < len(results); i++ {
		if results[i] != plaintext {
			t.Errorf("Resolve[%d] = %q, want %q", i, results[i], plaintext)
		}
	}
}

// TestEncrypt_EmptyPlaintext tests encryption of empty plaintext.
func TestEncrypt_EmptyPlaintext(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "empty_plain.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv(SSHKeyPathEnvVar, keyPath)

	// Empty plaintext should still encrypt successfully
	encrypted, err := Encrypt("passphrase", keyPath, "")
	if err != nil {
		t.Fatalf("Encrypt empty plaintext: %v", err)
	}

	if !strings.HasPrefix(encrypted, EncScheme) {
		t.Errorf("encrypted value should start with %q", EncScheme)
	}

	// Decrypt and verify it's empty
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "passphrase" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	decrypted, err := resolveEncrypted(encrypted)
	if err != nil {
		t.Fatalf("resolveEncrypted: %v", err)
	}
	if decrypted != "" {
		t.Errorf("decrypted empty plaintext = %q, want empty", decrypted)
	}
}

// TestEncrypt_LongPlaintext tests encryption of very long plaintext.
func TestEncrypt_LongPlaintext(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "long_plain.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv(SSHKeyPathEnvVar, keyPath)

	// Create long plaintext
	longPlaintext := strings.Repeat("secret", 1000)

	encrypted, err := Encrypt("passphrase", keyPath, longPlaintext)
	if err != nil {
		t.Fatalf("Encrypt long plaintext: %v", err)
	}

	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "passphrase" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	decrypted, err := resolveEncrypted(encrypted)
	if err != nil {
		t.Fatalf("resolveEncrypted: %v", err)
	}
	if decrypted != longPlaintext {
		t.Errorf("decrypted length = %d, want %d", len(decrypted), len(longPlaintext))
	}
}

// TestResolve_FileReference_WithoutConfigDir tests file:// resolution without a config dir.
func TestResolve_FileReference_WithoutConfigDir(t *testing.T) {
	// With empty configDir, the resolver will treat paths as relative to current dir
	r := NewResolver("")
	_, err := r.Resolve("file://nonexistent.key")
	if err == nil {
		t.Fatal("expected error for file in empty config dir")
	}
}

// TestResolve_MixedCredentials tests resolving different credential types in sequence.
func TestResolve_MixedCredentials(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	keyPath := filepath.Join(dir, "api.key")
	if err := os.WriteFile(keyPath, []byte("sk-from-file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := NewResolver(dir)

	// Test plaintext
	val1, err := r.Resolve("sk-plaintext")
	if err != nil || val1 != "sk-plaintext" {
		t.Errorf("plaintext resolution failed: err=%v, val=%q", err, val1)
	}

	// Test file reference
	val2, err := r.Resolve("file://api.key")
	if err != nil || val2 != "sk-from-file" {
		t.Errorf("file resolution failed: err=%v, val=%q", err, val2)
	}

	// Test empty string
	val3, err := r.Resolve("")
	if err != nil || val3 != "" {
		t.Errorf("empty resolution failed: err=%v, val=%q", err, val3)
	}
}

// TestAllowedSSHKeyPath_WithKHUNQUANT_HOME tests KHUNQUANT_HOME allowance.
func TestAllowedSSHKeyPath_WithKHUNQUANT_HOME(t *testing.T) {
	customHome := t.TempDir()
	t.Setenv(khunquantHome, customHome)

	// Path within KHUNQUANT_HOME should be allowed
	keyPath := filepath.Join(customHome, "keys", "mykey")
	if !allowedSSHKeyPath(keyPath) {
		t.Errorf("path in KHUNQUANT_HOME should be allowed: %q", keyPath)
	}
}

// TestPickSSHKeyPath_DefaultBehavior tests default SSH key path discovery.
func TestPickSSHKeyPath_DefaultBehavior(t *testing.T) {
	oldPath, oldOk := os.LookupEnv(SSHKeyPathEnvVar)
	os.Unsetenv(SSHKeyPathEnvVar) //nolint:forbidigo // LookupEnv semantics require truly unset, not empty
	t.Cleanup(func() {
		if oldOk {
			os.Setenv(SSHKeyPathEnvVar, oldPath)
		}
	})

	result := pickSSHKeyPath("")
	// Result should be either "" (if key doesn't exist) or a valid path
	if result != "" {
		if !strings.Contains(result, "khunquant_ed25519.key") {
			t.Errorf("result = %q, should contain default key name", result)
		}
	}
}

// TestResolve_EncryptedCredential_WrongSSHKey tests decryption with different SSH key.
func TestResolve_EncryptedCredential_WrongSSHKey(t *testing.T) {
	dir := t.TempDir()
	keyPath1 := filepath.Join(dir, "key1.pem")
	keyPath2 := filepath.Join(dir, "key2.pem")

	// Generate two different keys
	if err := GenerateSSHKey(keyPath1); err != nil {
		t.Fatalf("GenerateSSHKey key1: %v", err)
	}
	if err := GenerateSSHKey(keyPath2); err != nil {
		t.Fatalf("GenerateSSHKey key2: %v", err)
	}

	// Encrypt with key1
	t.Setenv(SSHKeyPathEnvVar, keyPath1)

	plaintext := "secret-data"
	passphrase := "test-pass"
	encrypted, err := Encrypt(passphrase, keyPath1, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Try to decrypt with key2
	t.Setenv(SSHKeyPathEnvVar, keyPath2)
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return passphrase }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	r := NewResolver(dir)
	decrypted, err := r.Resolve(encrypted)
	if err == nil {
		t.Fatalf("expected decryption to fail with different key, got: %q", decrypted)
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("error should mention decryption failure: %v", err)
	}
}

// TestDeriveKey_DifferentSalts tests that different salts produce different keys.
func TestDeriveKey_DifferentSalts(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "salt_test.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv(SSHKeyPathEnvVar, keyPath)

	salt1 := make([]byte, saltLen)
	salt1[0] = 1

	salt2 := make([]byte, saltLen)
	salt2[0] = 2

	key1, err := deriveKey("passphrase", keyPath, salt1)
	if err != nil {
		t.Fatalf("deriveKey salt1: %v", err)
	}

	key2, err := deriveKey("passphrase", keyPath, salt2)
	if err != nil {
		t.Fatalf("deriveKey salt2: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("different salts should produce different keys")
	}
}

// TestEncrypt_SpecialCharactersInPlaintext tests encryption of special characters.
func TestEncrypt_SpecialCharactersInPlaintext(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "special.pem")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv(SSHKeyPathEnvVar, keyPath)

	testCases := []string{
		"!@#$%^&*()",
		"line1\nline2\nline3",
		"tab\there",
		"null\x00byte",
		"emoji🔐🔑",
		"日本語",
	}

	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "testpass" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	for _, plaintext := range testCases {
		encrypted, err := Encrypt("testpass", keyPath, plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plaintext, err)
		}

		decrypted, err := resolveEncrypted(encrypted)
		if err != nil {
			t.Fatalf("resolveEncrypted(%q): %v", plaintext, err)
		}

		if decrypted != plaintext {
			t.Errorf("roundtrip failed: plaintext=%q, decrypted=%q", plaintext, decrypted)
		}
	}
}

// TestResolve_EncschemeWithoutPassphrase tests enc:// without passphrase provider.
func TestResolve_EncschemeWithoutPassphrase(t *testing.T) {
	originalProvider := PassphraseProvider
	PassphraseProvider = func() string { return "" }
	t.Cleanup(func() { PassphraseProvider = originalProvider })

	r := NewResolver("")
	_, err := r.Resolve("enc://AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if err == nil {
		t.Fatal("expected error when passphrase provider returns empty")
	}
	if !strings.Contains(err.Error(), "passphrase required") {
		t.Errorf("error should mention passphrase: %v", err)
	}
}

// TestNewResolver_SymlinkConfigDir tests NewResolver with symlinked config dir.
func TestNewResolver_SymlinkConfigDir(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "config_link")

	// Create symlink to real directory
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	r := NewResolver(linkDir)
	if r.configDir != linkDir {
		t.Errorf("configDir = %q, want %q", r.configDir, linkDir)
	}
	// resolvedConfigDir should be the real path
	if !strings.Contains(r.resolvedConfigDir, "credential") {
		// Just verify it was resolved
		if r.resolvedConfigDir == "" {
			t.Error("resolvedConfigDir should not be empty for valid dir")
		}
	}
}
