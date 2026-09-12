package paper

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func newTestEngine(t *testing.T, bid, ask float64) (*Engine, *Store, *FakeQuoteSource) {
	t.Helper()
	store, _ := newTestStore(t)
	now := testTime()
	source := NewFakeQuoteSource(Quote{Symbol: SymbolXAUUSD, Bid: bid, Ask: ask, AsOf: now})
	engine, err := NewEngine(store, source)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	engine.SetClock(func() time.Time { return now })
	return engine, store, source
}

func TestEngineConcurrentOpenUsesClientRequestIDOnce(t *testing.T) {
	engine, store, _ := newTestEngine(t, 100, 101)
	ctx := context.Background()
	const callers = 12
	ids := make(chan string, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			position, err := engine.OpenPosition(ctx, openRequest("concurrent-open", SideBuy))
			if err == nil {
				ids <- position.ID
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("OpenPosition() error = %v", err)
		}
	}
	var want string
	for id := range ids {
		if want == "" {
			want = id
		}
		if id != want {
			t.Fatalf("position ID = %q, want %q", id, want)
		}
	}
	orders, err := store.ListRecentOrders(ctx, DefaultAccountID, 20)
	if err != nil || len(orders) != 1 {
		t.Fatalf("orders = %#v, error %v", orders, err)
	}
}

func openRequest(id string, side Side) OpenPositionRequest {
	return OpenPositionRequest{ClientRequestID: id, Symbol: SymbolXAUUSD, Side: side, VolumeLots: .01}
}

func TestEngineOpenPositionIsAtomicAndIdempotent(t *testing.T) {
	engine, store, _ := newTestEngine(t, 2400, 2401)
	ctx := context.Background()
	position, err := engine.OpenPosition(ctx, openRequest("open-1", SideBuy))
	if err != nil {
		t.Fatalf("OpenPosition() error = %v", err)
	}
	if position.EntryPrice != 2401 || position.Status != PositionStatusOpen {
		t.Fatalf("position = %#v", position)
	}
	repeated, err := engine.OpenPosition(ctx, openRequest("open-1", SideBuy))
	if err != nil || repeated.ID != position.ID {
		t.Fatalf("repeated = %#v, %v", repeated, err)
	}
	orders, _ := store.ListRecentOrders(ctx, DefaultAccountID, 20)
	if len(orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(orders))
	}
	events, _ := store.ListAuditEvents(ctx, DefaultAccountID, 20)
	if len(events) != 1 || events[0].Action != "position.opened" {
		t.Fatalf("events = %#v", events)
	}
}

func TestEngineOpenPositionRiskChecks(t *testing.T) {
	tests := []struct {
		name      string
		request   OpenPositionRequest
		limits    RiskLimits
		wantField string
	}{
		{name: "lot step", request: OpenPositionRequest{ClientRequestID: "bad-lot", Symbol: SymbolXAUUSD, Side: SideBuy, VolumeLots: .015}, limits: DefaultRiskLimits(), wantField: "volume_lots"},
		{name: "balance", request: OpenPositionRequest{ClientRequestID: "too-large", Symbol: SymbolXAUUSD, Side: SideBuy, VolumeLots: 1}, limits: DefaultRiskLimits(), wantField: "balance"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, _, _ := newTestEngine(t, 2400, 2401)
			_ = engine.SetRiskLimits(tt.limits)
			_, err := engine.OpenPosition(context.Background(), tt.request)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Field != tt.wantField {
				t.Fatalf("error = %v, want field %s", err, tt.wantField)
			}
		})
	}

	engine, _, _ := newTestEngine(t, 100, 101)
	limits := DefaultRiskLimits()
	limits.MaxOpenPositions = 1
	_ = engine.SetRiskLimits(limits)
	if _, err := engine.OpenPosition(context.Background(), openRequest("first", SideBuy)); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.OpenPosition(context.Background(), openRequest("second", SideSell)); err == nil {
		t.Fatal("position limit error = nil")
	}

	engine, _, source := newTestEngine(t, 100, 101)
	limits = DefaultRiskLimits()
	limits.MaxDailyLossUSD = 1
	if err := engine.SetRiskLimits(limits); err != nil {
		t.Fatal(err)
	}
	position, err := engine.OpenPosition(context.Background(), openRequest("loss-open", SideBuy))
	if err != nil {
		t.Fatal(err)
	}
	source.SetQuote(Quote{Symbol: SymbolXAUUSD, Bid: 99, Ask: 100, AsOf: testTime()})
	if _, err := engine.ClosePosition(context.Background(), position.ID, "loss-close", CloseReasonManual); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.OpenPosition(context.Background(), openRequest("after-loss", SideBuy)); err == nil {
		t.Fatal("daily loss limit error = nil")
	}
}

