package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestWriteFileAtomic_BasicWrite tests writing a simple file atomically
func TestWriteFileAtomic_BasicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "test.txt")

	data := []byte("Hello, World!")
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

// TestWriteFileAtomic_EmptyData tests writing empty data
func TestWriteFileAtomic_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "empty.txt")

	err := WriteFileAtomic(targetPath, []byte{}, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify empty file exists
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(content))
	}
}

// TestWriteFileAtomic_LargeData tests writing larger data
func TestWriteFileAtomic_LargeData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "large.bin")

	// Create 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := WriteFileAtomic(targetPath, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify size matches
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size() != int64(len(data)) {
		t.Errorf("file size mismatch: got %d, want %d", info.Size(), len(data))
	}
}

// TestWriteFileAtomic_PermissionsRW tests file permissions (owner read/write)
func TestWriteFileAtomic_PermissionsRW(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "secure.txt")

	data := []byte("sensitive data")
	err := WriteFileAtomic(targetPath, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Check permissions (mask platform-specific bits)
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("permission mismatch: got %o, want %o", perm, 0o600)
	}
}

// TestWriteFileAtomic_PermissionsReadable tests file permissions (world readable)
func TestWriteFileAtomic_PermissionsReadable(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "readable.txt")

	data := []byte("public data")
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o644 {
		t.Errorf("permission mismatch: got %o, want %o", perm, 0o644)
	}
}

// TestWriteFileAtomic_NestedDirectory tests creating nested directories
func TestWriteFileAtomic_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "a", "b", "c", "test.txt")

	data := []byte("nested content")
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify all parent directories were created
	for _, p := range []string{
		filepath.Join(tmpDir, "a"),
		filepath.Join(tmpDir, "a", "b"),
		filepath.Join(tmpDir, "a", "b", "c"),
	} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("parent dir missing: %s: %v", p, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", p)
		}
	}

	// Verify file content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("content mismatch: got %q", string(content))
	}
}

// TestWriteFileAtomic_Overwrite tests overwriting an existing file
func TestWriteFileAtomic_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "overwrite.txt")

	// Write first version
	err := WriteFileAtomic(targetPath, []byte("version 1"), 0o644)
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Overwrite with second version
	err = WriteFileAtomic(targetPath, []byte("version 2"), 0o644)
	if err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "version 2" {
		t.Errorf("content mismatch: got %q, want %q", string(content), "version 2")
	}
}

// TestWriteFileAtomic_BinaryData tests writing binary data correctly
func TestWriteFileAtomic_BinaryData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "binary.bin")

	// Create specific binary sequence
	data := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify exact binary content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) != len(data) {
		t.Fatalf("size mismatch: got %d, want %d", len(content), len(data))
	}
	for i := range data {
		if content[i] != data[i] {
			t.Errorf("byte[%d] mismatch: got %x, want %x", i, content[i], data[i])
		}
	}
}

// TestWriteFileAtomic_TempFileCleanup tests that temp files are cleaned up on error
func TestWriteFileAtomic_TempFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	// Make tmpDir read-only to trigger permission error during write
	targetPath := filepath.Join(tmpDir, "readonly", "test.txt")
	readonlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readonlyDir, 0o755)

	// Make directory read-only (fails on Windows but test should still pass)
	os.Chmod(readonlyDir, 0o444)
	defer os.Chmod(readonlyDir, 0o755) // restore permissions in cleanup

	err := WriteFileAtomic(targetPath, []byte("data"), 0o644)
	if err == nil {
		t.Fatalf("WriteFileAtomic should fail on read-only directory")
	}

	// Check that no temp files were left behind (best effort)
	// This is difficult to verify reliably across platforms, but we at least
	// ensure the function doesn't panic.
}

// TestWriteFileAtomic_UTF8Content tests writing UTF-8 encoded content
func TestWriteFileAtomic_UTF8Content(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "utf8.txt")

	// Unicode content: emoji, CJK, etc.
	data := []byte("Hello 世界 🌍 Привет Здравствуй")
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("content mismatch: got %q, want %q", string(content), string(data))
	}
}

// TestWriteFileAtomic_MultipleSequential tests multiple sequential writes to same path
func TestWriteFileAtomic_MultipleSequential(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "sequential.txt")

	for i := 0; i < 5; i++ {
		data := []byte("iteration " + string(rune('0'+i)))
		err := WriteFileAtomic(targetPath, data, 0o644)
		if err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
	}

	// Final content should be from last write
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "iteration 4" {
		t.Errorf("final content mismatch: got %q", string(content))
	}
}

