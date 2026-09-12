package paper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const defaultQuoteMaxAge = 30 * time.Second

// Engine owns all calculations and state mutations for paper trading.
type Engine struct {
	store       *Store
	quotes      QuoteSource
	now         func() time.Time
	instrument  Instrument
	riskLimits  RiskLimits
	maxQuoteAge time.Duration
}

func NewEngine(store *Store, quotes QuoteSource) (*Engine, error) {
	if store == nil || quotes == nil {
		return nil, errors.New("paper: store and quote source are required")
	}
	limits := DefaultRiskLimits()
	if err := validateRiskLimits(limits); err != nil {
		return nil, err
	}
	return &Engine{
		store: store, quotes: quotes, now: time.Now, maxQuoteAge: defaultQuoteMaxAge, riskLimits: limits,
		instrument: Instrument{Symbol: SymbolXAUUSD, BaseCurrency: "XAU", QuoteCurrency: "USD", ContractSize: 100, MinimumLot: .01, LotStep: .01, MaximumLot: 1, PriceDigits: 2},
	}, nil
}

// SetClock installs a deterministic clock. It is intended for composition and tests.
func (e *Engine) SetClock(now func() time.Time) {
	if now != nil {
		e.now = now
	}
}
func (e *Engine) SetMaxQuoteAge(age time.Duration) { e.maxQuoteAge = age }
func (e *Engine) SetRiskLimits(limits RiskLimits) error {
	if err := validateRiskLimits(limits); err != nil {
		return err
	}
	e.riskLimits = limits
	return nil
}

func (e *Engine) currentQuote(ctx context.Context) (Quote, error) {
	quote, err := e.quotes.Quote(ctx, SymbolXAUUSD)
	if err != nil {
		return Quote{}, fmt.Errorf("paper: get quote: %w", err)
	}
	if err := ValidateQuoteFreshness(quote, e.now(), e.maxQuoteAge); err != nil {
		return Quote{}, err
	}
	return quote, nil
}

// CurrentQuote returns the current validated quote for the paper engine.
func (e *Engine) CurrentQuote(ctx context.Context) (Quote, error) {
	return e.currentQuote(ctx)
}

// PrepareProposal validates a potential trade and creates a short-lived proposal.
func (e *Engine) PrepareProposal(ctx context.Context, request OpenPositionRequest, ttl time.Duration) (*Proposal, error) {
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	request, err = ValidateOpenPositionRequest(request, quote)
	if err != nil {
		return nil, err
	}
	if err := validateLot(e.instrument, request.VolumeLots); err != nil {
		return nil, err
	}
	price, err := quote.OpenPrice(request.Side)
	if err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	now := e.now().UTC()
	return &Proposal{
		ID:        "prop_" + uuid.NewString(),
		Request:   request,
		Quote:     quote,
		OpenPrice: price,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}, nil
}


// OpenPosition validates risk and atomically creates one filled order and position.
func (e *Engine) OpenPosition(ctx context.Context, request OpenPositionRequest) (*Position, error) {
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	request, err = ValidateOpenPositionRequest(request, quote)
	if err != nil {
		return nil, err
	}
	if err := validateLot(e.instrument, request.VolumeLots); err != nil {
		return nil, err
	}
	price, _ := quote.OpenPrice(request.Side)
	now := e.now().UTC()
	var result *Position
	err = e.store.WithTx(ctx, func(tx *Tx) error {
		existing, lookupErr := tx.GetOrderByClientRequestID(ctx, request.ClientRequestID)
		if lookupErr == nil {
			position, err := tx.GetPosition(ctx, existing.PositionID)
			if err != nil {
				return err
			}
			result = position
			return nil
		}
		if !errors.Is(lookupErr, ErrNotFound) {
			return lookupErr
		}
		account, err := tx.GetAccount(ctx, DefaultAccountID)
		if err != nil {
			return err
		}
		positions, err := tx.ListOpenPositions(ctx, account.ID)
		if err != nil {
			return err
		}
		if len(positions) >= e.riskLimits.MaxOpenPositions {
			return invalid("positions", "open position limit reached")
		}
		if request.VolumeLots > e.riskLimits.MaxVolumeLots {
			return invalid("volume_lots", "paper risk limit exceeded")
		}
		dailyPnL, err := tx.DailyRealizedPnL(ctx, account.ID, startOfUTCDay(now))
		if err != nil {
			return err
		}
		if dailyPnL <= -e.riskLimits.MaxDailyLossUSD {
			return invalid("daily_pnl", "daily loss limit reached")
		}
		notional := price * request.VolumeLots * e.instrument.ContractSize
		for _, position := range positions {
			notional += position.CurrentPrice * position.VolumeLots * position.ContractSize
		}
		if notional > account.Balance {
			return invalid("balance", "insufficient cash for unleveraged exposure")
		}
		position := Position{ID: uuid.NewString(), Symbol: request.Symbol, Side: request.Side, VolumeLots: request.VolumeLots,
			ContractSize: e.instrument.ContractSize, EntryPrice: price, CurrentPrice: price, StopLoss: request.StopLoss,
			TakeProfit: request.TakeProfit, Status: PositionStatusOpen, OpenedAt: now, UpdatedAt: now}
		order := Order{ID: uuid.NewString(), ClientRequestID: request.ClientRequestID, PositionID: position.ID,
			Symbol: request.Symbol, Side: request.Side, Type: OrderTypeMarket, VolumeLots: request.VolumeLots,
			FillPrice: price, StopLoss: request.StopLoss, TakeProfit: request.TakeProfit, Status: OrderStatusFilled, CreatedAt: now, FilledAt: &now}
		if err := tx.SavePosition(ctx, account.ID, position); err != nil {
			return err
		}
		if err := tx.SaveOrder(ctx, account.ID, order); err != nil {
			return err
		}
		if err := tx.SaveAuditEvent(ctx, newAudit(account.ID, "position.opened", position.ID, now)); err != nil {
			return err
		}
		result = &position
		return nil
	})
	return result, err
}

