package credential

import (
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// TestDefaultSSHKeyPath tests DefaultSSHKeyPath returns valid path.
func TestDefaultSSHKeyPath(t *testing.T) {
	path, err := DefaultSSHKeyPath()
	if err != nil {
		t.Fatalf("DefaultSSHKeyPath: %v", err)
	}

	if path == "" {
		t.Fatal("path should not be empty")
	}

	// Should contain .ssh and khunquant_ed25519.key
	if !contains(path, ".ssh") {
		t.Errorf("path should contain '.ssh': %q", path)
	}
	if !contains(path, "khunquant_ed25519.key") {
		t.Errorf("path should contain 'khunquant_ed25519.key': %q", path)
	}
}

// TestGenerateSSHKey_CreateFiles tests that GenerateSSHKey creates both key files.
func TestGenerateSSHKey_CreateFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test_key")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Check private key exists
	privInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Stat private key: %v", err)
	}
	if !privInfo.Mode().IsRegular() {
		t.Errorf("private key should be a regular file")
	}
	// Check permissions are 0600
	if privInfo.Mode().Perm() != 0o600 {
		t.Errorf("private key permissions = %o, want 0o600", privInfo.Mode().Perm())
	}

	// Check public key exists
	pubPath := keyPath + ".pub"
	pubInfo, err := os.Stat(pubPath)
	if err != nil {
		t.Fatalf("Stat public key: %v", err)
	}
	if !pubInfo.Mode().IsRegular() {
		t.Errorf("public key should be a regular file")
	}
	// Check permissions are 0644
	if pubInfo.Mode().Perm() != 0o644 {
		t.Errorf("public key permissions = %o, want 0o644", pubInfo.Mode().Perm())
	}
}

// TestGenerateSSHKey_ValidPEM tests that the generated private key is valid PEM.
func TestGenerateSSHKey_ValidPEM(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "pem_test")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	privData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	block, _ := pem.Decode(privData)
	if block == nil {
		t.Fatal("private key is not valid PEM")
	}

	if block.Type != "OPENSSH PRIVATE KEY" {
		t.Errorf("PEM block type = %q, want 'OPENSSH PRIVATE KEY'", block.Type)
	}
}

// TestGenerateSSHKey_ValidPublicKey tests that the public key is valid.
func TestGenerateSSHKey_ValidPublicKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "pubkey_test")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	pubData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Parse the authorized_keys line
	key, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}

	// Should be Ed25519
	if key.Type() != "ssh-ed25519" {
		t.Errorf("key type = %q, want 'ssh-ed25519'", key.Type())
	}
}

// TestGenerateSSHKey_CreatesDirectory tests that GenerateSSHKey creates parent directory.
func TestGenerateSSHKey_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "nested", "sub", "dir", "key")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Check that the key file exists
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file should exist: %v", err)
	}

	// Check that the directory was created with correct permissions
	dirPath := filepath.Dir(keyPath)
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("Stat directory: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("parent should be a directory")
	}
	// Check directory permissions are 0700
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("directory permissions = %o, want 0o700", dirInfo.Mode().Perm())
	}
}

// TestGenerateSSHKey_Overwrite tests that GenerateSSHKey overwrites existing files.
func TestGenerateSSHKey_Overwrite(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "overwrite_test")

	// Generate first key
	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey (first): %v", err)
	}

	privData1, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile (first): %v", err)
	}

	// Generate second key (should overwrite)
	err = GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey (second): %v", err)
	}

	privData2, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile (second): %v", err)
	}

	// The two keys should be different (extremely unlikely to be the same)
	if string(privData1) == string(privData2) {
		t.Error("overwritten key should be different from first key")
	}
}

// TestGenerateSSHKey_BothFiles tests that both private and public files are created.
func TestGenerateSSHKey_BothFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "both_files")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	privData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile private: %v", err)
	}

	pubData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatalf("ReadFile public: %v", err)
	}

	if len(privData) == 0 {
		t.Error("private key data should not be empty")
	}
	if len(pubData) == 0 {
		t.Error("public key data should not be empty")
	}

	// Private key should be larger than public key (generally true for Ed25519)
	if len(privData) <= len(pubData) {
		t.Logf("note: private key (%d bytes) is not larger than public key (%d bytes)", len(privData), len(pubData))
	}
}

// TestGenerateSSHKey_PrivateKeyContentIsCorrect tests that private key can be used.
func TestGenerateSSHKey_PrivateKeyContentIsCorrect(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "valid_private")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	privData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Decode and parse the private key
	block, _ := pem.Decode(privData)
	if block == nil {
		t.Fatal("invalid PEM format")
	}

	privKey, err := ssh.ParsePrivateKey(privData)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	if privKey == nil {
		t.Fatal("private key should not be nil")
	}

	// Verify the public key type
	if privKey.PublicKey().Type() != "ssh-ed25519" {
		t.Errorf("public key type = %q, want 'ssh-ed25519'", privKey.PublicKey().Type())
	}
}

// TestDefaultSSHKeyPath_NoHome tests DefaultSSHKeyPath when home cannot be determined.
func TestDefaultSSHKeyPath_NoHome(t *testing.T) {
	// This test can't easily mock os.UserHomeDir failure, but we can verify
	// that the function returns a valid path when home is available
	path, err := DefaultSSHKeyPath()
	if err == nil && path == "" {
		t.Error("DefaultSSHKeyPath should return either a valid path or an error")
	}
}