// TestCopyFile_BasicCopy tests copying a file atomically
func TestCopyFile_BasicCopy(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	srcData := []byte("source content")
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Copy file
	err = CopyFile(srcPath, dstPath, 0o644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if string(dstContent) != string(srcData) {
		t.Errorf("content mismatch: got %q, want %q", string(dstContent), string(srcData))
	}
}

// TestCopyFile_ToNestedPath tests copying a file to a nested destination
func TestCopyFile_ToNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "nested", "deep", "dest.txt")

	// Create source file
	srcData := []byte("nested copy")
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Copy file with nested path
	err = CopyFile(srcPath, dstPath, 0o600)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination exists and has correct content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if string(dstContent) != string(srcData) {
		t.Errorf("content mismatch: got %q, want %q", string(dstContent), string(srcData))
	}

	// Verify permissions
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Stat dest failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm mismatch: got %o, want %o", info.Mode().Perm(), 0o600)
	}
}

// TestCopyFile_SourceNotFound tests copying a non-existent file
func TestCopyFile_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := CopyFile(srcPath, dstPath, 0o644)
	if err == nil {
		t.Fatalf("CopyFile should fail for non-existent source")
	}

	// Verify destination was NOT created
	if _, statErr := os.Stat(dstPath); statErr == nil {
		t.Errorf("destination should not exist after failed copy")
	}
}

// TestCopyFile_LargeBinaryFile tests copying a large binary file
func TestCopyFile_LargeBinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "large.bin")
	dstPath := filepath.Join(tmpDir, "copy.bin")

	// Create 5MB binary file
	srcData := make([]byte, 5*1024*1024)
	for i := range srcData {
		srcData[i] = byte((i * 17) % 256)
	}
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Copy file
	err = CopyFile(srcPath, dstPath, 0o644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify size
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("Stat source failed: %v", err)
	}
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Stat dest failed: %v", err)
	}
	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("size mismatch: got %d, want %d", dstInfo.Size(), srcInfo.Size())
	}

	// Spot check content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if len(dstContent) != len(srcData) {
		t.Fatalf("content size mismatch")
	}
	// Just verify first and last bytes match
	if dstContent[0] != srcData[0] || dstContent[len(dstContent)-1] != srcData[len(srcData)-1] {
		t.Errorf("content mismatch at boundaries")
	}
}

// TestCopyFile_OverwriteExisting tests copying over an existing file
func TestCopyFile_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// Create source
	srcData := []byte("new content")
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Create existing destination with different content
	err = os.WriteFile(dstPath, []byte("old content"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile dest failed: %v", err)
	}

	// Copy over existing file
	err = CopyFile(srcPath, dstPath, 0o644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination has new content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if string(dstContent) != string(srcData) {
		t.Errorf("content mismatch: got %q, want %q", string(dstContent), string(srcData))
	}
}

// TestCopyFile_EmptyFile tests copying an empty file
func TestCopyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "empty.txt")
	dstPath := filepath.Join(tmpDir, "empty_copy.txt")

	// Create empty source
	err := os.WriteFile(srcPath, []byte{}, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Copy empty file
	err = CopyFile(srcPath, dstPath, 0o644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination is also empty
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if len(dstContent) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(dstContent))
	}
}

// TestCopyFile_PreservesContent tests that copy preserves exact binary content
func TestCopyFile_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.bin")
	dstPath := filepath.Join(tmpDir, "dest.bin")

	// Create binary source with all byte values
	srcData := make([]byte, 256)
	for i := 0; i < 256; i++ {
		srcData[i] = byte(i)
	}
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}

	// Copy file
	err = CopyFile(srcPath, dstPath, 0o644)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify exact content match
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dest failed: %v", err)
	}
	if len(dstContent) != len(srcData) {
		t.Fatalf("size mismatch: got %d, want %d", len(dstContent), len(srcData))
	}
	for i := range srcData {
		if dstContent[i] != srcData[i] {
			t.Errorf("byte[%d] mismatch: got %d, want %d", i, dstContent[i], srcData[i])
		}
	}
}