func TestEngineSnapshotValuesBuyAndSellAtExecutablePrices(t *testing.T) {
	engine, _, source := newTestEngine(t, 2400, 2401)
	ctx := context.Background()
	if _, err := engine.OpenPosition(ctx, openRequest("buy", SideBuy)); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.OpenPosition(ctx, openRequest("sell", SideSell)); err != nil {
		t.Fatal(err)
	}
	source.SetQuote(Quote{Symbol: SymbolXAUUSD, Bid: 2410, Ask: 2411, AsOf: testTime()})
	snapshot, err := engine.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// BUY: (2410-2401)*1 = 9; SELL: (2400-2411)*1 = -11.
	if snapshot.UnrealizedPnL != -2 || snapshot.Equity != DefaultStartingBalance-2 || snapshot.AsOf != testTime() {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestEngineClosePositionUpdatesBalanceAndIsIdempotent(t *testing.T) {
	engine, store, source := newTestEngine(t, 2400, 2401)
	ctx := context.Background()
	position, _ := engine.OpenPosition(ctx, openRequest("open", SideBuy))
	source.SetQuote(Quote{Symbol: SymbolXAUUSD, Bid: 2411, Ask: 2412, AsOf: testTime()})
	trade, err := engine.ClosePosition(ctx, position.ID, "close", CloseReasonManual)
	if err != nil {
		t.Fatal(err)
	}
	if trade.RealizedPnL != 10 {
		t.Fatalf("P/L = %.2f, want 10", trade.RealizedPnL)
	}
	repeated, err := engine.ClosePosition(ctx, position.ID, "close", CloseReasonManual)
	if err != nil || repeated.ID != trade.ID {
		t.Fatalf("repeat = %#v, %v", repeated, err)
	}
	account, _ := store.GetAccount(ctx, DefaultAccountID)
	if account.Balance != 10010 {
		t.Fatalf("balance = %.2f", account.Balance)
	}
	snapshot, _ := engine.Snapshot(ctx)
	if snapshot.DailyPnL != 10 || len(snapshot.RecentTrades) != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestEngineProtectionUpdateAndTriggers(t *testing.T) {
	for _, tt := range []struct {
		name                             string
		side                             Side
		bid, ask, triggerBid, triggerAsk float64
		sl, tp                           float64
		reason                           CloseReason
	}{
		{"buy take profit", SideBuy, 100, 101, 105, 106, 95, 105, CloseReasonTakeProfit},
		{"sell stop loss", SideSell, 100, 101, 105, 106, 105, 95, CloseReasonStopLoss},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine, _, source := newTestEngine(t, tt.bid, tt.ask)
			ctx := context.Background()
			position, err := engine.OpenPosition(ctx, openRequest("open", tt.side))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := engine.UpdateProtection(ctx, position.ID, &tt.sl, &tt.tp); err != nil {
				t.Fatal(err)
			}
			source.SetQuote(Quote{Symbol: SymbolXAUUSD, Bid: tt.triggerBid, Ask: tt.triggerAsk, AsOf: testTime()})
			trades, err := engine.EvaluateProtections(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(trades) != 1 || trades[0].CloseReason != tt.reason {
				t.Fatalf("trades = %#v", trades)
			}
		})
	}
}

func TestEngineEmergencyCloseOnlyOpenPositionsAndAudits(t *testing.T) {
	engine, store, _ := newTestEngine(t, 100, 101)
	ctx := context.Background()
	if _, err := engine.OpenPosition(ctx, openRequest("one", SideBuy)); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.OpenPosition(ctx, openRequest("two", SideSell)); err != nil {
		t.Fatal(err)
	}
	trades, err := engine.EmergencyClose(ctx)
	if err != nil || len(trades) != 2 {
		t.Fatalf("trades = %#v, %v", trades, err)
	}
	again, err := engine.EmergencyClose(ctx)
	if err != nil || len(again) != 0 {
		t.Fatalf("second close = %#v, %v", again, err)
	}
	positions, _ := store.ListOpenPositions(ctx, DefaultAccountID)
	if len(positions) != 0 {
		t.Fatalf("open = %d", len(positions))
	}
	events, _ := store.ListAuditEvents(ctx, DefaultAccountID, 20)
	if len(events) != 4 {
		t.Fatalf("audit events = %d, want 4", len(events))
	}
	emergencyEvents := 0
	for _, event := range events {
		if event.Action == "position.closed.emergency" {
			emergencyEvents++
		}
	}
	if emergencyEvents != 2 {
		t.Fatalf("emergency audit events = %d, want 2", emergencyEvents)
	}
}
