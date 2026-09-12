package paper

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFakeQuoteSourceAndFreshness(t *testing.T) {
	now := testTime()
	source := NewFakeQuoteSource(Quote{Symbol: "xau/usd", Bid: 2400, Ask: 2401, AsOf: now})
	quote, err := source.Quote(context.Background(), SymbolXAUUSD)
	if err != nil || quote.Symbol != SymbolXAUUSD {
		t.Fatalf("Quote() = %#v, %v", quote, err)
	}
	if err := ValidateQuoteFreshness(quote, now.Add(31*time.Second), 30*time.Second); !errors.Is(err, ErrQuoteStale) {
		t.Fatalf("stale error = %v, want ErrQuoteStale", err)
	}
	if err := ValidateQuoteFreshness(quote, now.Add(-time.Second), 30*time.Second); err == nil {
		t.Fatal("future quote error = nil")
	}
	if _, err := source.Quote(context.Background(), "EURUSD"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing quote error = %v", err)
	}
}
