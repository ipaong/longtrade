package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/paper"
)

const (
	defaultProposalTTL = 30 * time.Second
)

// defaultQuoteSource provides realistic, fresh quotes for local paper trading
// when no live external quote feed is connected.
type defaultQuoteSource struct {
	mu  sync.RWMutex
	bid float64
	ask float64
}

func newDefaultQuoteSource() *defaultQuoteSource {
	return &defaultQuoteSource{
		bid: 2400.00,
		ask: 2400.50,
	}
}

func (s *defaultQuoteSource) Quote(_ context.Context, symbol string) (paper.Quote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paper.Quote{
		Symbol: paper.NormalizeSymbol(symbol),
		Bid:    s.bid,
		Ask:    s.ask,
		AsOf:   time.Now().UTC(),
	}, nil
}

// proposalStore maintains in-memory short-lived trade proposals for explicit confirmation.
type proposalStore struct {
	mu        sync.RWMutex
	proposals map[string]paper.Proposal
	now       func() time.Time
}

func newProposalStore() *proposalStore {
	return &proposalStore{
		proposals: make(map[string]paper.Proposal),
		now:       time.Now,
	}
}

func (s *proposalStore) Put(p paper.Proposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	for k, v := range s.proposals {
		if !now.Before(v.ExpiresAt) {
			delete(s.proposals, k)
		}
	}
	s.proposals[p.ID] = p
}

// Consume atomically retrieves and removes a proposal. A proposal can therefore
// result in at most one execution even when confirmations arrive concurrently.
func (s *proposalStore) Consume(id string) (paper.Proposal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.proposals[id]
	if !ok {
		return paper.Proposal{}, false
	}
	delete(s.proposals, id)
	if !s.now().UTC().Before(p.ExpiresAt) {
		return paper.Proposal{}, false
	}
	return p, true
}

// Structured error response envelope.
type PaperErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type PaperErrorResponse struct {
	Error PaperErrorDetail `json:"error"`
}

func writePaperError(w http.ResponseWriter, status int, code, message, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(PaperErrorResponse{
		Error: PaperErrorDetail{
			Code:    code,
			Message: message,
			Field:   field,
		},
	})
}

func decodePaperJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

// SetPaperEngine installs a pre-configured engine and store for testing.
func (h *Handler) SetPaperEngine(engine *paper.Engine, store *paper.Store) {
	h.paperMu.Lock()
	defer h.paperMu.Unlock()
	h.paperEngine = engine
	h.paperStore = store
}

// SetPaperQuotes sets the quote source for the paper engine.
func (h *Handler) SetPaperQuotes(quotes paper.QuoteSource) {
	h.paperMu.Lock()
	defer h.paperMu.Unlock()
	h.paperQuotes = quotes
	h.paperEngine = nil // force reinitialization with new quote source
}

func (h *Handler) getPaperComponents() (*paper.Engine, *paper.Store, *proposalStore, error) {
	h.paperMu.Lock()
	defer h.paperMu.Unlock()

	if h.paperProposals == nil {
		h.paperProposals = newProposalStore()
	}

	if h.paperEngine != nil && h.paperStore != nil {
		return h.paperEngine, h.paperStore, h.paperProposals, nil
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	ws := cfg.WorkspacePath()
	store, err := paper.NewStore(ws)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open paper store: %w", err)
	}

	quotes := h.paperQuotes
	if quotes == nil {
		quotes = newDefaultQuoteSource()
		h.paperQuotes = quotes
	}

	engine, err := paper.NewEngine(store, quotes)
	if err != nil {
		_ = store.Close()
		return nil, nil, nil, fmt.Errorf("failed to initialize paper engine: %w", err)
	}

	h.paperStore = store
	h.paperEngine = engine
	return h.paperEngine, h.paperStore, h.paperProposals, nil
}

