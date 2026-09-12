package credential

import (
	"sync"
	"testing"
)

// TestNewSecureStore creates a new SecureStore.
func TestNewSecureStore(t *testing.T) {
	store := NewSecureStore()
	if store == nil {
		t.Fatal("NewSecureStore should not return nil")
	}
	if store.IsSet() {
		t.Error("new SecureStore should be empty")
	}
}

// TestSecureStore_SetString_Get tests basic Set and Get operations.
func TestSecureStore_SetString_Get(t *testing.T) {
	store := NewSecureStore()
	passphrase := "test-passphrase-123"

	store.SetString(passphrase)
	retrieved := store.Get()

	if retrieved != passphrase {
		t.Errorf("Get() = %q, want %q", retrieved, passphrase)
	}
}

// TestSecureStore_SetString_Empty tests that SetString with empty string clears the store.
func TestSecureStore_SetString_Empty(t *testing.T) {
	store := NewSecureStore()
	store.SetString("initial-value")

	if !store.IsSet() {
		t.Error("store should be set after SetString")
	}

	store.SetString("")
	if store.IsSet() {
		t.Error("store should be empty after SetString with empty string")
	}
	if val := store.Get(); val != "" {
		t.Errorf("Get() = %q, want empty", val)
	}
}

// TestSecureStore_IsSet_InitialEmpty tests that IsSet is false on new store.
func TestSecureStore_IsSet_InitialEmpty(t *testing.T) {
	store := NewSecureStore()
	if store.IsSet() {
		t.Error("new store should not be set")
	}
}

// TestSecureStore_IsSet_AfterSet tests that IsSet is true after SetString.
func TestSecureStore_IsSet_AfterSet(t *testing.T) {
	store := NewSecureStore()
	store.SetString("value")
	if !store.IsSet() {
		t.Error("store should be set after SetString")
	}
}

// TestSecureStore_Get_DefaultEmpty tests that Get returns empty for unset store.
func TestSecureStore_Get_DefaultEmpty(t *testing.T) {
	store := NewSecureStore()
	val := store.Get()
	if val != "" {
		t.Errorf("Get() on empty store = %q, want empty", val)
	}
}

// TestSecureStore_Clear tests that Clear removes the stored value.
func TestSecureStore_Clear(t *testing.T) {
	store := NewSecureStore()
	store.SetString("secret-value")

	if !store.IsSet() {
		t.Error("store should be set before Clear")
	}

	store.Clear()

	if store.IsSet() {
		t.Error("store should be empty after Clear")
	}
	if val := store.Get(); val != "" {
		t.Errorf("Get() after Clear = %q, want empty", val)
	}
}

// TestSecureStore_MultipleWrites tests multiple consecutive writes.
func TestSecureStore_MultipleWrites(t *testing.T) {
	store := NewSecureStore()

	values := []string{"first", "second", "third"}
	for _, val := range values {
		store.SetString(val)
		if retrieved := store.Get(); retrieved != val {
			t.Errorf("Get() = %q, want %q", retrieved, val)
		}
	}

	// Final value should be "third"
	if final := store.Get(); final != "third" {
		t.Errorf("final Get() = %q, want 'third'", final)
	}
}

// TestSecureStore_SetClearCycle tests alternating Set and Clear.
func TestSecureStore_SetClearCycle(t *testing.T) {
	store := NewSecureStore()

	// Set and clear twice
	store.SetString("pass1")
	if !store.IsSet() {
		t.Error("should be set after SetString")
	}
	store.Clear()
	if store.IsSet() {
		t.Error("should not be set after Clear")
	}

	store.SetString("pass2")
	if !store.IsSet() {
		t.Error("should be set again after SetString")
	}
	if val := store.Get(); val != "pass2" {
		t.Errorf("Get() = %q, want 'pass2'", val)
	}
	store.Clear()
	if store.IsSet() {
		t.Error("should not be set after second Clear")
	}
}

