package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetOpenOrders_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetOpenOrders_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetOpenOrders_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOpenOrders_WithSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOpenOrders_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOpenOrders_WithSymbolAndAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOpenOrdersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestPtr_Nil(t *testing.T) {
	if got := ptr(nil); got != "-" {
		t.Errorf("ptr(nil) = %q, want -", got)
	}
}

func TestPtr_NonNil(t *testing.T) {
	s := "hello"
	if got := ptr(&s); got != "hello" {
		t.Errorf("ptr(&s) = %q, want hello", got)
	}
}

func TestFmtFloat_Nil(t *testing.T) {
	if got := fmtFloat(nil); got != "-" {
		t.Errorf("fmtFloat(nil) = %q, want -", got)
	}
}

func TestFmtFloat_NonNil(t *testing.T) {
	f := 1.5
	got := fmtFloat(&f)
	if got == "-" || got == "" {
		t.Errorf("fmtFloat(&1.5) = %q, expected a formatted number", got)
	}
}

func TestTern_True(t *testing.T) {
	if got := tern(true, "yes", "no"); got != "yes" {
		t.Errorf("tern(true) = %q, want yes", got)
	}
}

func TestTern_False(t *testing.T) {
	if got := tern(false, "yes", "no"); got != "no" {
		t.Errorf("tern(false) = %q, want no", got)
	}
}

func TestGetOpenOrders_ParametersSchema(t *testing.T) {
	tool := NewGetOpenOrdersTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// Only provider should be required
	if len(required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(required))
	}
}

func TestGetOpenOrders_Name(t *testing.T) {
	tool := NewGetOpenOrdersTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetOpenOrders {
		t.Errorf("Name() = %q, want %q", name, NameGetOpenOrders)
	}
}

func TestGetOpenOrders_Description(t *testing.T) {
	tool := NewGetOpenOrdersTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}