func (h *Handler) registerPaperRoutes(mux *http.ServeMux) {
	// Account & Dashboard (LT-201)
	mux.HandleFunc("GET /api/paper/account", h.handlePaperGetAccount)
	mux.HandleFunc("POST /api/paper/account/reset", h.handlePaperResetAccount)
	mux.HandleFunc("POST /api/paper/reset", h.handlePaperResetAccount)
	mux.HandleFunc("GET /api/paper/positions", h.handlePaperGetPositions)
	mux.HandleFunc("GET /api/paper/trades", h.handlePaperGetTrades)

	// Simulated Trade Workflow (LT-202)
	mux.HandleFunc("POST /api/paper/trade/prepare", h.handlePaperPrepareTrade)
	mux.HandleFunc("POST /api/paper/trade/confirm", h.handlePaperConfirmTrade)

	// Position Management & Emergency Close (LT-203)
	mux.HandleFunc("POST /api/paper/positions/{id}/close", h.handlePaperClosePosition)
	mux.HandleFunc("POST /api/paper/positions/{id}/protection", h.handlePaperUpdateProtection)
	mux.HandleFunc("POST /api/paper/emergency-close", h.handlePaperEmergencyClose)
}

// handlePaperGetAccount returns the current paper account snapshot.
func (h *Handler) handlePaperGetAccount(w http.ResponseWriter, r *http.Request) {
	engine, _, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	snapshot, err := engine.Snapshot(r.Context())
	if err != nil {
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

// handlePaperResetAccount restores the paper account to default starting balance.
func (h *Handler) handlePaperResetAccount(w http.ResponseWriter, r *http.Request) {
	engine, store, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	var req struct{}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodePaperJSON(r, &req); err != nil {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json request body", "")
			return
		}
	}

	now := time.Now().UTC()
	_, err = store.ResetDefaultAccount(r.Context(), now)
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	snapshot, err := engine.Snapshot(r.Context())
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

// handlePaperGetPositions returns all active open positions.
func (h *Handler) handlePaperGetPositions(w http.ResponseWriter, r *http.Request) {
	engine, _, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	snapshot, err := engine.Snapshot(r.Context())
	if err != nil {
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	positions := snapshot.OpenPositions
	if positions == nil {
		positions = []paper.Position{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(positions)
}

// handlePaperGetTrades returns historical closed trades.
func (h *Handler) handlePaperGetTrades(w http.ResponseWriter, r *http.Request) {
	_, store, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		l, parseErr := strconv.Atoi(lStr)
		if parseErr != nil || l <= 0 || l > 500 {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit must be between 1 and 500", "limit")
			return
		}
		limit = l
	}

	trades, err := store.ListRecentTrades(r.Context(), paper.DefaultAccountID, limit)
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}
	if trades == nil {
		trades = []paper.Trade{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(trades)
}

type prepareTradeRequest struct {
	Symbol     string     `json:"symbol"`
	Side       paper.Side `json:"side"`
	VolumeLots float64    `json:"volume_lots"`
	StopLoss   *float64   `json:"stop_loss,omitempty"`
	TakeProfit *float64   `json:"take_profit,omitempty"`
}

// handlePaperPrepareTrade creates a temporary trade proposal bound to current quote.
func (h *Handler) handlePaperPrepareTrade(w http.ResponseWriter, r *http.Request) {
	engine, _, proposals, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	var req prepareTradeRequest
	if err := decodePaperJSON(r, &req); err != nil {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json request body", "")
		return
	}

	openReq := paper.OpenPositionRequest{
		Symbol:     req.Symbol,
		Side:       req.Side,
		VolumeLots: req.VolumeLots,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
	}

	proposal, err := engine.PrepareProposal(r.Context(), openReq, defaultProposalTTL)
	if err != nil {
		var validationErr *paper.ValidationError
		if errors.As(err, &validationErr) {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, validationErr.Field)
			return
		}
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	proposals.Put(*proposal)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(proposal)
}

type confirmTradeRequest struct {
	ProposalID      string `json:"proposal_id"`
	ClientRequestID string `json:"client_request_id,omitempty"`
}

// handlePaperConfirmTrade confirms and executes a previously prepared trade proposal.
func (h *Handler) handlePaperConfirmTrade(w http.ResponseWriter, r *http.Request) {
	engine, store, proposals, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	var req confirmTradeRequest
	if err := decodePaperJSON(r, &req); err != nil {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json request body", "")
		return
	}
	if req.ProposalID == "" {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "proposal_id is required", "proposal_id")
		return
	}
	if req.ClientRequestID == "" {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_request_id is required", "client_request_id")
		return
	}

	proposal, ok := proposals.Consume(req.ProposalID)
	if !ok {
		writePaperError(w, http.StatusGone, "EXPIRED", "proposal has expired or does not exist", "proposal_id")
		return
	}

	openReq := proposal.Request
	openReq.ClientRequestID = req.ClientRequestID

	// Serialize the existence check with execution. Engine.OpenPosition remains
	// idempotent for domain callers, while the HTTP contract deliberately reports
	// a reused client_request_id as a conflict.
	h.paperConfirmMu.Lock()
	if _, lookupErr := store.GetOrderByClientRequestID(r.Context(), req.ClientRequestID); lookupErr == nil {
		h.paperConfirmMu.Unlock()
		writePaperError(w, http.StatusConflict, "CONFLICT", "duplicate client request id", "client_request_id")
		return
	} else if !errors.Is(lookupErr, paper.ErrNotFound) {
		h.paperConfirmMu.Unlock()
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", lookupErr.Error(), "")
		return
	}
	position, err := engine.OpenPosition(r.Context(), openReq)
	h.paperConfirmMu.Unlock()
	if err != nil {
		if errors.Is(err, paper.ErrDuplicateClientRequestID) {
			writePaperError(w, http.StatusConflict, "CONFLICT", "duplicate client request id", "client_request_id")
			return
		}
		var validationErr *paper.ValidationError
		if errors.As(err, &validationErr) {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, validationErr.Field)
			return
		}
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"position": position,
	})
}

type closePositionRequest struct {
	ClientRequestID string            `json:"client_request_id,omitempty"`
	Reason          paper.CloseReason `json:"reason,omitempty"`
}

// handlePaperClosePosition closes an open position at current market price.
func (h *Handler) handlePaperClosePosition(w http.ResponseWriter, r *http.Request) {
	engine, _, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	positionID := r.PathValue("id")
	if positionID == "" {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "position id required", "id")
		return
	}

	var req closePositionRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodePaperJSON(r, &req); err != nil {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json request body", "")
			return
		}
	}
	if req.ClientRequestID == "" {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_request_id is required", "client_request_id")
		return
	}
	if req.Reason == "" {
		req.Reason = paper.CloseReasonManual
	}

	trade, err := engine.ClosePosition(r.Context(), positionID, req.ClientRequestID, req.Reason)
	if err != nil {
		if errors.Is(err, paper.ErrNotFound) {
			writePaperError(w, http.StatusNotFound, "NOT_FOUND", "position not found or already closed", "id")
			return
		}
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		var validationErr *paper.ValidationError
		if errors.As(err, &validationErr) {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, validationErr.Field)
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(trade)
}

type updateProtectionRequest struct {
	StopLoss   *float64 `json:"stop_loss"`
	TakeProfit *float64 `json:"take_profit"`
}

// handlePaperUpdateProtection updates Stop Loss and Take Profit levels on an open position.
func (h *Handler) handlePaperUpdateProtection(w http.ResponseWriter, r *http.Request) {
	engine, _, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	positionID := r.PathValue("id")
	if positionID == "" {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "position id required", "id")
		return
	}

	var req updateProtectionRequest
	if err := decodePaperJSON(r, &req); err != nil {
		writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json request body", "")
		return
	}

	pos, err := engine.UpdateProtection(r.Context(), positionID, req.StopLoss, req.TakeProfit)
	if err != nil {
		if errors.Is(err, paper.ErrNotFound) {
			writePaperError(w, http.StatusNotFound, "NOT_FOUND", "position not found", "id")
			return
		}
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		var validationErr *paper.ValidationError
		if errors.As(err, &validationErr) {
			writePaperError(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, validationErr.Field)
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pos)
}

// handlePaperEmergencyClose closes all open positions immediately.
func (h *Handler) handlePaperEmergencyClose(w http.ResponseWriter, r *http.Request) {
	engine, _, _, err := h.getPaperComponents()
	if err != nil {
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}

	trades, err := engine.EmergencyClose(r.Context())
	if err != nil {
		if errors.Is(err, paper.ErrQuoteStale) {
			writePaperError(w, http.StatusServiceUnavailable, "QUOTE_STALE", err.Error(), "")
			return
		}
		writePaperError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), "")
		return
	}
	if trades == nil {
		trades = []paper.Trade{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"closed_trades": trades,
		"count":         len(trades),
	})
}
