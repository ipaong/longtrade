package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

// TestScanDeltaNeutral_NameDescParams covers the trivial tool-surface methods.
func TestScanDeltaNeutral_NameDescParams(t *testing.T) {
	tool := NewScanDeltaNeutralOpportunitiesTool(&config.Config{})
	if tool.Name() != NameScanDeltaNeutralOpportunities {
		t.Fatalf("unexpected name %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("description must be non-empty")
	}
	if p := tool.Parameters(); p == nil || p["properties"] == nil {
		t.Fatal("parameters must define properties")
	}
}

// TestScanDeltaNeutral_MarketFilterAndStability drives the deeper scan path:
// active-swap market filtering, the batch-funding ranking, and Stage-2 stability
// (FetchPublicFundingRateHistory → computeFundingStatsWindow → formatScanResults
// with the stability columns).
func TestScanDeltaNeutral_MarketFilterAndStability(t *testing.T) {
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()
	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return []string{"BTC", "ETH", "DEAD"}, nil // DEAD has no active market → filtered
	}

	oldFuturesFn := futuresProviderFn
	defer func() { futuresProviderFn = oldFuturesFn }()

	interval := "8h"
	yes := true
	no := false
	mock := &mockFuturesProvider{
		loadMarketsFn: func(ctx context.Context) (map[string]ccxt.MarketInterface, error) {
			// BTC active swap; ETH active swap; DEAD inactive (filtered out).
			return map[string]ccxt.MarketInterface{
				"BTC/USDT:USDT":  {Active: &yes, Swap: &yes},
				"ETH/USDT:USDT":  {Active: &yes, Swap: &yes},
				"DEAD/USDT:USDT": {Active: &no, Swap: &yes},
			}, nil
		},
		fundingRatesFn: func(ctx context.Context, symbols []string) (map[string]ccxt.FundingRate, error) {
			b, e := 0.0003, -0.0001
			return map[string]ccxt.FundingRate{
				"BTC/USDT:USDT": {FundingRate: &b, Interval: &interval},
				"ETH/USDT:USDT": {FundingRate: &e, Interval: &interval},
			}, nil
		},
		fetchPublicFundingRateHistoryFn: func(ctx context.Context, symbol string, since *int64, limit int) ([]ccxt.FundingRateHistory, error) {
			now := time.Now().UTC().UnixMilli()
			rate := 0.0002
			hist := make([]ccxt.FundingRateHistory, 0, 10)
			for i := 0; i < 10; i++ {
				ts := now - int64(i)*8*3600*1000 // 8h apart, within 7d/14d windows
				r := rate
				hist = append(hist, ccxt.FundingRateHistory{Timestamp: &ts, FundingRate: &r})
			}
			return hist, nil
		},
	}
	futuresProviderFn = func(ctx context.Context, cfg *config.Config, providerID, account string) (broker.FuturesProvider, error) {
		return mock, nil
	}

	tool := NewScanDeltaNeutralOpportunitiesTool(&config.Config{})
	res := tool.Execute(context.Background(), map[string]any{
		"provider":          "binance",
		"include_stability": true,
		"top_k_stability":   5.0,
	})
	if res.IsError {
		t.Fatalf("unexpected error: %v", res.ForLLM)
	}
	out := res.ForUser
	// Filtered DEAD must not appear; BTC/ETH must.
	if strings.Contains(out, "DEAD") {
		t.Fatalf("inactive market DEAD should be filtered out:\n%s", out)
	}
	if !strings.Contains(out, "BTC") || !strings.Contains(out, "ETH") {
		t.Fatalf("expected BTC and ETH:\n%s", out)
	}
	// Stability columns present (Stage 2 ran).
	if !strings.Contains(out, "7d Mean%") {
		t.Fatalf("expected stability columns in output:\n%s", out)
	}
}

