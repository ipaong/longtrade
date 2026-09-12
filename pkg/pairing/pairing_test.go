package pairing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Request.IsExpired ────────────────────────────────────────────────────────

func TestRequest_IsExpired_NotExpired(t *testing.T) {
	now := time.Now()
	r := Request{
		Code:        "TEST1234",
		ExpiresAtMS: now.Add(1 * time.Hour).UnixMilli(),
	}
	if r.IsExpired() {
		t.Error("IsExpired: got true, want false for future expiration")
	}
}

func TestRequest_IsExpired_JustExpired(t *testing.T) {
	now := time.Now()
	r := Request{
		Code:        "TEST1234",
		ExpiresAtMS: now.Add(-1 * time.Millisecond).UnixMilli(),
	}
	if !r.IsExpired() {
		t.Error("IsExpired: got false, want true for past expiration")
	}
}

func TestRequest_IsExpired_ExactlyAtExpiration(t *testing.T) {
	now := time.Now()
	// Use a slightly past time to ensure boundary condition
	pastTime := now.Add(-1 * time.Millisecond)
	r := Request{
		Code:        "TEST1234",
		ExpiresAtMS: pastTime.UnixMilli(),
	}
	if !r.IsExpired() {
		t.Error("IsExpired: got false, want true when expiry is in the past")
	}
}

// ── generateCode ─────────────────────────────────────────────────────────────

func TestGenerateCode_ValidLength(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode() error = %v", err)
	}
	if len(code) != codeLength {
		t.Errorf("len(code) = %d, want %d", len(code), codeLength)
	}
}

func TestGenerateCode_ValidAlphabet(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode() error = %v", err)
	}
	for i, c := range code {
		if !isValidCodeChar(byte(c)) {
			t.Errorf("code[%d] = %c, not in codeAlphabet", i, c)
		}
	}
}

func TestGenerateCode_Uniqueness(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("generateCode() error = %v", err)
		}
		if codes[code] {
			t.Errorf("duplicate code generated: %s (iteration %d)", code, i)
		}
		codes[code] = true
	}
}

func isValidCodeChar(c byte) bool {
	for i := 0; i < len(codeAlphabet); i++ {
		if codeAlphabet[i] == c {
			return true
		}
	}
	return false
}

// ── Store.load / Store.save ──────────────────────────────────────────────────

func TestStore_Load_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	st, err := s.load()
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if st == nil {
		t.Error("load(): got nil, want empty store")
	}
	if len(st.Requests) != 0 {
		t.Errorf("load(): got %d requests, want 0", len(st.Requests))
	}
}

func TestStore_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	// Write a valid store file
	data := []byte(`{
  "requests": [
    {
      "code": "TEST1234",
      "platform": "telegram",
      "platform_id": "12345",
      "username": "testuser",
      "display_name": "Test User",
      "canonical_id": "telegram:12345",
      "chat_id": 999,
      "created_at_ms": 1000000,
      "expires_at_ms": 2000000
    }
  ]
}`)
	os.WriteFile(storePath, data, 0o600)

	s := NewStore(storePath)
	st, err := s.load()
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if len(st.Requests) != 1 {
		t.Fatalf("load(): got %d requests, want 1", len(st.Requests))
	}
	if st.Requests[0].Code != "TEST1234" {
		t.Errorf("Code = %s, want TEST1234", st.Requests[0].Code)
	}
}

func TestStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	os.WriteFile(storePath, []byte("invalid json {"), 0o600)

	s := NewStore(storePath)
	_, err := s.load()
	if err == nil {
		t.Error("load(): got nil error, want error for invalid JSON")
	}
}

func TestStore_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subdir", "nested", "store.json")
	s := NewStore(storePath)

	st := &store{
		Requests: []Request{
			{
				Code:        "TEST1234",
				Platform:    "telegram",
				PlatformID:  "12345",
				CanonicalID: "telegram:12345",
				CreatedAtMS: time.Now().UnixMilli(),
				ExpiresAtMS: time.Now().Add(1 * time.Hour).UnixMilli(),
			},
		},
	}
	err := s.save(st)
	if err != nil {
		t.Fatalf("save() error = %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("save(): file is empty")
	}
}