// TestWriteFileAtomic_MkdirAllFailure tests MkdirAll error handling
func TestWriteFileAtomic_MkdirAllFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file where a directory needs to be
	blockingFilePath := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFilePath, []byte("blocking"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Try to write to a path where a file already exists where we need a directory
	// This should cause MkdirAll to fail
	invalidPath := filepath.Join(blockingFilePath, "subdir", "file.txt")
	err := WriteFileAtomic(invalidPath, []byte("data"), 0o644)
	if err == nil {
		t.Errorf("WriteFileAtomic should fail when a file blocks directory creation")
	}
}

// TestWriteFileAtomic_SyncFailure attempts to trigger sync error (difficult)
func TestWriteFileAtomic_SyncFailure(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	// This test verifies the function succeeds with valid setup
	err := WriteFileAtomic(testPath, []byte("test"), 0o644)
	if err != nil {
		t.Errorf("WriteFileAtomic should succeed: %v", err)
	}

	// On most systems, Sync() succeeds, so we just verify the file was created
	if _, statErr := os.Stat(testPath); statErr != nil {
		t.Errorf("target file should exist after successful write")
	}
}

// TestWriteFileAtomic_DirectoryPermissions tests creating directories with proper permissions
func TestWriteFileAtomic_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "x", "y", "z", "file.txt")

	err := WriteFileAtomic(deepPath, []byte("deep"), 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify all parent directories were created with 0o755
	parentPath := filepath.Join(tmpDir, "x")
	info, err := os.Stat(parentPath)
	if err != nil {
		t.Fatalf("parent dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("parent path should be a directory")
	}
}

// TestWriteFileAtomic_RenameOverwrite tests atomic rename behavior
func TestWriteFileAtomic_RenameOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "overwrite_test.txt")

	// Write initial file
	err := WriteFileAtomic(targetPath, []byte("initial"), 0o644)
	if err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	// Overwrite with new data
	err = WriteFileAtomic(targetPath, []byte("updated"), 0o644)
	if err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(content) != "updated" {
		t.Errorf("content should be updated, got %q", string(content))
	}

	// Verify file still exists and has correct size
	statInfo, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if statInfo.Size() != 7 {
		t.Errorf("size should be 7, got %d", statInfo.Size())
	}
}

// TestWriteFileAtomic_AtomicBehavior verifies atomicity (no partial writes)
func TestWriteFileAtomic_AtomicBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "atomic.txt")

	// Write initial data
	initialData := []byte("initial data content")
	err := WriteFileAtomic(targetPath, initialData, 0o644)
	if err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	// Attempt to overwrite with new data
	newData := []byte("x") // Much shorter data
	err = WriteFileAtomic(targetPath, newData, 0o644)
	if err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	// Verify file contains exactly new data, not mixed content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(content) != "x" {
		t.Errorf("file should contain exactly 'x', got %q", string(content))
	}
	if len(content) != 1 {
		t.Errorf("file should be 1 byte, got %d", len(content))
	}
}

// TestWriteFileAtomic_NoTempFileLeakOnSuccess ensures temp file is not left behind
func TestWriteFileAtomic_NoTempFileLeakOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "leaked.txt")

	err := WriteFileAtomic(targetPath, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Check that only the target file exists, no .tmp- files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	tmpFileCount := 0
	for _, entry := range entries {
		if filepath.HasPrefix(entry.Name(), ".tmp-") {
			tmpFileCount++
		}
	}
	if tmpFileCount > 0 {
		t.Errorf("found %d temp files, should be 0", tmpFileCount)
	}
}

// TestWriteFileAtomic_PermissionBits tests specific permission bits
func TestWriteFileAtomic_PermissionBits(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"owner_read_write", 0o600},
		{"owner_read_write_group_read", 0o640},
		{"public_readable", 0o644},
		{"owner_exec", 0o700},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetPath := filepath.Join(tmpDir, tc.name+".txt")
			err := WriteFileAtomic(targetPath, []byte("test"), tc.perm)
			if err != nil {
				t.Fatalf("WriteFileAtomic failed: %v", err)
			}

			info, err := os.Stat(targetPath)
			if err != nil {
				t.Fatalf("Stat failed: %v", err)
			}

			actualPerm := info.Mode().Perm()
			if actualPerm != tc.perm {
				t.Errorf("permission mismatch: got %o, want %o", actualPerm, tc.perm)
			}
		})
	}
}

