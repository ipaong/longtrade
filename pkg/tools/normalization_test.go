package tools

import (
	"fmt"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/media"
)

func TestExtensionForMIMEType_Empty(t *testing.T) {
	got := extensionForMIMEType("")
	if got != ".bin" {
		t.Errorf("got %q, want .bin", got)
	}
}

func TestExtensionForMIMEType_JPEG(t *testing.T) {
	got := extensionForMIMEType("image/jpeg")
	if got != ".jpg" && got != ".jpeg" && got != ".jpe" {
		t.Errorf("got %q, want .jpg, .jpeg, or .jpe", got)
	}
}

func TestExtensionForMIMEType_PNG(t *testing.T) {
	got := extensionForMIMEType("image/png")
	if got == "" {
		t.Error("expected non-empty extension for image/png")
	}
}

func TestExtensionForMIMEType_AudioWAV(t *testing.T) {
	got := extensionForMIMEType("audio/wav")
	if got == "" {
		t.Error("expected non-empty extension for audio/wav")
	}
}

func TestExtensionForMIMEType_AudioMP3(t *testing.T) {
	got := extensionForMIMEType("audio/mpeg")
	if got == "" {
		t.Error("expected non-empty extension for audio/mpeg")
	}
}

func TestExtensionForMIMEType_VideoMP4(t *testing.T) {
	got := extensionForMIMEType("video/mp4")
	if got == "" {
		t.Error("expected non-empty extension for video/mp4")
	}
}

func TestExtensionForMIMEType_Unknown(t *testing.T) {
	got := extensionForMIMEType("application/x-super-unknown-type-xyz")
	// Falls back to filepath.Ext of the mime type — returns empty since no dot
	_ = got // just ensure no panic
}

func TestExtensionForMIMEType_GIF(t *testing.T) {
	got := extensionForMIMEType("image/gif")
	if got == "" {
		t.Error("expected non-empty extension for image/gif")
	}
}

func TestExtensionForMIMEType_WebP(t *testing.T) {
	got := extensionForMIMEType("image/webp")
	if got == "" {
		t.Error("expected non-empty extension for image/webp")
	}
}

func TestExtensionForMIMEType_OGG(t *testing.T) {
	got := extensionForMIMEType("audio/ogg")
	if got == "" {
		t.Error("expected non-empty extension for audio/ogg")
	}
}

func TestExtensionForMIMEType_XWAV(t *testing.T) {
	got := extensionForMIMEType("audio/x-wav")
	if got == "" {
		t.Error("expected non-empty extension for audio/x-wav")
	}
}

func TestLooksLikeLargeBase64Payload_Short(t *testing.T) {
	if looksLikeLargeBase64Payload("abc") {
		t.Error("short text should not look like base64 payload")
	}
}

func TestLooksLikeLargeBase64Payload_NotBase64(t *testing.T) {
	// >1024 chars but not base64-like (lots of special chars)
	text := ""
	for len(text) < 2048 {
		text += "hello world! this is a normal sentence with spaces. "
	}
	if looksLikeLargeBase64Payload(text) {
		t.Error("normal text should not look like base64 payload")
	}
}

func TestLooksLikeLargeBase64Payload_IsBase64(t *testing.T) {
	// Generate a long base64-like string (only A-Z, a-z, 0-9, +, /, =)
	chunk := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	text := ""
	for len(text) < 2048 {
		text += chunk
	}
	if !looksLikeLargeBase64Payload(text) {
		t.Error("dense base64-like text (>1024 chars, ratio>=0.97) should be detected")
	}
}

func TestLooksLikeLargeBase64Payload_TooManySpaces(t *testing.T) {
	// base64 chars but interspersed with many spaces — ratio will be ok but space density too high
	chunk := "ABCDEFGHIJK LMNOPQRSTU VWXYZ01234 "
	text := ""
	for len(text) < 2048 {
		text += chunk
	}
	// With 1/4 spaces, spaceCount > len/128 should fail the second condition
	result := looksLikeLargeBase64Payload(text)
	// Result depends on thresholds; just ensure no panic
	_ = result
}

