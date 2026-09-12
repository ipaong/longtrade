package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cryptoquantumwave/khunquant/pkg/paper"
)

func setupPaperTestEnv(t *testing.T) (*Handler, *http.ServeMux, *paper.Engine, *paper.Store, *paper.FakeQuoteSource) {
	t.Helper()
	dir := t.TempDir()
	store, err := paper.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	quote := paper.Quote{
		Symbol: paper.SymbolXAUUSD,
		Bid:    2400.00,
		Ask:    2400.50,
		AsOf:   time.Now().UTC(),
	}
	quotes := paper.NewFakeQuoteSource(quote)
	engine, err := paper.NewEngine(store, quotes)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	configPath := filepath.Join(dir, "config.json")
	h := NewHandler(configPath)
	h.SetPaperEngine(engine, store)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	return h, mux, engine, store, quotes
}

func TestPaperAccountAndReset(t *testing.T) {
	_, mux, _, _, _ := setupPaperTestEnv(t)

	// GET /api/paper/account
	req := httptest.NewRequest(http.MethodGet, "/api/paper/account", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/paper/account status = %d; want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var snapshot paper.AccountSnapshot
	if err := json.NewDecoder(rec.Body).Decode(&snapshot); err != nil {
		t.Fatalf("failed to decode account snapshot: %v", err)
	}
	if snapshot.Account.Balance != 10000.0 {
		t.Errorf("balance = %f; want 10000.0", snapshot.Account.Balance)
	}
	if snapshot.Account.Currency != "USD" {
		t.Errorf("currency = %q; want 'USD'", snapshot.Account.Currency)
	}

	// POST /api/paper/account/reset
	resetReq := httptest.NewRequest(http.MethodPost, "/api/paper/account/reset", bytes.NewBufferString(`{}`))
	resetRec := httptest.NewRecorder()
	mux.ServeHTTP(resetRec, resetReq)

	if resetRec.Code != http.StatusOK {
		t.Fatalf("POST /api/paper/account/reset status = %d; want %d; body = %s", resetRec.Code, http.StatusOK, resetRec.Body.String())
	}
}

func TestPaperPositionsAndTrades(t *testing.T) {
	_, mux, _, _, _ := setupPaperTestEnv(t)

	// GET /api/paper/positions
	posReq := httptest.NewRequest(http.MethodGet, "/api/paper/positions", nil)
	posRec := httptest.NewRecorder()
	mux.ServeHTTP(posRec, posReq)

	if posRec.Code != http.StatusOK {
		t.Fatalf("GET /api/paper/positions status = %d; want %d", posRec.Code, http.StatusOK)
	}

	var positions []paper.Position
	if err := json.NewDecoder(posRec.Body).Decode(&positions); err != nil {
		t.Fatalf("failed to decode positions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len(positions) = %d; want 0", len(positions))
	}

	// GET /api/paper/trades
	tradeReq := httptest.NewRequest(http.MethodGet, "/api/paper/trades?limit=10", nil)
	tradeRec := httptest.NewRecorder()
	mux.ServeHTTP(tradeRec, tradeReq)

	if tradeRec.Code != http.StatusOK {
		t.Fatalf("GET /api/paper/trades status = %d; want %d", tradeRec.Code, http.StatusOK)
	}

	var trades []paper.Trade
	if err := json.NewDecoder(tradeRec.Body).Decode(&trades); err != nil {
		t.Fatalf("failed to decode trades: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("len(trades) = %d; want 0", len(trades))
	}
}

func TestPaperTradeWorkflow(t *testing.T) {
	_, mux, _, _, _ := setupPaperTestEnv(t)

	// 1. Prepare trade proposal
	prepareBody := `{"symbol": "XAUUSD", "side": "BUY", "volume_lots": 0.01, "stop_loss": 2390.0, "take_profit": 2420.0}`
	prepReq := httptest.NewRequest(http.MethodPost, "/api/paper/trade/prepare", bytes.NewBufferString(prepareBody))
	prepRec := httptest.NewRecorder()
	mux.ServeHTTP(prepRec, prepReq)

	if prepRec.Code != http.StatusOK {
		t.Fatalf("POST /api/paper/trade/prepare status = %d; want %d; body = %s", prepRec.Code, http.StatusOK, prepRec.Body.String())
	}

	var proposal paper.Proposal
	if err := json.NewDecoder(prepRec.Body).Decode(&proposal); err != nil {
		t.Fatalf("failed to decode proposal: %v", err)
	}
	if proposal.ID == "" {
		t.Fatalf("expected non-empty proposal ID")
	}
	if proposal.OpenPrice != 2400.50 { // ask price for BUY
		t.Errorf("proposal.OpenPrice = %f; want 2400.50", proposal.OpenPrice)
	}

	// 2. Confirm trade
	confirmBody, _ := json.Marshal(map[string]string{
		"proposal_id":       proposal.ID,
		"client_request_id": "test_req_001",
	})
	confReq := httptest.NewRequest(http.MethodPost, "/api/paper/trade/confirm", bytes.NewReader(confirmBody))
	confRec := httptest.NewRecorder()
	mux.ServeHTTP(confRec, confReq)

	if confRec.Code != http.StatusOK {
		t.Fatalf("POST /api/paper/trade/confirm status = %d; want %d; body = %s", confRec.Code, http.StatusOK, confRec.Body.String())
	}

	var confResp struct {
		Position paper.Position `json:"position"`
	}
	if err := json.NewDecoder(confRec.Body).Decode(&confResp); err != nil {
		t.Fatalf("failed to decode confirm response: %v", err)
	}
	if confResp.Position.ID == "" {
		t.Fatalf("expected created position ID")
	}
	if confResp.Position.VolumeLots != 0.01 {
		t.Errorf("position.VolumeLots = %f; want 0.01", confResp.Position.VolumeLots)
	}

	// 3. Confirm again with the same proposal ID should return 410 GONE (proposal consumed)
	confAgainReq := httptest.NewRequest(http.MethodPost, "/api/paper/trade/confirm", bytes.NewReader(confirmBody))
	confAgainRec := httptest.NewRecorder()
	mux.ServeHTTP(confAgainRec, confAgainReq)

	if confAgainRec.Code != http.StatusGone {
		t.Errorf("second confirm status = %d; want %d (StatusGone)", confAgainRec.Code, http.StatusGone)
	}
}

func TestPaperPositionProtectionAndClose(t *testing.T) {
	_, mux, engine, _, _ := setupPaperTestEnv(t)

	// Create position directly via engine
	sl := 2380.0
	tp := 2430.0
	pos, err := engine.OpenPosition(context.Background(), paper.OpenPositionRequest{
		Symbol:          paper.SymbolXAUUSD,
		Side:            paper.SideBuy,
		VolumeLots:      0.01,
		StopLoss:        &sl,
		TakeProfit:      &tp,
		ClientRequestID: "pos_test_01",
	})
	if err != nil {
		t.Fatalf("OpenPosition failed: %v", err)
	}

	// Update protection: change SL & TP
	newSL := 2390.0
	newTP := 2440.0
	protectBody, _ := json.Marshal(map[string]*float64{
		"stop_loss":   &newSL,
		"take_profit": &newTP,
	})
	protReq := httptest.NewRequest(http.MethodPost, "/api/paper/positions/"+pos.ID+"/protection", bytes.NewReader(protectBody))
	protRec := httptest.NewRecorder()
	mux.ServeHTTP(protRec, protReq)

	if protRec.Code != http.StatusOK {
		t.Fatalf("POST protection status = %d; want %d; body = %s", protRec.Code, http.StatusOK, protRec.Body.String())
	}

	var updatedPos paper.Position
	if err := json.NewDecoder(protRec.Body).Decode(&updatedPos); err != nil {
		t.Fatalf("decode updated position failed: %v", err)
	}
	if updatedPos.StopLoss == nil || *updatedPos.StopLoss != 2390.0 {
		t.Errorf("StopLoss = %v; want 2390.0", updatedPos.StopLoss)
	}

	// Close position
	closeBody := bytes.NewBufferString(`{"client_request_id":"close_pos_test_01","reason":"manual"}`)
	closeReq := httptest.NewRequest(http.MethodPost, "/api/paper/positions/"+pos.ID+"/close", closeBody)
	closeRec := httptest.NewRecorder()
	mux.ServeHTTP(closeRec, closeReq)

	if closeRec.Code != http.StatusOK {
		t.Fatalf("POST close status = %d; want %d; body = %s", closeRec.Code, http.StatusOK, closeRec.Body.String())
	}

	var closedTrade paper.Trade
	if err := json.NewDecoder(closeRec.Body).Decode(&closedTrade); err != nil {
		t.Fatalf("decode closed trade failed: %v", err)
	}
	if closedTrade.PositionID != pos.ID {
		t.Errorf("trade.PositionID = %q; want %q", closedTrade.PositionID, pos.ID)
	}
}

func TestPaperEmergencyClose(t *testing.T) {
	_, mux, engine, _, _ := setupPaperTestEnv(t)

	// Open two positions
	_, err := engine.OpenPosition(context.Background(), paper.OpenPositionRequest{
		Symbol:          paper.SymbolXAUUSD,
		Side:            paper.SideBuy,
		VolumeLots:      0.01,
		ClientRequestID: "emg_1",
	})
	if err != nil {
		t.Fatalf("OpenPosition 1 failed: %v", err)
	}
	_, err = engine.OpenPosition(context.Background(), paper.OpenPositionRequest{
		Symbol:          paper.SymbolXAUUSD,
		Side:            paper.SideSell,
		VolumeLots:      0.01,
		ClientRequestID: "emg_2",
	})
	if err != nil {
		t.Fatalf("OpenPosition 2 failed: %v", err)
	}

	// Emergency Close
	emgReq := httptest.NewRequest(http.MethodPost, "/api/paper/emergency-close", nil)
	emgRec := httptest.NewRecorder()
	mux.ServeHTTP(emgRec, emgReq)

	if emgRec.Code != http.StatusOK {
		t.Fatalf("POST emergency-close status = %d; want %d; body = %s", emgRec.Code, http.StatusOK, emgRec.Body.String())
	}

	var emgResp struct {
		ClosedTrades []paper.Trade `json:"closed_trades"`
		Count        int           `json:"count"`
	}
	if err := json.NewDecoder(emgRec.Body).Decode(&emgResp); err != nil {
		t.Fatalf("decode emergency response failed: %v", err)
	}
	if emgResp.Count != 2 {
		t.Errorf("closed count = %d; want 2", emgResp.Count)
	}
}

func TestPaperErrorFormat(t *testing.T) {
	_, mux, _, _, _ := setupPaperTestEnv(t)

	// 1. Invalid JSON body on prepare -> 400 VALIDATION_ERROR
	badReq := httptest.NewRequest(http.MethodPost, "/api/paper/trade/prepare", bytes.NewBufferString("{invalid-json"))
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badReq)

	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", badRec.Code, http.StatusBadRequest)
	}

	var errResp PaperErrorResponse
	if err := json.NewDecoder(badRec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error envelope failed: %v", err)
	}
	if errResp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("error.code = %q; want VALIDATION_ERROR", errResp.Error.Code)
	}

	// 2. Not found position close -> 404 NOT_FOUND
	notFoundReq := httptest.NewRequest(http.MethodPost, "/api/paper/positions/non-existent/close", bytes.NewBufferString(`{"client_request_id":"close_missing","reason":"manual"}`))
	notFoundRec := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRec, notFoundReq)

	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want %d", notFoundRec.Code, http.StatusNotFound)
	}
	var nfResp PaperErrorResponse
	if err := json.NewDecoder(notFoundRec.Body).Decode(&nfResp); err != nil {
		t.Fatalf("decode error envelope failed: %v", err)
	}
	if nfResp.Error.Code != "NOT_FOUND" {
		t.Errorf("error.code = %q; want NOT_FOUND", nfResp.Error.Code)
	}
}