// TestSecureStore_EmptyStringDistinct tests that empty string is treated as unset.
func TestSecureStore_EmptyStringDistinct(t *testing.T) {
	store := NewSecureStore()

	// Set to empty string
	store.SetString("")
	if store.IsSet() {
		t.Error("empty string should result in IsSet = false")
	}

	// Set to a value
	store.SetString("value")
	if !store.IsSet() {
		t.Error("non-empty string should result in IsSet = true")
	}

	// Set back to empty
	store.SetString("")
	if store.IsSet() {
		t.Error("empty string should result in IsSet = false")
	}
}

// TestSecureStore_LongPassphrase tests with a long passphrase.
func TestSecureStore_LongPassphrase(t *testing.T) {
	store := NewSecureStore()
	longPass := ""
	for i := 0; i < 1000; i++ {
		longPass += "x"
	}

	store.SetString(longPass)
	if retrieved := store.Get(); retrieved != longPass {
		t.Errorf("Get() length = %d, want %d", len(retrieved), len(longPass))
	}
}

// TestSecureStore_SpecialCharacters tests with special characters.
func TestSecureStore_SpecialCharacters(t *testing.T) {
	store := NewSecureStore()
	specials := []string{
		"pass\nwith\nnewlines",
		"pass\twith\ttabs",
		"pass with spaces",
		"παθ-ωορδ",
		"пароль",
		"密码",
		"!@#$%^&*()_+-=[]{}|;:',.<>?/",
	}

	for _, pass := range specials {
		store.SetString(pass)
		if retrieved := store.Get(); retrieved != pass {
			t.Errorf("Get() for special chars: got %q, want %q", retrieved, pass)
		}
	}
}

// TestSecureStore_Concurrent_Write_Read tests concurrent writes and reads.
func TestSecureStore_Concurrent_Write_Read(t *testing.T) {
	store := NewSecureStore()
	const numGoroutines = 10
	const numOps = 20

	var wg sync.WaitGroup

	// Writers
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < numOps; i++ {
				val := "pass_" + string(rune(g*numOps+i))
				store.SetString(val)
			}
		}(g)
	}

	// Readers (concurrently)
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < numOps; i++ {
				_ = store.Get()
				_ = store.IsSet()
			}
		}(g)
	}

	wg.Wait()

	// Store should still be functional after concurrent ops
	store.SetString("final-test")
	if retrieved := store.Get(); retrieved != "final-test" {
		t.Errorf("store broken after concurrent ops: got %q", retrieved)
	}
}

// TestSecureStore_Concurrent_Clear tests concurrent Clear operations.
func TestSecureStore_Concurrent_Clear(t *testing.T) {
	store := NewSecureStore()
	store.SetString("initial")

	const numGoroutines = 10

	var wg sync.WaitGroup
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Clear()
			_ = store.IsSet()
			store.SetString("temp")
		}()
	}

	wg.Wait()

	// The final state after all concurrent clears should be that
	// last goroutine either cleared or set, but store is still usable
	_ = store.Get()
	store.SetString("post-concurrent")
	if retrieved := store.Get(); retrieved != "post-concurrent" {
		t.Errorf("store broken after concurrent clears: got %q", retrieved)
	}
}

// TestSecureStore_SetAndGetSequence tests a sequence of Set/Get/Clear operations.
func TestSecureStore_SetAndGetSequence(t *testing.T) {
	store := NewSecureStore()

	sequence := []struct {
		op    string
		value string
		want  string
		isset bool
	}{
		{"set", "pass1", "pass1", true},
		{"get", "", "pass1", true},
		{"get", "", "pass1", true},
		{"set", "pass2", "pass2", true},
		{"get", "", "pass2", true},
		{"clear", "", "", false},
		{"get", "", "", false},
		{"set", "pass3", "pass3", true},
	}

	for _, step := range sequence {
		switch step.op {
		case "set":
			store.SetString(step.value)
		case "get":
			val := store.Get()
			if val != step.want {
				t.Errorf("Get() = %q, want %q", val, step.want)
			}
		case "clear":
			store.Clear()
		}

		if set := store.IsSet(); set != step.isset {
			t.Errorf("after %s: IsSet() = %v, want %v", step.op, set, step.isset)
		}
	}
}