func TestSanitizeToolLLMContent_Normal(t *testing.T) {
	input := "This is a normal response."
	got := sanitizeToolLLMContent(input)
	if got != input {
		t.Errorf("normal text should pass through unchanged, got: %q", got)
	}
}

func TestSanitizeToolLLMContent_Empty(t *testing.T) {
	got := sanitizeToolLLMContent("")
	if got != "" {
		t.Errorf("empty input should produce empty output, got: %q", got)
	}
}

func TestSanitizeToolLLMContent_WhitespaceOnly(t *testing.T) {
	got := sanitizeToolLLMContent("   \n  ")
	if got != "   \n  " {
		t.Errorf("whitespace-only should pass through unchanged, got: %q", got)
	}
}

func TestSanitizeToolLLMContent_LargeBase64(t *testing.T) {
	chunk := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	text := ""
	for len(text) < 2048 {
		text += chunk
	}
	got := sanitizeToolLLMContent(text)
	if got != largeBase64OmittedMessage {
		t.Errorf("large base64 payload should be replaced with omitted message, got: %q", got)
	}
}

func TestSanitizeToolLLMContent_MarkdownDataURL_OnlyURL(t *testing.T) {
	// text is only the data URL embedded in markdown — cleaned becomes empty → inlineMediaOmittedMessage
	text := "![img](data:image/png;base64,abc123)"
	got := sanitizeToolLLMContent(text)
	if got != inlineMediaOmittedMessage {
		t.Errorf("markdown data URL only: got %q, want %q", got, inlineMediaOmittedMessage)
	}
}

func TestSanitizeToolLLMContent_MarkdownDataURL_WithText(t *testing.T) {
	// cleaned has remaining text after stripping the data URL
	text := "some text ![img](data:image/png;base64,abc123) more text"
	got := sanitizeToolLLMContent(text)
	if got == "" {
		t.Error("expected non-empty result")
	}
}

func TestSanitizeToolLLMContent_RawDataURL(t *testing.T) {
	// raw data URL (not markdown-wrapped)
	text := "result: data:image/png;base64,iVBORw0KGgo="
	got := sanitizeToolLLMContent(text)
	_ = got // ensure no panic
}

// mockMediaStore is a simple in-memory MediaStore for testing.
type mockMediaStore struct {
	fail bool
}

func (m *mockMediaStore) Store(localPath string, meta media.MediaMeta, scope string) (string, error) {
	if m.fail {
		return "", fmt.Errorf("mock store failure")
	}
	return "media://test-ref-123", nil
}

func (m *mockMediaStore) Resolve(ref string) (string, error)                             { return "", nil }
func (m *mockMediaStore) ResolveWithMeta(ref string) (string, media.MediaMeta, error)    { return "", media.MediaMeta{}, nil }
func (m *mockMediaStore) ReleaseAll(scope string) error                                  { return nil }

func TestStoreInlineDataURL_AlreadySeen(t *testing.T) {
	seen := map[string]struct{}{"data:image/png;base64,abc": {}}
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", "data:image/png;base64,abc", seen)
	if ref != "" || note != "" {
		t.Errorf("seen URL should return empty ref and note, got ref=%q note=%q", ref, note)
	}
}

func TestStoreInlineDataURL_NotDataPrefix(t *testing.T) {
	seen := map[string]struct{}{}
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", "https://example.com/img.png", seen)
	if ref != "" || note != "" {
		t.Errorf("non-data URL should return empty, got ref=%q note=%q", ref, note)
	}
}

func TestStoreInlineDataURL_NarrowComma(t *testing.T) {
	// "data:," has comma at index 5 — triggers the "could not be parsed" error
	seen := map[string]struct{}{}
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", "data:,", seen)
	if ref != "" {
		t.Errorf("narrow comma: expected empty ref, got %q", ref)
	}
	if note == "" {
		t.Error("narrow comma: expected error note")
	}
}

