package channels

import (
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/bus"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestRegisterFactory_And_GetFactory(t *testing.T) {
	name := "test-channel-registry-unique"
	factory := func(cfg *config.Config, b *bus.MessageBus) (Channel, error) {
		return nil, nil
	}
	RegisterFactory(name, factory)

	got, ok := getFactory(name)
	if !ok {
		t.Fatal("getFactory should find registered factory")
	}
	if got == nil {
		t.Error("factory should not be nil")
	}
}

func TestGetFactory_NotFound(t *testing.T) {
	_, ok := getFactory("nonexistent-channel-xyz")
	if ok {
		t.Error("getFactory should return false for unknown name")
	}
}

func TestRegisterFactory_Overwrite(t *testing.T) {
	name := "test-channel-overwrite"
	var calls int
	first := func(cfg *config.Config, b *bus.MessageBus) (Channel, error) {
		calls = 1
		return nil, nil
	}
	second := func(cfg *config.Config, b *bus.MessageBus) (Channel, error) {
		calls = 2
		return nil, nil
	}
	RegisterFactory(name, first)
	RegisterFactory(name, second)

	got, ok := getFactory(name)
	if !ok {
		t.Fatal("factory should be found after re-registration")
	}
	got(nil, nil) //nolint:errcheck
	if calls != 2 {
		t.Errorf("expected second factory to be active, got calls=%d", calls)
	}
}