// Snapshot values open positions at executable bid/ask prices.
func (e *Engine) Snapshot(ctx context.Context) (*AccountSnapshot, error) {
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	account, err := e.store.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		return nil, err
	}
	positions, err := e.store.ListOpenPositions(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	var unrealized float64
	for i := range positions {
		price, _ := quote.ClosePrice(positions[i].Side)
		positions[i].CurrentPrice = price
		positions[i].UnrealizedPnL = positionPnL(positions[i], price)
		unrealized += positions[i].UnrealizedPnL
	}
	trades, err := e.store.ListRecentTrades(ctx, account.ID, 20)
	if err != nil {
		return nil, err
	}
	daily, err := e.store.DailyRealizedPnL(ctx, account.ID, startOfUTCDay(e.now()))
	if err != nil {
		return nil, err
	}
	return &AccountSnapshot{Mode: ModePaper, AsOf: quote.AsOf, Account: *account, Equity: account.Balance + unrealized,
		DailyPnL: daily, UnrealizedPnL: unrealized, OpenPositions: positions, RecentTrades: trades}, nil
}

// ClosePosition closes a full position once and records its realized result.
func (e *Engine) ClosePosition(ctx context.Context, positionID, clientRequestID string, reason CloseReason) (*Trade, error) {
	if clientRequestID == "" {
		return nil, invalid("client_request_id", "is required")
	}
	if reason != CloseReasonManual && reason != CloseReasonStopLoss && reason != CloseReasonTakeProfit && reason != CloseReasonEmergency {
		return nil, invalid("close_reason", "is not supported")
	}
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	now := e.now().UTC()
	var result *Trade
	err = e.store.WithTx(ctx, func(tx *Tx) error {
		if existing, lookupErr := tx.GetOrderByClientRequestID(ctx, clientRequestID); lookupErr == nil {
			trade, err := getTradeByCloseOrder(ctx, tx.runner, existing.ID)
			if err != nil {
				return err
			}
			result = trade
			return nil
		} else if !errors.Is(lookupErr, ErrNotFound) {
			return lookupErr
		}
		position, err := tx.GetPosition(ctx, positionID)
		if err != nil {
			return err
		}
		if position.Status != PositionStatusOpen {
			return invalid("position", "is already closed")
		}
		price, err := quote.ClosePrice(position.Side)
		if err != nil {
			return err
		}
		pnl := positionPnL(*position, price)
		account, err := tx.GetAccount(ctx, DefaultAccountID)
		if err != nil {
			return err
		}
		account.Balance += pnl
		account.UpdatedAt = now
		position.Status = PositionStatusClosed
		position.CurrentPrice = price
		position.UnrealizedPnL = 0
		position.UpdatedAt = now
		closeSide := SideSell
		if position.Side == SideSell {
			closeSide = SideBuy
		}
		order := Order{ID: uuid.NewString(), ClientRequestID: clientRequestID, PositionID: position.ID, Symbol: position.Symbol,
			Side: closeSide, Type: OrderTypeMarket, VolumeLots: position.VolumeLots, FillPrice: price, Status: OrderStatusFilled, CreatedAt: now, FilledAt: &now}
		openOrder, err := findOpenOrder(ctx, tx.runner, position.ID)
		if err != nil {
			return err
		}
		trade := Trade{ID: uuid.NewString(), PositionID: position.ID, OpenOrderID: openOrder.ID, CloseOrderID: order.ID,
			Symbol: position.Symbol, Side: position.Side, VolumeLots: position.VolumeLots, ContractSize: position.ContractSize,
			EntryPrice: position.EntryPrice, ExitPrice: price, GrossPnL: pnl, RealizedPnL: pnl, CloseReason: reason, OpenedAt: position.OpenedAt, ClosedAt: now}
		if err := tx.SaveAccount(ctx, *account); err != nil {
			return err
		}
		if err := tx.SavePosition(ctx, account.ID, *position); err != nil {
			return err
		}
		if err := tx.SaveOrder(ctx, account.ID, order); err != nil {
			return err
		}
		if err := tx.SaveTrade(ctx, account.ID, trade); err != nil {
			return err
		}
		if err := tx.SaveAuditEvent(ctx, newAudit(account.ID, "position.closed."+string(reason), position.ID, now)); err != nil {
			return err
		}
		result = &trade
		return nil
	})
	return result, err
}