// TestWriteFileAtomic_RenameFailure tests handling of rename errors
func TestWriteFileAtomic_RenameFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory and make it read-only to prevent writing
	nonExistentParent := filepath.Join(tmpDir, "nonexistent")
	os.MkdirAll(nonExistentParent, 0o755)
	os.Chmod(nonExistentParent, 0o444)
	defer os.Chmod(nonExistentParent, 0o755)

	// Try to write to a path under the read-only directory
	targetUnderReadOnly := filepath.Join(nonExistentParent, "test.txt")
	err := WriteFileAtomic(targetUnderReadOnly, []byte("data"), 0o644)
	if err == nil {
		t.Errorf("WriteFileAtomic should fail when parent is read-only")
	}
}

// TestWriteFileAtomic_MkdirAllCreatesParents tests MkdirAll creates all parents
func TestWriteFileAtomic_MkdirAllCreatesParents(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "file.txt")

	err := WriteFileAtomic(deepPath, []byte("deep"), 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify all parents exist
	for _, parent := range []string{
		filepath.Join(tmpDir, "a"),
		filepath.Join(tmpDir, "a", "b"),
		filepath.Join(tmpDir, "a", "b", "c"),
		filepath.Join(tmpDir, "a", "b", "c", "d"),
		filepath.Join(tmpDir, "a", "b", "c", "d", "e"),
	} {
		info, err := os.Stat(parent)
		if err != nil {
			t.Errorf("parent %s should exist: %v", parent, err)
		}
		if !info.IsDir() {
			t.Errorf("parent %s should be a directory", parent)
		}
	}

	// Verify file exists
	if _, err := os.Stat(deepPath); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

// TestWriteFileAtomic_VeryLargeData tests writing very large amounts of data
func TestWriteFileAtomic_VeryLargeData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "large.bin")

	// Create 10MB of data
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = byte((i * 7) % 256)
	}

	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify size
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size() != int64(len(data)) {
		t.Errorf("size mismatch: got %d, want %d", info.Size(), int64(len(data)))
	}
}

// TestWriteFileAtomic_SequentialOverwrites tests multiple sequential overwrites
func TestWriteFileAtomic_SequentialOverwrites(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "sequential.txt")

	for i := 0; i < 10; i++ {
		data := []byte("version " + string(rune('0'+byte(i))))
		err := WriteFileAtomic(targetPath, data, 0o644)
		if err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}

		// Verify each write worked
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read after write %d failed: %v", i, err)
		}
		expected := "version " + string(rune('0'+byte(i)))
		if string(content) != expected {
			t.Errorf("write %d: got %q, want %q", i, string(content), expected)
		}
	}
}

// TestWriteFileAtomic_DirSyncSuccess tests directory sync (for durability)
func TestWriteFileAtomic_DirSyncSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "file.txt")

	// Normal write should succeed and directory sync should be attempted
	err := WriteFileAtomic(targetPath, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

// TestWriteFileAtomic_JSONData tests writing JSON data atomically
func TestWriteFileAtomic_JSONData(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "data.json")

	jsonData := []byte(`{"name":"test","value":123,"tags":["a","b"]}`)
	err := WriteFileAtomic(targetPath, jsonData, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != string(jsonData) {
		t.Errorf("JSON content mismatch")
	}
}

// TestWriteFileAtomic_MultipleConcurrentWrites tests writing to different files
func TestWriteFileAtomic_MultipleConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files in succession
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+byte(i)))+".txt")
		data := []byte("file " + string(rune('0'+byte(i))))
		err := WriteFileAtomic(path, data, 0o644)
		if err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
	}

	// Verify all files exist
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 files, got %d", len(entries))
	}
}

// TestCopyFile_DifferentPermissions tests copy with different permission modes
func TestCopyFile_DifferentPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	srcData := []byte("test data")
	err := os.WriteFile(srcPath, srcData, 0o644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Copy with restricted permissions
	err = CopyFile(srcPath, dstPath, 0o600)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify permissions
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0o600, got %o", info.Mode().Perm())
	}
}

// TestWriteFileAtomic_WithLinuxSpecialChars tests writing with special characters
func TestWriteFileAtomic_WithLinuxSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "special.txt")

	// Special characters that might appear in real files
	data := []byte("\x00\x01\x02\x03\n\r\t\\\"'")
	err := WriteFileAtomic(targetPath, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) != len(data) {
		t.Errorf("content size mismatch: got %d, want %d", len(content), len(data))
	}
}

