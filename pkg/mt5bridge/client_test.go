package mt5bridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadOnlyContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/health":
			_, _ = w.Write([]byte(`{"Status":"ok","Version":"v1","Terminal":"XM Demo"}`))
		case "/api/v1/account":
			_, _ = w.Write([]byte(`{"login":42,"currency":"USD","balance":10000,"equity":10010}`))
		case "/api/v1/quote":
			_, _ = w.Write([]byte(`{"symbol":"XAUUSD","bid":2400,"ask":2401,"as_of":"2026-09-12T00:00:00Z"}`))
		case "/api/v1/positions":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if h, err := client.Health(context.Background()); err != nil || h.Status != "ok" {
		t.Fatalf("Health = %#v, %v", h, err)
	}
	if a, err := client.Account(context.Background()); err != nil || a.Login != 42 {
		t.Fatalf("Account = %#v, %v", a, err)
	}
	if q, err := client.Quote(context.Background(), "XAUUSD"); err != nil || q.Ask != 2401 {
		t.Fatalf("Quote = %#v, %v", q, err)
	}
	if p, err := client.Positions(context.Background()); err != nil || len(p) != 0 {
		t.Fatalf("Positions = %#v, %v", p, err)
	}
}

func TestRejectsInvalidURL(t *testing.T) {
	if _, err := New("localhost:8000", 0); err == nil {
		t.Fatal("New() error = nil")
	}
}