// TestScanDeltaNeutral_MinFundingFilter exercises the min_abs_funding_apr filter
// and the empty-result message path.
func TestScanDeltaNeutral_AllFilteredOut(t *testing.T) {
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()
	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return []string{"BTC"}, nil
	}
	oldFuturesFn := futuresProviderFn
	defer func() { futuresProviderFn = oldFuturesFn }()

	interval := "8h"
	mock := &mockFuturesProvider{
		loadMarketsFn: func(ctx context.Context) (map[string]ccxt.MarketInterface, error) { return nil, nil },
		fundingRatesFn: func(ctx context.Context, symbols []string) (map[string]ccxt.FundingRate, error) {
			small := 0.00001 // ~1% APR, below a high filter
			return map[string]ccxt.FundingRate{"BTC/USDT:USDT": {FundingRate: &small, Interval: &interval}}, nil
		},
	}
	futuresProviderFn = func(ctx context.Context, cfg *config.Config, providerID, account string) (broker.FuturesProvider, error) {
		return mock, nil
	}

	res := NewScanDeltaNeutralOpportunitiesTool(&config.Config{}).Execute(context.Background(), map[string]any{
		"provider":            "binance",
		"include_stability":   false,
		"min_abs_funding_apr": 50.0, // filters out the ~1% APR row
	})
	if res.IsError {
		t.Fatalf("unexpected error: %v", res.ForLLM)
	}
	if !strings.Contains(strings.ToLower(res.ForUser), "no opportunities") {
		t.Fatalf("expected empty-result message:\n%s", res.ForUser)
	}
}

// TestScanDeltaNeutral_AllProvidersFail covers the path where every requested
// provider errors out (combined error result).
func TestScanDeltaNeutral_AllProvidersFail(t *testing.T) {
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()
	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return []string{"BTC"}, nil
	}
	oldFuturesFn := futuresProviderFn
	defer func() { futuresProviderFn = oldFuturesFn }()
	futuresProviderFn = func(ctx context.Context, cfg *config.Config, providerID, account string) (broker.FuturesProvider, error) {
		return nil, context.DeadlineExceeded
	}

	res := NewScanDeltaNeutralOpportunitiesTool(&config.Config{}).Execute(context.Background(), map[string]any{
		"provider": "all",
	})
	if !res.IsError {
		t.Fatal("expected error when all providers fail")
	}
	if !strings.Contains(res.ForLLM, "scan failed for all providers") {
		t.Fatalf("expected combined-failure message, got: %s", res.ForLLM)
	}
}

// TestFetchCMCListing covers the real CMC HTTP fetch+paging+decode via httptest.
func TestFetchCMCListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// One page of 2 symbols, fewer than the 100 page size → loop terminates.
		_, _ = w.Write([]byte(`{"data":{"cryptoCurrencyList":[{"symbol":"BTC","cmcRank":1},{"symbol":"ETH","cmcRank":2}]}}`))
	}))
	defer srv.Close()

	syms, err := fetchCMCListing(context.Background(), srv.URL, 10)
	if err != nil {
		t.Fatalf("fetchCMCListing error: %v", err)
	}
	if len(syms) != 2 || syms[0] != "BTC" || syms[1] != "ETH" {
		t.Fatalf("unexpected symbols: %v", syms)
	}

	// Error path: non-200 status.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	if _, err := fetchCMCListing(context.Background(), bad.URL, 10); err == nil {
		t.Fatal("expected error on non-200 CMC response")
	}
}

// TestPeriodsPerDay covers all interval branches.
func TestPeriodsPerDay(t *testing.T) {
	cases := map[string]float64{"1h": 24, "4h": 6, "8h": 3, "": 3, "weird": 3}
	for iv, want := range cases {
		var p *string
		if iv != "" {
			s := iv
			p = &s
		}
		if got := periodsPerDay(p); got != want {
			t.Errorf("periodsPerDay(%q) = %v, want %v", iv, got, want)
		}
	}
}