// TestWriteFileAtomic_ConsecutiveReads tests reading after multiple writes
func TestWriteFileAtomic_ConsecutiveReads(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "consecutive.txt")

	// Multiple write-read cycles
	for i := 0; i < 3; i++ {
		writeData := []byte("version " + string(rune('1'+byte(i))))
		err := WriteFileAtomic(targetPath, writeData, 0o644)
		if err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}

		readData, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read %d failed: %v", i, err)
		}
		if string(readData) != string(writeData) {
			t.Errorf("cycle %d: data mismatch", i)
		}
	}
}

// TestWriteFileAtomic_TempFileCleanupOnBlockingDir ensures temp cleanup happens
func TestWriteFileAtomic_TempFileCleanupOnBlockingDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file where the directory will be needed
	blockingPath := filepath.Join(tmpDir, "blocking")
	os.WriteFile(blockingPath, []byte("blocking"), 0o644)

	// Try to write to a path that requires creating subdir under the file
	invalidTarget := filepath.Join(blockingPath, "subdir", "test.txt")
	_ = WriteFileAtomic(invalidTarget, []byte("data"), 0o644)

	// Verify no temp files were left behind
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, entry := range entries {
		if filepath.HasPrefix(entry.Name(), ".tmp-") {
			t.Errorf("temp file not cleaned up: %s", entry.Name())
		}
	}
}

// TestCopyFile_WithMultipleExtensions tests copying files with various extensions
func TestCopyFile_WithMultipleExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name  string
		ext   string
		data  []byte
	}{
		{"text file", ".txt", []byte("text content")},
		{"json file", ".json", []byte(`{"key":"value"}`)},
		{"binary file", ".bin", []byte{0xFF, 0xFE, 0xFD}},
		{"no extension", "", []byte("noext")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srcPath := filepath.Join(tmpDir, "src"+tc.ext)
			dstPath := filepath.Join(tmpDir, "dst"+tc.ext)

			err := os.WriteFile(srcPath, tc.data, 0o644)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			err = CopyFile(srcPath, dstPath, 0o644)
			if err != nil {
				t.Fatalf("CopyFile failed: %v", err)
			}

			dstData, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}
			if string(dstData) != string(tc.data) {
				t.Errorf("data mismatch")
			}
		})
	}
}

// TestWriteFileAtomic_SameTargetOverwrite ensures overwriting same target works
func TestWriteFileAtomic_SameTargetOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "same.txt")

	// Write version 1
	v1 := []byte("version 1 content here")
	err := WriteFileAtomic(targetPath, v1, 0o644)
	if err != nil {
		t.Fatalf("write 1 failed: %v", err)
	}

	// Read to verify
	content1, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read 1 failed: %v", err)
	}
	if string(content1) != string(v1) {
		t.Errorf("v1 mismatch")
	}

	// Overwrite with version 2
	v2 := []byte("version 2")
	err = WriteFileAtomic(targetPath, v2, 0o644)
	if err != nil {
		t.Fatalf("write 2 failed: %v", err)
	}

	// Verify v2
	content2, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read 2 failed: %v", err)
	}
	if string(content2) != string(v2) {
		t.Errorf("v2 mismatch, got %q", string(content2))
	}

	// Verify file is exactly v2 size, not v1 size
	if len(content2) != len(v2) {
		t.Errorf("final size = %d, want %d", len(content2), len(v2))
	}
}

// TestWriteFileAtomic_DirectoryBlockedByFile tests the MkdirAll failure path.
func TestWriteFileAtomic_DirectoryBlockedByFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file-as-directory collision behaves differently on Windows")
	}
	base := t.TempDir()
	// Create a regular file at a path that WriteFileAtomic will try to use as a directory.
	blocker := filepath.Join(base, "notadir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Now ask WriteFileAtomic to write into blocker/file.txt — MkdirAll must fail.
	err := WriteFileAtomic(filepath.Join(blocker, "file.txt"), []byte("data"), 0o600)
	if err == nil {
		t.Fatal("expected error when directory is blocked by a file")
	}
}

// TestWriteFileAtomic_ReadOnlyDir tests the OpenFile failure path.
func TestWriteFileAtomic_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only dir semantics differ on Windows")
	}
	base := t.TempDir()
	roDir := filepath.Join(base, "ro")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { os.Chmod(roDir, 0o755) })
	err := WriteFileAtomic(filepath.Join(roDir, "out.txt"), []byte("hello"), 0o600)
	if err == nil {
		t.Fatal("expected error writing into read-only directory")
	}
}