func (e *Engine) UpdateProtection(ctx context.Context, positionID string, stopLoss, takeProfit *float64) (*Position, error) {
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	now := e.now().UTC()
	var result *Position
	err = e.store.WithTx(ctx, func(tx *Tx) error {
		position, err := tx.GetPosition(ctx, positionID)
		if err != nil {
			return err
		}
		if position.Status != PositionStatusOpen {
			return invalid("position", "is not open")
		}
		price, _ := quote.ClosePrice(position.Side)
		if err := ValidateProtection(position.Side, price, stopLoss, takeProfit); err != nil {
			return err
		}
		position.StopLoss, position.TakeProfit, position.UpdatedAt = stopLoss, takeProfit, now
		if err := tx.SavePosition(ctx, DefaultAccountID, *position); err != nil {
			return err
		}
		if err := tx.SaveAuditEvent(ctx, newAudit(DefaultAccountID, "position.protection_updated", position.ID, now)); err != nil {
			return err
		}
		result = position
		return nil
	})
	return result, err
}

// EvaluateProtections deterministically closes positions whose SL/TP was crossed.
func (e *Engine) EvaluateProtections(ctx context.Context) ([]Trade, error) {
	quote, err := e.currentQuote(ctx)
	if err != nil {
		return nil, err
	}
	positions, err := e.store.ListOpenPositions(ctx, DefaultAccountID)
	if err != nil {
		return nil, err
	}
	var trades []Trade
	for _, p := range positions {
		price, _ := quote.ClosePrice(p.Side)
		reason := CloseReason("")
		if p.Side == SideBuy {
			if p.StopLoss != nil && price <= *p.StopLoss {
				reason = CloseReasonStopLoss
			}
			if p.TakeProfit != nil && price >= *p.TakeProfit {
				reason = CloseReasonTakeProfit
			}
		} else {
			if p.StopLoss != nil && price >= *p.StopLoss {
				reason = CloseReasonStopLoss
			}
			if p.TakeProfit != nil && price <= *p.TakeProfit {
				reason = CloseReasonTakeProfit
			}
		}
		if reason != "" {
			trade, err := e.ClosePosition(ctx, p.ID, "protection-"+p.ID+"-"+string(reason), reason)
			if err != nil {
				return trades, err
			}
			trades = append(trades, *trade)
		}
	}
	return trades, nil
}

func (e *Engine) EmergencyClose(ctx context.Context) ([]Trade, error) {
	positions, err := e.store.ListOpenPositions(ctx, DefaultAccountID)
	if err != nil {
		return nil, err
	}
	var trades []Trade
	for _, p := range positions {
		trade, err := e.ClosePosition(ctx, p.ID, "emergency-"+p.ID, CloseReasonEmergency)
		if err != nil {
			return trades, err
		}
		trades = append(trades, *trade)
	}
	return trades, nil
}

func positionPnL(p Position, price float64) float64 {
	multiplier := 1.0
	if p.Side == SideSell {
		multiplier = -1
	}
	return (price - p.EntryPrice) * p.VolumeLots * p.ContractSize * multiplier
}
func startOfUTCDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func newAudit(account, action, entity string, now time.Time) AuditEvent {
	return AuditEvent{ID: uuid.NewString(), AccountID: account, Action: action, EntityID: entity, CreatedAt: now}
}

func findOpenOrder(ctx context.Context, runner sqlRunner, positionID string) (*Order, error) {
	order, err := scanOrder(runner.QueryRowContext(ctx, orderSelect+" WHERE position_id = ? AND side = (SELECT side FROM paper_positions WHERE id = ?) ORDER BY created_at ASC LIMIT 1", positionID, positionID).Scan)
	if err != nil {
		return nil, recordError("open order", positionID, err)
	}
	return order, nil
}
func getTradeByCloseOrder(ctx context.Context, runner sqlRunner, orderID string) (*Trade, error) {
	trade, err := scanTrade(runner.QueryRowContext(ctx, tradeSelect+" WHERE close_order_id = ?", orderID).Scan)
	if err != nil {
		return nil, recordError("trade", orderID, err)
	}
	return trade, nil
}