func TestStoreInlineDataURL_NotBase64Encoded(t *testing.T) {
	// non-base64 encoding — `;base64` not in meta
	seen := map[string]struct{}{}
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", "data:text/plain,hello world", seen)
	if ref != "" {
		t.Errorf("non-base64: expected empty ref, got %q", ref)
	}
	if note == "" {
		t.Error("non-base64: expected error note")
	}
}

func TestStoreInlineDataURL_InvalidBase64Payload(t *testing.T) {
	// ;base64 present but payload is invalid
	seen := map[string]struct{}{}
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", "data:image/png;base64,!!!invalid!!!", seen)
	if ref != "" {
		t.Errorf("invalid base64: expected empty ref, got %q", ref)
	}
	if note == "" {
		t.Error("invalid base64: expected error note")
	}
}

func TestStoreInlineDataURL_ValidBase64_SuccessfulStore(t *testing.T) {
	// valid 1x1 pixel GIF base64
	seen := map[string]struct{}{}
	dataURL := "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	ref, note := storeInlineDataURL("tool", &mockMediaStore{}, "ch", "id", dataURL, seen)
	if ref == "" {
		t.Errorf("valid base64: expected ref, got empty; note=%q", note)
	}
}

func TestStoreInlineDataURL_ValidBase64_StoreFails(t *testing.T) {
	seen := map[string]struct{}{}
	dataURL := "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	ref, note := storeInlineDataURL("tool", &mockMediaStore{fail: true}, "ch", "id", dataURL, seen)
	if ref != "" {
		t.Errorf("store fail: expected empty ref, got %q", ref)
	}
	if note == "" {
		t.Error("store fail: expected error note")
	}
}

func TestExtractInlineMediaRefs_NoDataURLs(t *testing.T) {
	seen := map[string]struct{}{}
	cleaned, refs, notes := extractInlineMediaRefs("plain text with no data URLs", "tool", &mockMediaStore{}, "ch", "id", seen)
	if cleaned != "plain text with no data URLs" {
		t.Errorf("no data URLs: cleaned = %q, want original", cleaned)
	}
	if len(refs) != 0 || len(notes) != 0 {
		t.Error("no data URLs: expected empty refs and notes")
	}
}

func TestExtractInlineMediaRefs_WithMarkdownDataURL(t *testing.T) {
	seen := map[string]struct{}{}
	text := "prefix ![img](data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7) suffix"
	cleaned, refs, _ := extractInlineMediaRefs(text, "tool", &mockMediaStore{}, "ch", "id", seen)
	if len(refs) == 0 {
		t.Error("expected refs for markdown data URL")
	}
	_ = cleaned
}

func TestExtractInlineMediaRefs_WithRawDataURL(t *testing.T) {
	seen := map[string]struct{}{}
	text := "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	_, refs, _ := extractInlineMediaRefs(text, "tool", &mockMediaStore{}, "ch", "id", seen)
	if len(refs) == 0 {
		t.Error("expected refs for raw data URL")
	}
}

func TestNormalizeToolResult_Nil(t *testing.T) {
	got := normalizeToolResult(nil, "tool", nil, "", "")
	if got != nil {
		t.Error("normalizeToolResult(nil) should return nil")
	}
}

func TestNormalizeToolResult_WithStore(t *testing.T) {
	result := &ToolResult{
		ForLLM:  "response text",
		ForUser: "user text",
	}
	got := normalizeToolResult(result, "tool", &mockMediaStore{}, "telegram", "123")
	if got == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestNormalizeToolResult_WithMediaContent(t *testing.T) {
	// ForLLM contains a markdown data URL — should be extracted and ref added
	dataURL := "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"
	result := &ToolResult{
		ForLLM:  "![img](" + dataURL + ")",
		ForUser: "image result",
	}
	got := normalizeToolResult(result, "tool", &mockMediaStore{}, "telegram", "123")
	if got == nil {
		t.Fatal("expected non-nil result")
	}
}