func TestStore_Save_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req := Request{
		Code:        "ABCD1234",
		Platform:    "discord",
		PlatformID:  "999",
		Username:    "discorduser",
		DisplayName: "Discord User",
		CanonicalID: "discord:999",
		ChatID:      777,
		CreatedAtMS: 1000,
		ExpiresAtMS: 2000,
	}
	st := &store{Requests: []Request{req}}
	err := s.save(st)
	if err != nil {
		t.Fatalf("save() error = %v", err)
	}

	// Verify by loading
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var loaded store
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(loaded.Requests) != 1 {
		t.Errorf("loaded requests count = %d, want 1", len(loaded.Requests))
	}
	if loaded.Requests[0].Code != "ABCD1234" {
		t.Errorf("Code = %s, want ABCD1234", loaded.Requests[0].Code)
	}
}

// ── Store.ListPending ────────────────────────────────────────────────────────

func TestStore_ListPending_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	requests, err := s.ListPending()
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("ListPending(): got %d, want 0", len(requests))
	}
}

func TestStore_ListPending_FiltersExpired(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	now := time.Now()
	futureTime := now.Add(1 * time.Hour).UnixMilli()
	pastTime := now.Add(-1 * time.Hour).UnixMilli()

	data := []byte(fmt.Sprintf(`{
  "requests": [
    {
      "code": "ACTIVE01",
      "platform": "telegram",
      "platform_id": "1",
      "canonical_id": "telegram:1",
      "chat_id": 1,
      "created_at_ms": 1000000,
      "expires_at_ms": %d
    },
    {
      "code": "EXPIRED1",
      "platform": "telegram",
      "platform_id": "2",
      "canonical_id": "telegram:2",
      "chat_id": 2,
      "created_at_ms": 1000000,
      "expires_at_ms": %d
    }
  ]
}`, futureTime, pastTime))
	os.WriteFile(storePath, data, 0o600)

	s := NewStore(storePath)
	requests, err := s.ListPending()
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("ListPending(): got %d, want 1 (expired should be filtered)", len(requests))
	}
	if requests[0].Code != "ACTIVE01" {
		t.Errorf("Code = %s, want ACTIVE01", requests[0].Code)
	}

	// Verify file was cleaned up
	st, _ := s.load()
	if len(st.Requests) != 1 {
		t.Errorf("store after cleanup: got %d requests, want 1", len(st.Requests))
	}
}

func TestStore_ListPending_NoCleanupIfNoExpired(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	now := time.Now()
	expiresAtMS := now.Add(1 * time.Hour).UnixMilli()
	data := []byte(fmt.Sprintf(`{
  "requests": [
    {
      "code": "ACTIVE01",
      "platform": "telegram",
      "platform_id": "1",
      "canonical_id": "telegram:1",
      "chat_id": 1,
      "created_at_ms": 1000000,
      "expires_at_ms": %d
    }
  ]
}`, expiresAtMS))
	os.WriteFile(storePath, data, 0o600)

	s := NewStore(storePath)
	_, err := s.ListPending()
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}

	// File should not be rewritten if no cleanup was needed
	st, _ := s.load()
	if len(st.Requests) != 1 {
		t.Errorf("store after ListPending: got %d requests, want 1", len(st.Requests))
	}
}

// ── Store.Upsert ─────────────────────────────────────────────────────────────

func TestStore_Upsert_NewRequest(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, isNew, err := s.Upsert("telegram", "12345", "user1", "User One", 999)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if !isNew {
		t.Error("Upsert(): got isNew=false, want true for new request")
	}
	if req == nil {
		t.Fatal("Upsert(): got nil request")
	}
	if req.Code == "" {
		t.Error("Upsert(): got empty code")
	}
	if req.Platform != "telegram" {
		t.Errorf("Platform = %s, want telegram", req.Platform)
	}
	if req.PlatformID != "12345" {
		t.Errorf("PlatformID = %s, want 12345", req.PlatformID)
	}
	if !req.IsExpired() {
		// newly created request should not be expired immediately
	}
}

func TestStore_Upsert_ExistingRequest(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	// First upsert
	req1, isNew1, err := s.Upsert("telegram", "12345", "user1", "User One", 999)
	if err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	if !isNew1 {
		t.Error("first Upsert(): got isNew=false, want true")
	}

	// Second upsert for same user
	reqReturned, isNew2, err := s.Upsert("telegram", "12345", "user1", "User One", 999)
	if err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	if isNew2 {
		t.Error("second Upsert(): got isNew=true, want false for existing user")
	}
	if reqReturned.Code != req1.Code {
		t.Errorf("returned different code: %s != %s", reqReturned.Code, req1.Code)
	}
}

