package tools

import (
	"os"
	"regexp"
	"testing"
)

func TestContainsUnescapedRegexMeta_NoMeta(t *testing.T) {
	cases := []string{"/home/user/projects", "simple_name", ""}
	for _, s := range cases {
		if containsUnescapedRegexMeta(s) {
			t.Errorf("containsUnescapedRegexMeta(%q) = true, want false", s)
		}
	}
}

func TestContainsUnescapedRegexMeta_WithMeta(t *testing.T) {
	cases := []string{"file.txt", "foo+bar", "a*b", "a?b", "(group)", "[set]", "{n}", "a|b"}
	for _, s := range cases {
		if !containsUnescapedRegexMeta(s) {
			t.Errorf("containsUnescapedRegexMeta(%q) = false, want true", s)
		}
	}
}

func TestContainsUnescapedRegexMeta_Escaped(t *testing.T) {
	// Escaped dot \. should not count as meta
	if containsUnescapedRegexMeta(`\/home\/user`) {
		t.Error(`containsUnescapedRegexMeta("\/home\/user") should be false (escaped slashes, no meta)`)
	}
}

func TestContainsUnescapedRegexMeta_TrailingBackslash(t *testing.T) {
	// Trailing backslash alone is considered "escaped" state = true
	if !containsUnescapedRegexMeta(`abc\`) {
		t.Error(`trailing backslash should return true`)
	}
}

func TestUnescapeRegexLiteral_Simple(t *testing.T) {
	got, ok := unescapeRegexLiteral("/home/user")
	if !ok || got != "/home/user" {
		t.Errorf("unescapeRegexLiteral('/home/user') = %q, %v", got, ok)
	}
}

func TestUnescapeRegexLiteral_Escaped(t *testing.T) {
	got, ok := unescapeRegexLiteral(`\/home\/user`)
	if !ok || got != "/home/user" {
		t.Errorf(`unescapeRegexLiteral("\/home\/user") = %q, %v`, got, ok)
	}
}

func TestUnescapeRegexLiteral_TrailingBackslash(t *testing.T) {
	_, ok := unescapeRegexLiteral(`abc\`)
	if ok {
		t.Error("unescapeRegexLiteral with trailing backslash should return ok=false")
	}
}

func TestUnescapeRegexLiteral_Empty(t *testing.T) {
	got, ok := unescapeRegexLiteral("")
	if !ok || got != "" {
		t.Errorf("unescapeRegexLiteral('') = %q, %v", got, ok)
	}
}

func TestAppendUniquePath_NewPath(t *testing.T) {
	paths := []string{"/a", "/b"}
	result := appendUniquePath(paths, "/c")
	if len(result) != 3 {
		t.Errorf("appendUniquePath new: got %d, want 3", len(result))
	}
}

func TestAppendUniquePath_Duplicate(t *testing.T) {
	paths := []string{"/a", "/b"}
	result := appendUniquePath(paths, "/a")
	if len(result) != 2 {
		t.Errorf("appendUniquePath duplicate: got %d, want 2 (no duplicate added)", len(result))
	}
}

func TestAppendUniquePath_Empty(t *testing.T) {
	result := appendUniquePath(nil, "/a")
	if len(result) != 1 || result[0] != "/a" {
		t.Errorf("appendUniquePath nil: %v", result)
	}
}

func TestExtractAllowedPathRoot_LiteralAnchor(t *testing.T) {
	re := regexp.MustCompile(`^/home/user(?:/|$)`)
	root, abs := extractAllowedPathRoot(re)
	if root != "/home/user" {
		t.Errorf("extractAllowedPathRoot = %q, want /home/user", root)
	}
	if !abs {
		t.Error("expected abs=true for absolute path")
	}
}

func TestExtractAllowedPathRoot_NoAnchor(t *testing.T) {
	re := regexp.MustCompile(`/home/user`)
	root, _ := extractAllowedPathRoot(re)
	if root != "" {
		t.Errorf("extractAllowedPathRoot without ^ anchor = %q, want empty", root)
	}
}

func TestExtractAllowedPathRoot_WithMeta(t *testing.T) {
	re := regexp.MustCompile(`^/home/user.*`)
	root, _ := extractAllowedPathRoot(re)
	if root != "" {
		t.Errorf("extractAllowedPathRoot with meta chars = %q, want empty", root)
	}
}

func TestResolveExistingAncestor_ExistingDir(t *testing.T) {
	tmp := t.TempDir()
	got, err := resolveExistingAncestor(tmp)
	if err != nil {
		t.Fatalf("resolveExistingAncestor existing dir: %v", err)
	}
	if got == "" {
		t.Error("resolveExistingAncestor should return non-empty for existing dir")
	}
}

func TestResolveExistingAncestor_MissingChildOfExisting(t *testing.T) {
	tmp := t.TempDir()
	missing := tmp + "/does/not/exist"
	got, err := resolveExistingAncestor(missing)
	if err != nil {
		t.Fatalf("resolveExistingAncestor missing child: %v", err)
	}
	// resolved ancestor should be equal to tmp (the deepest existing part)
	if got == "" {
		t.Error("resolveExistingAncestor should return non-empty (ancestor exists)")
	}
}

func TestIsWithinWorkspace_Inside(t *testing.T) {
	if !isWithinWorkspace("/work/project/src/main.go", "/work/project") {
		t.Error("expected inside workspace")
	}
}

func TestIsWithinWorkspace_Outside(t *testing.T) {
	if isWithinWorkspace("/other/path/file.go", "/work/project") {
		t.Error("expected outside workspace")
	}
}

func TestIsWithinWorkspace_Root(t *testing.T) {
	if !isWithinWorkspace("/work/project", "/work/project") {
		t.Error("same path should be within workspace")
	}
}

func TestReadFileTool_Metadata(t *testing.T) {
	tool := NewReadFileTool(t.TempDir(), false, 1024*1024)
	if tool.Name() != NameReadFile {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameReadFile)
	}
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

func TestWriteFileTool_Metadata(t *testing.T) {
	tool := NewWriteFileTool(t.TempDir(), false)
	if tool.Name() != NameWriteFile {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameWriteFile)
	}
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

func TestListDirTool_Metadata(t *testing.T) {
	tool := NewListDirTool(t.TempDir(), false)
	if tool.Name() != NameListDir {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameListDir)
	}
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

func TestSandboxFs_ReadDir_Basic(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(tmp+"/hello.txt", []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Mkdir(tmp+"/subdir", 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	fs := &sandboxFs{workspace: tmp}
	entries, err := fs.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestSandboxFs_ReadDir_Nonexistent(t *testing.T) {
	tmp := t.TempDir()
	fs := &sandboxFs{workspace: tmp}
	_, err := fs.ReadDir(tmp + "/no-such-dir")
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
}

func TestSandboxFs_ReadDir_EmptyWorkspace(t *testing.T) {
	fs := &sandboxFs{workspace: ""}
	_, err := fs.ReadDir("/anything")
	if err == nil {
		t.Error("expected error for empty workspace")
	}
}

func TestWhitelistFs_ReadFile_Matching(t *testing.T) {
	tmp := t.TempDir()
	filePath := tmp + "/secret.txt"
	if err := os.WriteFile(filePath, []byte("secret"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Pattern matches the temp dir
	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(tmp))
	wfs := &whitelistFs{
		sandbox:  &sandboxFs{workspace: tmp},
		host:     hostFs{},
		patterns: []*regexp.Regexp{pattern},
	}

	data, err := wfs.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile via whitelist: %v", err)
	}
	if string(data) != "secret" {
		t.Errorf("ReadFile content = %q, want secret", string(data))
	}
}

func TestWhitelistFs_ReadFile_NotMatching(t *testing.T) {
	tmp := t.TempDir()
	filePath := tmp + "/data.txt"
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Pattern does NOT match tmp
	pattern := regexp.MustCompile(`^/totally/different/path`)
	wfs := &whitelistFs{
		sandbox:  &sandboxFs{workspace: tmp},
		host:     hostFs{},
		patterns: []*regexp.Regexp{pattern},
	}

	data, err := wfs.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile via sandbox: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("ReadFile content = %q, want data", string(data))
	}
}

func TestWhitelistFs_ReadDir_Matching(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(tmp+"/file.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(tmp))
	wfs := &whitelistFs{
		sandbox:  &sandboxFs{workspace: tmp},
		host:     hostFs{},
		patterns: []*regexp.Regexp{pattern},
	}

	entries, err := wfs.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry")
	}
}

func TestWhitelistFs_ReadDir_NotMatching(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(tmp+"/file.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pattern := regexp.MustCompile(`^/totally/different/path`)
	wfs := &whitelistFs{
		sandbox:  &sandboxFs{workspace: tmp},
		host:     hostFs{},
		patterns: []*regexp.Regexp{pattern},
	}

	entries, err := wfs.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir via sandbox: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry from sandbox")
	}
}

func TestValidatePathWithAllowPaths_EmptyWorkspace(t *testing.T) {
	_, err := validatePathWithAllowPaths("/some/path", "", false, nil)
	if err == nil {
		t.Error("expected error for empty workspace")
	}
}

func TestValidatePathWithAllowPaths_AbsolutePath_NoRestrict(t *testing.T) {
	tmp := t.TempDir()
	got, err := validatePathWithAllowPaths(tmp+"/file.txt", tmp, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmp+"/file.txt" {
		t.Errorf("got %q, want %q", got, tmp+"/file.txt")
	}
}

func TestValidatePathWithAllowPaths_RelativePath_NoRestrict(t *testing.T) {
	tmp := t.TempDir()
	got, err := validatePathWithAllowPaths("subdir/file.txt", tmp, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmp+"/subdir/file.txt" {
		t.Errorf("got %q, want %q", got, tmp+"/subdir/file.txt")
	}
}

func TestValidatePathWithAllowPaths_Restrict_InsideWorkspace(t *testing.T) {
	tmp := t.TempDir()
	got, err := validatePathWithAllowPaths(tmp+"/file.txt", tmp, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmp+"/file.txt" {
		t.Errorf("got %q, want %q", got, tmp+"/file.txt")
	}
}

func TestValidatePathWithAllowPaths_Restrict_OutsideWorkspace(t *testing.T) {
	tmp := t.TempDir()
	_, err := validatePathWithAllowPaths("/etc/passwd", tmp, true, nil)
	if err == nil {
		t.Error("expected access denied error for path outside workspace")
	}
}

func TestValidatePathWithAllowPaths_Restrict_AllowedByPattern(t *testing.T) {
	tmp := t.TempDir()
	// Path outside workspace but matched by an allow pattern
	pattern := regexp.MustCompile(`^/etc/`)
	got, err := validatePathWithAllowPaths("/etc/passwd", tmp, true, []*regexp.Regexp{pattern})
	if err != nil {
		t.Fatalf("unexpected error with allowed pattern: %v", err)
	}
	if got != "/etc/passwd" {
		t.Errorf("got %q, want /etc/passwd", got)
	}
}

func TestValidatePathWithAllowPaths_Restrict_SymlinkOutside(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()

	// Create a file in the outside dir and symlink it inside workspace
	outsideFile := outside + "/secret.txt"
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	symlinkPath := workspace + "/evil"
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := validatePathWithAllowPaths(symlinkPath, workspace, true, nil)
	if err == nil {
		t.Error("expected error: symlink resolves outside workspace")
	}
}

func TestGetInt64Arg_Missing(t *testing.T) {
	got, err := getInt64Arg(map[string]any{}, "key", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Errorf("got %d, want 42 (default)", got)
	}
}

func TestGetInt64Arg_Float64Valid(t *testing.T) {
	got, err := getInt64Arg(map[string]any{"key": float64(10)}, "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Errorf("got %d, want 10", got)
	}
}

func TestGetInt64Arg_Float64NonInteger(t *testing.T) {
	_, err := getInt64Arg(map[string]any{"key": float64(3.14)}, "key", 0)
	if err == nil {
		t.Error("expected error for non-integer float")
	}
}

func TestGetInt64Arg_IntType(t *testing.T) {
	got, err := getInt64Arg(map[string]any{"key": int(7)}, "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 7 {
		t.Errorf("got %d, want 7", got)
	}
}

func TestGetInt64Arg_Int64Type(t *testing.T) {
	got, err := getInt64Arg(map[string]any{"key": int64(99)}, "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 99 {
		t.Errorf("got %d, want 99", got)
	}
}

func TestGetInt64Arg_StringValid(t *testing.T) {
	got, err := getInt64Arg(map[string]any{"key": "123"}, "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 123 {
		t.Errorf("got %d, want 123", got)
	}
}

func TestGetInt64Arg_StringInvalid(t *testing.T) {
	_, err := getInt64Arg(map[string]any{"key": "not-a-number"}, "key", 0)
	if err == nil {
		t.Error("expected error for invalid string integer")
	}
}

func TestGetInt64Arg_UnsupportedType(t *testing.T) {
	_, err := getInt64Arg(map[string]any{"key": true}, "key", 0)
	if err == nil {
		t.Error("expected error for unsupported type bool")
	}
}
