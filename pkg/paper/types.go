// Package paper defines the deterministic domain model for local paper trading.
package paper

import "time"

// ModePaper is the only execution mode supported by this package.
const ModePaper = "paper"

// Canonical symbols supported by the MVP.
const (
	SymbolXAUUSD = "XAUUSD"
)

// Side describes whether a position benefits from a rising or falling price.
type Side string

const (
	// SideBuy opens a long position.
	SideBuy Side = "buy"
	// SideSell opens a short position.
	SideSell Side = "sell"
)

// Valid reports whether the side is supported.
func (s Side) Valid() bool {
	return s == SideBuy || s == SideSell
}

// OrderType identifies how an order is filled.
type OrderType string

const (
	// OrderTypeMarket fills at the current paper quote.
	OrderTypeMarket OrderType = "market"
)

// OrderStatus describes the lifecycle state of a paper order.
type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "pending"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusRejected OrderStatus = "rejected"
)

// PositionStatus describes whether a position can still change in value.
type PositionStatus string

const (
	PositionStatusOpen   PositionStatus = "open"
	PositionStatusClosed PositionStatus = "closed"
)

// CloseReason records why a paper position was closed.
type CloseReason string

const (
	CloseReasonManual     CloseReason = "manual"
	CloseReasonStopLoss   CloseReason = "stop_loss"
	CloseReasonTakeProfit CloseReason = "take_profit"
	CloseReasonEmergency  CloseReason = "emergency"
)

// Account stores durable cash state. Equity and P/L are calculated in a snapshot.
type Account struct {
	ID              string    `json:"id"`
	Currency        string    `json:"currency"`
	StartingBalance float64   `json:"starting_balance"`
	Balance         float64   `json:"balance"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Instrument describes the contract and lot rules captured for a paper symbol.
type Instrument struct {
	Symbol        string  `json:"symbol"`
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	ContractSize  float64 `json:"contract_size"`
	MinimumLot    float64 `json:"minimum_lot"`
	LotStep       float64 `json:"lot_step"`
	MaximumLot    float64 `json:"maximum_lot"`
	PriceDigits   int     `json:"price_digits"`
}

// AuditEvent records a safety-relevant paper account mutation.
type AuditEvent struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Action    string    `json:"action"`
	EntityID  string    `json:"entity_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Quote is a point-in-time bid and ask pair.
type Quote struct {
	Symbol string    `json:"symbol"`
	Bid    float64   `json:"bid"`
	Ask    float64   `json:"ask"`
	AsOf   time.Time `json:"as_of"`
}

// OpenPositionRequest is the normalized input for a simulated market order.
type OpenPositionRequest struct {
	ClientRequestID string   `json:"client_request_id"`
	Symbol          string   `json:"symbol"`
	Side            Side     `json:"side"`
	VolumeLots      float64  `json:"volume_lots"`
	StopLoss        *float64 `json:"stop_loss,omitempty"`
	TakeProfit      *float64 `json:"take_profit,omitempty"`
}

// Position stores a simulated exposure. Monetary P/L values use account currency.
type Position struct {
	ID            string         `json:"id"`
	Symbol        string         `json:"symbol"`
	Side          Side           `json:"side"`
	VolumeLots    float64        `json:"volume_lots"`
	ContractSize  float64        `json:"contract_size"`
	EntryPrice    float64        `json:"entry_price"`
	CurrentPrice  float64        `json:"current_price"`
	StopLoss      *float64       `json:"stop_loss,omitempty"`
	TakeProfit    *float64       `json:"take_profit,omitempty"`
	UnrealizedPnL float64        `json:"unrealized_pnl"`
	Status        PositionStatus `json:"status"`
	OpenedAt      time.Time      `json:"opened_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Order records one attempt to change paper-account exposure.
type Order struct {
	ID              string      `json:"id"`
	ClientRequestID string      `json:"client_request_id"`
	PositionID      string      `json:"position_id,omitempty"`
	Symbol          string      `json:"symbol"`
	Side            Side        `json:"side"`
	Type            OrderType   `json:"type"`
	VolumeLots      float64     `json:"volume_lots"`
	FillPrice       float64     `json:"fill_price"`
	StopLoss        *float64    `json:"stop_loss,omitempty"`
	TakeProfit      *float64    `json:"take_profit,omitempty"`
	Status          OrderStatus `json:"status"`
	RejectionReason string      `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	FilledAt        *time.Time  `json:"filled_at,omitempty"`
}

// Trade is the immutable record produced when a paper position is closed.
type Trade struct {
	ID           string      `json:"id"`
	PositionID   string      `json:"position_id"`
	OpenOrderID  string      `json:"open_order_id"`
	CloseOrderID string      `json:"close_order_id"`
	Symbol       string      `json:"symbol"`
	Side         Side        `json:"side"`
	VolumeLots   float64     `json:"volume_lots"`
	ContractSize float64     `json:"contract_size"`
	EntryPrice   float64     `json:"entry_price"`
	ExitPrice    float64     `json:"exit_price"`
	GrossPnL     float64     `json:"gross_pnl"`
	Fees         float64     `json:"fees"`
	RealizedPnL  float64     `json:"realized_pnl"`
	CloseReason  CloseReason `json:"close_reason"`
	OpenedAt     time.Time   `json:"opened_at"`
	ClosedAt     time.Time   `json:"closed_at"`
}

// AccountSnapshot is the read model used by the dashboard and chat tools.
type AccountSnapshot struct {
	Mode          string     `json:"mode"`
	AsOf          time.Time  `json:"as_of"`
	Account       Account    `json:"account"`
	Equity        float64    `json:"equity"`
	DailyPnL      float64    `json:"daily_pnl"`
	UnrealizedPnL float64    `json:"unrealized_pnl"`
	OpenPositions []Position `json:"open_positions"`
	RecentTrades  []Trade    `json:"recent_trades"`
}

// Proposal binds a validated request to the quote that was shown for confirmation.
type Proposal struct {
	ID        string              `json:"id"`
	Request   OpenPositionRequest `json:"request"`
	Quote     Quote               `json:"quote"`
	OpenPrice float64             `json:"open_price"`
	CreatedAt time.Time           `json:"created_at"`
	ExpiresAt time.Time           `json:"expires_at"`
}
