package credential

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	var originalPtr uintptr
	if PassphraseProvider != nil {
		originalPtr = reflect.ValueOf(PassphraseProvider).Pointer()
	}

	code := m.Run()

	var currentPtr uintptr
	if PassphraseProvider != nil {
		currentPtr = reflect.ValueOf(PassphraseProvider).Pointer()
	}

	if originalPtr != currentPtr {
		fmt.Fprintln(os.Stderr, "LEAK: PassphraseProvider was not restored by a test")
		os.Exit(1)
	}
	os.Exit(code)
}