// TestGenerateSSHKey_MultipleKeys tests generating multiple independent keys.
func TestGenerateSSHKey_MultipleKeys(t *testing.T) {
	dir := t.TempDir()

	// Generate two keys
	keyPath1 := filepath.Join(dir, "key1")
	keyPath2 := filepath.Join(dir, "key2")

	err1 := GenerateSSHKey(keyPath1)
	err2 := GenerateSSHKey(keyPath2)

	if err1 != nil || err2 != nil {
		t.Fatalf("GenerateSSHKey failed: %v, %v", err1, err2)
	}

	// Read both keys
	key1Data, _ := os.ReadFile(keyPath1)
	key2Data, _ := os.ReadFile(keyPath2)

	// Keys should be different
	if string(key1Data) == string(key2Data) {
		t.Error("two independently generated keys should not be identical")
	}
}

// TestGenerateSSHKey_PublicKeyFormat tests that public key is in correct authorized_keys format.
func TestGenerateSSHKey_PublicKeyFormat(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "fmt_test")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	pubData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	pubString := string(pubData)
	// authorized_keys format should start with key type
	if !strings.HasPrefix(pubString, "ssh-ed25519") {
		t.Errorf("public key should start with 'ssh-ed25519', got: %s", pubString[:20])
	}
	// Should end with newline
	if !strings.HasSuffix(pubString, "\n") {
		t.Error("public key should end with newline")
	}
}

// TestGenerateSSHKey_PrivateKeyNotReadable tests private key is not world-readable.
func TestGenerateSSHKey_PrivateKeyNotReadable(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "secure_key")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	mode := info.Mode()
	// Check that group and other don't have read access
	if mode&0o044 != 0 {
		t.Errorf("private key should not be readable by group/other, perms=%o", mode.Perm())
	}
}

// TestGenerateSSHKey_PublicKeyReadable tests public key is readable by all.
func TestGenerateSSHKey_PublicKeyReadable(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "readable_key")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	pubPath := keyPath + ".pub"
	info, err := os.Stat(pubPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	mode := info.Mode()
	// Check that others can read (0o644)
	if mode&0o004 == 0 {
		t.Errorf("public key should be readable by others, perms=%o", mode.Perm())
	}
}

// TestDefaultSSHKeyPath_Consistency tests that DefaultSSHKeyPath returns consistent path.
func TestDefaultSSHKeyPath_Consistency(t *testing.T) {
	path1, err1 := DefaultSSHKeyPath()
	path2, err2 := DefaultSSHKeyPath()

	if err1 != nil || err2 != nil {
		t.Fatalf("DefaultSSHKeyPath failed: %v, %v", err1, err2)
	}

	if path1 != path2 {
		t.Errorf("DefaultSSHKeyPath should be consistent: %q != %q", path1, path2)
	}
}

// TestGenerateSSHKey_DirectoryPermissions tests directory is created with 0700.
func TestGenerateSSHKey_DirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "deep", "nested", "test_key")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	// Check parent directory perms
	parentDir := filepath.Dir(keyPath)
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("Stat parent: %v", err)
	}

	if info.Mode().Perm() != 0o700 {
		t.Errorf("directory permissions = %o, want 0o700", info.Mode().Perm())
	}
}

// TestGenerateSSHKey_KeyFileSize tests that generated keys have reasonable size.
func TestGenerateSSHKey_KeyFileSize(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "size_test")

	err := GenerateSSHKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	privInfo, _ := os.Stat(keyPath)
	pubInfo, _ := os.Stat(keyPath + ".pub")

	privSize := privInfo.Size()
	pubSize := pubInfo.Size()

	// Ed25519 private key in OpenSSH format is roughly 400+ bytes
	// Public key is roughly 100+ bytes
	if privSize < 300 {
		t.Errorf("private key size = %d bytes, seems too small", privSize)
	}
	if pubSize < 50 {
		t.Errorf("public key size = %d bytes, seems too small", pubSize)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if len(s)-i >= len(substr) {
			match := true
			for j := 0; j < len(substr); j++ {
				if s[i+j] != substr[j] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// TestFindDefaultSSHKey_FileExists tests the branch where the default key file exists.
// It temporarily redirects HOME so findDefaultSSHKey locates the created key.
func TestFindDefaultSSHKey_FileExists(t *testing.T) {
	fakeHome := t.TempDir()
	sshDir := filepath.Join(fakeHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("setup ssh dir: %v", err)
	}
	// Generate a real key so the file exists and is non-empty.
	keyPath := filepath.Join(sshDir, "khunquant_ed25519.key")
	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}

	t.Setenv("HOME", fakeHome)

	got := findDefaultSSHKey()
	if got == "" {
		t.Error("findDefaultSSHKey should return a non-empty path when key file exists")
	}
	if got != keyPath {
		t.Errorf("findDefaultSSHKey = %q, want %q", got, keyPath)
	}
}

// TestGenerateSSHKey_DirectoryBlockedByFile tests MkdirAll failure path.
func TestGenerateSSHKey_DirectoryBlockedByFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file-as-directory collision behaves differently on Windows")
	}
	base := t.TempDir()
	// Create a file at a path we'll tell GenerateSSHKey to use as a directory.
	blocker := filepath.Join(base, "notadir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := GenerateSSHKey(filepath.Join(blocker, "key.pem"))
	if err == nil {
		t.Fatal("expected error when parent path is a file, not a directory")
	}
}
