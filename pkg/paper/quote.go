package paper

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuoteSource supplies prices to the deterministic paper engine.
type QuoteSource interface {
	Quote(ctx context.Context, symbol string) (Quote, error)
}

// ErrQuoteStale indicates that a structurally valid quote is too old to trade.
var ErrQuoteStale = fmt.Errorf("paper: quote is stale")

// ValidateQuoteFreshness rejects future and expired quotes.
func ValidateQuoteFreshness(quote Quote, now time.Time, maxAge time.Duration) error {
	if err := ValidateQuote(quote); err != nil {
		return err
	}
	if maxAge <= 0 {
		return invalid("max_quote_age", "must be positive")
	}
	age := now.Sub(quote.AsOf)
	if age < 0 {
		return invalid("as_of", "must not be in the future")
	}
	if age > maxAge {
		return fmt.Errorf("%w: age %s exceeds %s", ErrQuoteStale, age, maxAge)
	}
	return nil
}

// FakeQuoteSource is an in-memory quote source intended for deterministic tests
// and local development. It never performs network I/O.
type FakeQuoteSource struct {
	mu     sync.RWMutex
	quotes map[string]Quote
	err    error
}

func NewFakeQuoteSource(quotes ...Quote) *FakeQuoteSource {
	f := &FakeQuoteSource{quotes: make(map[string]Quote)}
	for _, quote := range quotes {
		f.SetQuote(quote)
	}
	return f
}

func (f *FakeQuoteSource) SetQuote(quote Quote) {
	f.mu.Lock()
	defer f.mu.Unlock()
	quote.Symbol = NormalizeSymbol(quote.Symbol)
	f.quotes[quote.Symbol] = quote
}

func (f *FakeQuoteSource) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *FakeQuoteSource) Quote(_ context.Context, symbol string) (Quote, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.err != nil {
		return Quote{}, f.err
	}
	quote, ok := f.quotes[NormalizeSymbol(symbol)]
	if !ok {
		return Quote{}, fmt.Errorf("%w: quote %q", ErrNotFound, symbol)
	}
	return quote, nil
}