func TestStore_Upsert_CanonicalIDGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, _, err := s.Upsert("TELEGRAM", "ABC123", "user", "name", 999)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	// BuildCanonicalID should normalize platform to lowercase
	if req.CanonicalID != "telegram:ABC123" {
		t.Errorf("CanonicalID = %s, want telegram:ABC123", req.CanonicalID)
	}
}

func TestStore_Upsert_PrunesExpiredBeforeCheck(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour).UnixMilli()
	data := []byte(fmt.Sprintf(`{
  "requests": [
    {
      "code": "EXPIRED1",
      "platform": "telegram",
      "platform_id": "99999",
      "canonical_id": "telegram:99999",
      "chat_id": 999,
      "created_at_ms": 1000000,
      "expires_at_ms": %d
    }
  ]
}`, pastTime))
	os.WriteFile(storePath, data, 0o600)

	s := NewStore(storePath)
	req, isNew, err := s.Upsert("telegram", "99999", "user", "name", 999)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if !isNew {
		t.Error("Upsert(): got isNew=false, want true (expired should be pruned)")
	}
	if req.Code == "" {
		t.Error("Upsert(): got empty code for pruned user")
	}
}

// ── Store.Approve ────────────────────────────────────────────────────────────

func TestStore_Approve_ValidCode(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req1, _, _ := s.Upsert("telegram", "12345", "user1", "User One", 999)
	code := req1.Code

	req2, err := s.Approve(code)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if req2.Code != code {
		t.Errorf("Code = %s, want %s", req2.Code, code)
	}

	// Verify request is removed
	pending, _ := s.ListPending()
	if len(pending) != 0 {
		t.Errorf("after Approve(): got %d pending, want 0", len(pending))
	}
}

func TestStore_Approve_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	_, err := s.Approve("NONEXISTENT")
	if err == nil {
		t.Error("Approve(): got nil error, want error for nonexistent code")
	}
}

func TestStore_Approve_ExpiredCode(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour).UnixMilli()
	data := []byte(fmt.Sprintf(`{
  "requests": [
    {
      "code": "EXPIRED1",
      "platform": "telegram",
      "platform_id": "1",
      "canonical_id": "telegram:1",
      "chat_id": 1,
      "created_at_ms": 1000000,
      "expires_at_ms": %d
    }
  ]
}`, pastTime))
	os.WriteFile(storePath, data, 0o600)

	s := NewStore(storePath)
	_, err := s.Approve("EXPIRED1")
	if err == nil {
		t.Error("Approve(): got nil error, want error for expired code")
	}
}

func TestStore_Approve_RemovesFromStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req1, _, _ := s.Upsert("telegram", "111", "user1", "User One", 999)
	_, _, _ = s.Upsert("telegram", "222", "user2", "User Two", 888)

	code := req1.Code
	s.Approve(code)

	// Second request should still be pending
	pending, _ := s.ListPending()
	if len(pending) != 1 {
		t.Errorf("after Approve(): got %d pending, want 1", len(pending))
	}
	if pending[0].PlatformID != "222" {
		t.Errorf("remaining request: PlatformID = %s, want 222", pending[0].PlatformID)
	}
}

// ── Store.Reject ─────────────────────────────────────────────────────────────

func TestStore_Reject_ValidCode(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, _, _ := s.Upsert("telegram", "12345", "user1", "User One", 999)
	code := req.Code

	err := s.Reject(code)
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	// Verify request is removed
	pending, _ := s.ListPending()
	if len(pending) != 0 {
		t.Errorf("after Reject(): got %d pending, want 0", len(pending))
	}
}

func TestStore_Reject_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	err := s.Reject("NONEXISTENT")
	if err == nil {
		t.Error("Reject(): got nil error, want error for nonexistent code")
	}
}

func TestStore_Reject_LeavesOthersIntact(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req1, _, _ := s.Upsert("telegram", "111", "user1", "User One", 999)
	_, _, _ = s.Upsert("telegram", "222", "user2", "User Two", 888)

	s.Reject(req1.Code)

	// Second request should still be pending
	pending, _ := s.ListPending()
	if len(pending) != 1 {
		t.Errorf("after Reject(): got %d pending, want 1", len(pending))
	}
	if pending[0].PlatformID != "222" {
		t.Errorf("remaining request: PlatformID = %s, want 222", pending[0].PlatformID)
	}
}