func TestPaperTradeConfirmRequiresIdempotencyKeyAndRejectsDuplicate(t *testing.T) {
	_, mux, engine, _, _ := setupPaperTestEnv(t)

	prepare := func() paper.Proposal {
		req := httptest.NewRequest(http.MethodPost, "/api/paper/trade/prepare", bytes.NewBufferString(
			`{"symbol":"XAUUSD","side":"BUY","volume_lots":0.01}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("prepare status = %d; body = %s", rec.Code, rec.Body.String())
		}
		var proposal paper.Proposal
		if err := json.NewDecoder(rec.Body).Decode(&proposal); err != nil {
			t.Fatal(err)
		}
		return proposal
	}

	missingKey := prepare()
	req := httptest.NewRequest(http.MethodPost, "/api/paper/trade/confirm", bytes.NewBufferString(
		`{"proposal_id":"`+missingKey.ID+`"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertPaperError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR", "client_request_id")

	if _, err := engine.OpenPosition(context.Background(), paper.OpenPositionRequest{
		ClientRequestID: "already-used", Symbol: paper.SymbolXAUUSD, Side: paper.SideBuy, VolumeLots: .01,
	}); err != nil {
		t.Fatal(err)
	}
	duplicate := prepare()
	req = httptest.NewRequest(http.MethodPost, "/api/paper/trade/confirm", bytes.NewBufferString(
		`{"proposal_id":"`+duplicate.ID+`","client_request_id":"already-used"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assertPaperError(t, rec, http.StatusConflict, "CONFLICT", "client_request_id")
}

func TestPaperProposalStoreConsumeIsAtomicAndExpiresAtBoundary(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	store := newProposalStore()
	store.now = func() time.Time { return now }
	proposal := paper.Proposal{ID: "prop_once", ExpiresAt: now.Add(time.Second)}
	store.Put(proposal)
	if _, ok := store.Consume(proposal.ID); !ok {
		t.Fatal("first Consume() did not return proposal")
	}
	if _, ok := store.Consume(proposal.ID); ok {
		t.Fatal("second Consume() returned an already-consumed proposal")
	}

	proposal.ID = "prop_expired"
	store.Put(proposal)
	now = proposal.ExpiresAt
	if _, ok := store.Consume(proposal.ID); ok {
		t.Fatal("Consume() returned proposal at expiration boundary")
	}
}

func TestPaperRequestValidationContracts(t *testing.T) {
	_, mux, _, _, _ := setupPaperTestEnv(t)
	tests := []struct {
		name, method, path, body, field string
	}{
		{"invalid trade limit", http.MethodGet, "/api/paper/trades?limit=zero", "", "limit"},
		{"excessive trade limit", http.MethodGet, "/api/paper/trades?limit=501", "", "limit"},
		{"malformed reset", http.MethodPost, "/api/paper/reset", "{", ""},
		{"unknown prepare field", http.MethodPost, "/api/paper/trade/prepare", `{"symbol":"XAUUSD","side":"buy","volume_lots":.01,"live":true}`, ""},
		{"malformed close", http.MethodPost, "/api/paper/positions/id/close", "{", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assertPaperError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR", tt.field)
		})
	}
}

func TestPaperStaleQuoteContract(t *testing.T) {
	_, mux, _, _, quotes := setupPaperTestEnv(t)
	quotes.SetQuote(paper.Quote{
		Symbol: paper.SymbolXAUUSD,
		Bid:    2400,
		Ask:    2400.5,
		AsOf:   time.Now().UTC().Add(-time.Minute),
	})
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/paper/account", ""},
		{http.MethodGet, "/api/paper/positions", ""},
		{http.MethodPost, "/api/paper/trade/prepare", `{"symbol":"XAUUSD","side":"buy","volume_lots":0.01}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertPaperError(t, rec, http.StatusServiceUnavailable, "QUOTE_STALE", "")
	}
}

func assertPaperError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body.String())
	}
	var response PaperErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != code || response.Error.Field != field {
		t.Fatalf("error = %#v, want code=%q field=%q", response.Error, code, field)
	}
}