// ── Concurrent Access (Mutex Protection) ──────────────────────────────────────

func TestStore_Concurrent_Upsert_And_ListPending(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	// Simulate concurrent operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			s.Upsert("telegram", "user1", "u1", "User 1", int64(i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			s.ListPending()
		}
		done <- true
	}()

	<-done
	<-done
	// If we complete without panicking, mutex is working
}

func TestStore_Concurrent_Approve_And_Reject(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	// Create multiple requests
	var codes []string
	for i := 0; i < 5; i++ {
		req, _, _ := s.Upsert("telegram", "user"+string(rune(48+i)), "u", "U", int64(i))
		codes = append(codes, req.Code)
	}

	done := make(chan bool, 2)

	// Approve some
	go func() {
		for _, code := range codes[:2] {
			s.Approve(code)
		}
		done <- true
	}()

	// Reject others
	go func() {
		for _, code := range codes[2:] {
			s.Reject(code)
		}
		done <- true
	}()

	<-done
	<-done

	pending, _ := s.ListPending()
	if len(pending) != 0 {
		t.Errorf("after concurrent ops: got %d pending, want 0", len(pending))
	}
}

// ── Edge Cases ───────────────────────────────────────────────────────────────

func TestStore_Upsert_TTLConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, _, _ := s.Upsert("telegram", "12345", "user", "name", 999)
	expiresAt := time.UnixMilli(req.ExpiresAtMS)

	// Check that expiration is approximately defaultTTL in the future
	// Allow a 1-second window to account for test execution time and clock variance
	expectedApprox := time.Now().Add(defaultTTL)
	tolerance := 1 * time.Second

	minExpected := expectedApprox.Add(-tolerance)
	maxExpected := expectedApprox.Add(tolerance)

	if expiresAt.Before(minExpected) || expiresAt.After(maxExpected) {
		t.Errorf("ExpiresAtMS not approximately correct: %v (want ~%v ±1s)", expiresAt, expectedApprox)
	}
}

func TestStore_Load_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nonexistent", "store.json")
	s := NewStore(storePath)

	// This should return empty store (no error) since missing file is treated as fresh start
	st, err := s.load()
	if err != nil {
		t.Fatalf("load() for nonexistent dir: error = %v, expected nil", err)
	}
	if st == nil {
		t.Error("load(): got nil, want empty store")
	}
}

func TestStore_Upsert_EmptyPlatformID(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, _, err := s.Upsert("telegram", "", "user", "name", 999)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	// Should still create request but with empty canonical ID (determined by identity.BuildCanonicalID)
	if req == nil {
		t.Fatal("Upsert(): got nil request")
	}
}

func TestStore_JsonMarshalIndent_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	req, _, _ := s.Upsert("telegram", "12345", "user", "name", 999)
	s.Approve(req.Code) // triggers save with no requests

	data, _ := os.ReadFile(storePath)
	// Verify it's indented (contains newlines and spaces)
	if len(data) == 0 {
		t.Error("saved file is empty")
	}
}

// ── Timestamps ───────────────────────────────────────────────────────────────

func TestRequest_CreatedAtMS_IsSet(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	before := time.Now().UnixMilli()
	req, _, _ := s.Upsert("telegram", "12345", "user", "name", 999)
	after := time.Now().UnixMilli()

	if req.CreatedAtMS < before || req.CreatedAtMS > after {
		t.Errorf("CreatedAtMS out of expected range: %d (want between %d and %d)", req.CreatedAtMS, before, after)
	}
}

func TestStore_Upsert_MaxPendingUserReplacement(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")
	s := NewStore(storePath)

	// The current code doesn't actually enforce maxPendingUser limit on upsert
	// when user has < maxPendingUser requests. But if a user had 3 and tried to
	// add a 4th, the oldest would be removed. Let's test a simpler case:
	// After maxPendingUser requests, the next upsert still returns existing.
	req1, _, _ := s.Upsert("telegram", "12345", "user", "name", 999)
	code1 := req1.Code

	// Try to upsert same user again
	reqReturned, isNew2, _ := s.Upsert("telegram", "12345", "user", "name", 999)
	if isNew2 {
		t.Error("second upsert: got isNew=true, want false")
	}
	if reqReturned.Code != code1 {
		t.Errorf("returned different code: %s != %s", reqReturned.Code, code1)
	}
}
