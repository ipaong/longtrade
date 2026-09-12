package paper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sqlite "modernc.org/sqlite"
)

const (
	currentSchemaVersion       = 2
	sqliteConstraintUniqueCode = 2067
)

var (
	// ErrNotFound is returned when a requested paper-trading record does not exist.
	ErrNotFound = errors.New("paper: record not found")
	// ErrDuplicateClientRequestID prevents one user action from creating multiple orders.
	ErrDuplicateClientRequestID = errors.New("paper: duplicate client request id")
)

type migration struct {
	version int
	sql     string
}

var storeMigrations = []migration{
	{
		version: 1,
		sql: `
CREATE TABLE IF NOT EXISTS paper_accounts (
    id                TEXT PRIMARY KEY,
    currency          TEXT NOT NULL,
    starting_balance  REAL NOT NULL CHECK (starting_balance >= 0),
    balance           REAL NOT NULL,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS paper_positions (
    id              TEXT PRIMARY KEY,
    account_id      TEXT NOT NULL REFERENCES paper_accounts(id),
    symbol          TEXT NOT NULL,
    side            TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    volume_lots     REAL NOT NULL CHECK (volume_lots > 0),
    contract_size   REAL NOT NULL CHECK (contract_size > 0),
    entry_price     REAL NOT NULL CHECK (entry_price > 0),
    current_price   REAL NOT NULL CHECK (current_price > 0),
    stop_loss       REAL CHECK (stop_loss IS NULL OR stop_loss > 0),
    take_profit     REAL CHECK (take_profit IS NULL OR take_profit > 0),
    unrealized_pnl  REAL NOT NULL DEFAULT 0,
    status          TEXT NOT NULL CHECK (status IN ('open', 'closed')),
    opened_at       TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS paper_orders (
    id                TEXT PRIMARY KEY,
    account_id        TEXT NOT NULL REFERENCES paper_accounts(id),
    client_request_id TEXT NOT NULL UNIQUE,
    position_id       TEXT REFERENCES paper_positions(id) ON DELETE SET NULL,
    symbol            TEXT NOT NULL,
    side              TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    order_type        TEXT NOT NULL CHECK (order_type = 'market'),
    volume_lots       REAL NOT NULL CHECK (volume_lots > 0),
    fill_price        REAL NOT NULL DEFAULT 0 CHECK (fill_price >= 0),
    stop_loss         REAL CHECK (stop_loss IS NULL OR stop_loss > 0),
    take_profit       REAL CHECK (take_profit IS NULL OR take_profit > 0),
    status            TEXT NOT NULL CHECK (status IN ('pending', 'filled', 'rejected')),
    rejection_reason  TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    filled_at         TEXT
);

CREATE TABLE IF NOT EXISTS paper_trades (
    id             TEXT PRIMARY KEY,
    account_id     TEXT NOT NULL REFERENCES paper_accounts(id),
    position_id    TEXT NOT NULL REFERENCES paper_positions(id),
    open_order_id  TEXT NOT NULL REFERENCES paper_orders(id),
    close_order_id TEXT NOT NULL REFERENCES paper_orders(id),
    symbol         TEXT NOT NULL,
    side           TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    volume_lots    REAL NOT NULL CHECK (volume_lots > 0),
    contract_size  REAL NOT NULL CHECK (contract_size > 0),
    entry_price    REAL NOT NULL CHECK (entry_price > 0),
    exit_price     REAL NOT NULL CHECK (exit_price > 0),
    gross_pnl      REAL NOT NULL,
    fees           REAL NOT NULL DEFAULT 0 CHECK (fees >= 0),
    realized_pnl   REAL NOT NULL,
    close_reason   TEXT NOT NULL CHECK (close_reason IN ('manual', 'stop_loss', 'take_profit', 'emergency')),
    opened_at      TEXT NOT NULL,
    closed_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_paper_positions_account_status
    ON paper_positions(account_id, status);
CREATE INDEX IF NOT EXISTS idx_paper_orders_account_created
    ON paper_orders(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_paper_trades_account_closed
    ON paper_trades(account_id, closed_at DESC);
`,
	},
	{
		version: 2,
		sql: `
CREATE TABLE IF NOT EXISTS paper_audit_events (
    id         TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES paper_accounts(id),
    action     TEXT NOT NULL,
    entity_id  TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_paper_audit_account_created
    ON paper_audit_events(account_id, created_at DESC);
`,
	},
}

// Store persists the local paper-trading ledger in SQLite.
type Store struct {
	db   *sql.DB
	path string
}

// Tx exposes store operations that participate in one SQLite transaction.
type Tx struct {
	runner sqlRunner
}

type sqlRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DatabasePath returns the paper database location for a workspace.
func DatabasePath(workspacePath string) string {
	return filepath.Join(workspacePath, "memory", "paper", "paper.db")
}

// NewStore opens or creates the paper database below workspacePath.
func NewStore(workspacePath string) (*Store, error) {
	if workspacePath == "" {
		return nil, errors.New("paper: workspace path is required")
	}

	dbPath := DatabasePath(workspacePath)
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("paper: create database directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("paper: secure database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("paper: open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	closeWithError := func(err error) (*Store, error) {
		_ = db.Close()
		return nil, err
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-2000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return closeWithError(fmt.Errorf("paper: %s: %w", pragma, err))
		}
	}
	if err := migrateStore(context.Background(), db); err != nil {
		return closeWithError(err)
	}
	store := &Store{db: db, path: dbPath}
	if _, err := store.EnsureDefaultAccount(context.Background(), time.Now()); err != nil {
		return closeWithError(err)
	}
	if err := os.Chmod(dbPath, 0o600); err != nil {
		return closeWithError(fmt.Errorf("paper: secure database file: %w", err))
	}

	return store, nil
}

func migrateStore(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("paper: begin migration: %w", err)
	}
	defer tx.Rollback()

	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("paper: read schema version: %w", err)
	}
	if version > currentSchemaVersion {
		return fmt.Errorf("paper: database schema version %d is newer than supported version %d", version, currentSchemaVersion)
	}

	for _, item := range storeMigrations {
		if item.version <= version {
			continue
		}
		if _, err := tx.ExecContext(ctx, item.sql); err != nil {
			return fmt.Errorf("paper: apply schema version %d: %w", item.version, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", item.version)); err != nil {
			return fmt.Errorf("paper: record schema version %d: %w", item.version, err)
		}
		version = item.version
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("paper: commit migration: %w", err)
	}
	return nil
}

// Path returns the database file used by the store.
func (s *Store) Path() string {
	return s.path
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// WithTx runs fn in a transaction and rolls it back unless fn and Commit succeed.
func (s *Store) WithTx(ctx context.Context, fn func(*Tx) error) error {
	if fn == nil {
		return errors.New("paper: transaction function is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("paper: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(&Tx{runner: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("paper: commit transaction: %w", err)
	}
	return nil
}

// SaveAccount inserts or replaces the mutable state of an account.
func (s *Store) SaveAccount(ctx context.Context, account Account) error {
	return saveAccount(ctx, s.db, account)
}

// SaveAccount inserts or replaces the mutable state of an account in this transaction.
func (tx *Tx) SaveAccount(ctx context.Context, account Account) error {
	return saveAccount(ctx, tx.runner, account)
}

func saveAccount(ctx context.Context, runner sqlRunner, account Account) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO paper_accounts
		    (id, currency, starting_balance, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		    currency=excluded.currency,
		    starting_balance=excluded.starting_balance,
		    balance=excluded.balance,
		    created_at=excluded.created_at,
		    updated_at=excluded.updated_at`,
		account.ID, account.Currency, account.StartingBalance, account.Balance,
		formatTime(account.CreatedAt), formatTime(account.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("paper: save account %q: %w", account.ID, err)
	}
	return nil
}

// GetAccount returns one account by ID.
func (s *Store) GetAccount(ctx context.Context, id string) (*Account, error) {
	return getAccount(ctx, s.db, id)
}

// GetAccount returns one account by ID from this transaction.
func (tx *Tx) GetAccount(ctx context.Context, id string) (*Account, error) {
	return getAccount(ctx, tx.runner, id)
}

func getAccount(ctx context.Context, runner sqlRunner, id string) (*Account, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT id, currency, starting_balance, balance, created_at, updated_at
		FROM paper_accounts WHERE id = ?`, id)

	var account Account
	var createdAt, updatedAt string
	if err := row.Scan(
		&account.ID, &account.Currency, &account.StartingBalance, &account.Balance,
		&createdAt, &updatedAt,
	); err != nil {
		return nil, recordError("account", id, err)
	}
	if err := assignTime("account.created_at", createdAt, &account.CreatedAt); err != nil {
		return nil, err
	}
	if err := assignTime("account.updated_at", updatedAt, &account.UpdatedAt); err != nil {
		return nil, err
	}
	return &account, nil
}

// SavePosition inserts or updates a position.
func (s *Store) SavePosition(ctx context.Context, accountID string, position Position) error {
	return savePosition(ctx, s.db, accountID, position)
}

// SavePosition inserts or updates a position in this transaction.
func (tx *Tx) SavePosition(ctx context.Context, accountID string, position Position) error {
	return savePosition(ctx, tx.runner, accountID, position)
}

func savePosition(ctx context.Context, runner sqlRunner, accountID string, position Position) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO paper_positions
		    (id, account_id, symbol, side, volume_lots, contract_size, entry_price,
		     current_price, stop_loss, take_profit, unrealized_pnl, status, opened_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		    account_id=excluded.account_id,
		    symbol=excluded.symbol,
		    side=excluded.side,
		    volume_lots=excluded.volume_lots,
		    contract_size=excluded.contract_size,
		    entry_price=excluded.entry_price,
		    current_price=excluded.current_price,
		    stop_loss=excluded.stop_loss,
		    take_profit=excluded.take_profit,
		    unrealized_pnl=excluded.unrealized_pnl,
		    status=excluded.status,
		    opened_at=excluded.opened_at,
		    updated_at=excluded.updated_at`,
		position.ID, accountID, position.Symbol, position.Side, position.VolumeLots,
		position.ContractSize, position.EntryPrice, position.CurrentPrice,
		nullableFloat(position.StopLoss), nullableFloat(position.TakeProfit),
		position.UnrealizedPnL, position.Status,
		formatTime(position.OpenedAt), formatTime(position.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("paper: save position %q: %w", position.ID, err)
	}
	return nil
}

// GetPosition returns one position by ID.
func (s *Store) GetPosition(ctx context.Context, id string) (*Position, error) {
	return getPosition(ctx, s.db, id)
}

// GetPosition returns one position by ID from this transaction.
func (tx *Tx) GetPosition(ctx context.Context, id string) (*Position, error) {
	return getPosition(ctx, tx.runner, id)
}

func getPosition(ctx context.Context, runner sqlRunner, id string) (*Position, error) {
	row := runner.QueryRowContext(ctx, positionSelect+" WHERE id = ?", id)
	position, err := scanPosition(row.Scan)
	if err != nil {
		return nil, recordError("position", id, err)
	}
	return position, nil
}

const positionSelect = `
	SELECT id, symbol, side, volume_lots, contract_size, entry_price, current_price,
	       stop_loss, take_profit, unrealized_pnl, status, opened_at, updated_at
	FROM paper_positions`

// ListOpenPositions returns open positions for an account ordered oldest first.
func (s *Store) ListOpenPositions(ctx context.Context, accountID string) ([]Position, error) {
	return listOpenPositions(ctx, s.db, accountID)
}

func (tx *Tx) ListOpenPositions(ctx context.Context, accountID string) ([]Position, error) {
	return listOpenPositions(ctx, tx.runner, accountID)
}

func listOpenPositions(ctx context.Context, runner sqlRunner, accountID string) ([]Position, error) {
	rows, err := runner.QueryContext(ctx,
		positionSelect+" WHERE account_id = ? AND status = ? ORDER BY opened_at ASC",
		accountID, PositionStatusOpen,
	)
	if err != nil {
		return nil, fmt.Errorf("paper: list open positions: %w", err)
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		position, err := scanPosition(rows.Scan)
		if err != nil {
			return nil, err
		}
		positions = append(positions, *position)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("paper: list open positions: %w", err)
	}
	return positions, nil
}

func (tx *Tx) DailyRealizedPnL(ctx context.Context, accountID string, since time.Time) (float64, error) {
	return dailyRealizedPnL(ctx, tx.runner, accountID, since)
}

func (s *Store) DailyRealizedPnL(ctx context.Context, accountID string, since time.Time) (float64, error) {
	return dailyRealizedPnL(ctx, s.db, accountID, since)
}

func dailyRealizedPnL(ctx context.Context, runner sqlRunner, accountID string, since time.Time) (float64, error) {
	var result float64
	err := runner.QueryRowContext(ctx, `SELECT COALESCE(SUM(realized_pnl), 0) FROM paper_trades
		WHERE account_id = ? AND closed_at >= ?`, accountID, formatTime(since)).Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("paper: calculate daily realized P/L: %w", err)
	}
	return result, nil
}

func scanPosition(scan func(...any) error) (*Position, error) {
	var position Position
	var side, status, openedAt, updatedAt string
	var stopLoss, takeProfit sql.NullFloat64
	if err := scan(
		&position.ID, &position.Symbol, &side, &position.VolumeLots,
		&position.ContractSize, &position.EntryPrice, &position.CurrentPrice,
		&stopLoss, &takeProfit, &position.UnrealizedPnL, &status, &openedAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	position.Side = Side(side)
	position.Status = PositionStatus(status)
	position.StopLoss = floatPointerFromNull(stopLoss)
	position.TakeProfit = floatPointerFromNull(takeProfit)
	if err := assignTime("position.opened_at", openedAt, &position.OpenedAt); err != nil {
		return nil, err
	}
	if err := assignTime("position.updated_at", updatedAt, &position.UpdatedAt); err != nil {
		return nil, err
	}
	return &position, nil
}

// SaveOrder inserts or updates an order.
func (s *Store) SaveOrder(ctx context.Context, accountID string, order Order) error {
	return saveOrder(ctx, s.db, accountID, order)
}

// SaveOrder inserts or updates an order in this transaction.
func (tx *Tx) SaveOrder(ctx context.Context, accountID string, order Order) error {
	return saveOrder(ctx, tx.runner, accountID, order)
}

func saveOrder(ctx context.Context, runner sqlRunner, accountID string, order Order) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO paper_orders
		    (id, account_id, client_request_id, position_id, symbol, side, order_type,
		     volume_lots, fill_price, stop_loss, take_profit, status, rejection_reason,
		     created_at, filled_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		    account_id=excluded.account_id,
		    client_request_id=excluded.client_request_id,
		    position_id=excluded.position_id,
		    symbol=excluded.symbol,
		    side=excluded.side,
		    order_type=excluded.order_type,
		    volume_lots=excluded.volume_lots,
		    fill_price=excluded.fill_price,
		    stop_loss=excluded.stop_loss,
		    take_profit=excluded.take_profit,
		    status=excluded.status,
		    rejection_reason=excluded.rejection_reason,
		    created_at=excluded.created_at,
		    filled_at=excluded.filled_at`,
		order.ID, accountID, order.ClientRequestID, nullableString(order.PositionID),
		order.Symbol, order.Side, order.Type, order.VolumeLots, order.FillPrice,
		nullableFloat(order.StopLoss), nullableFloat(order.TakeProfit), order.Status,
		order.RejectionReason, formatTime(order.CreatedAt), nullableTime(order.FilledAt),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return fmt.Errorf("%w: %q", ErrDuplicateClientRequestID, order.ClientRequestID)
		}
		return fmt.Errorf("paper: save order %q: %w", order.ID, err)
	}
	return nil
}

// GetOrder returns one order by ID.
func (s *Store) GetOrder(ctx context.Context, id string) (*Order, error) {
	return getOrder(ctx, s.db, "id", id)
}

// GetOrderByClientRequestID returns the order previously created for an idempotency key.
func (s *Store) GetOrderByClientRequestID(ctx context.Context, clientRequestID string) (*Order, error) {
	return getOrder(ctx, s.db, "client_request_id", clientRequestID)
}

// GetOrderByClientRequestID returns an order from this transaction.
func (tx *Tx) GetOrderByClientRequestID(ctx context.Context, clientRequestID string) (*Order, error) {
	return getOrder(ctx, tx.runner, "client_request_id", clientRequestID)
}

func getOrder(ctx context.Context, runner sqlRunner, field, value string) (*Order, error) {
	query := orderSelect + " WHERE " + field + " = ?"
	order, err := scanOrder(runner.QueryRowContext(ctx, query, value).Scan)
	if err != nil {
		return nil, recordError("order", value, err)
	}
	return order, nil
}

const orderSelect = `
	SELECT id, client_request_id, COALESCE(position_id, ''), symbol, side, order_type,
	       volume_lots, fill_price, stop_loss, take_profit, status, rejection_reason,
	       created_at, filled_at
	FROM paper_orders`

// ListRecentOrders returns newest orders first.
func (s *Store) ListRecentOrders(ctx context.Context, accountID string, limit int) ([]Order, error) {
	limit = boundedLimit(limit)
	rows, err := s.db.QueryContext(ctx,
		orderSelect+" WHERE account_id = ? ORDER BY created_at DESC LIMIT ?",
		accountID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("paper: list recent orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		order, err := scanOrder(rows.Scan)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("paper: list recent orders: %w", err)
	}
	return orders, nil
}

func scanOrder(scan func(...any) error) (*Order, error) {
	var order Order
	var side, orderType, status, createdAt string
	var stopLoss, takeProfit sql.NullFloat64
	var filledAt sql.NullString
	if err := scan(
		&order.ID, &order.ClientRequestID, &order.PositionID, &order.Symbol,
		&side, &orderType, &order.VolumeLots, &order.FillPrice,
		&stopLoss, &takeProfit, &status, &order.RejectionReason,
		&createdAt, &filledAt,
	); err != nil {
		return nil, err
	}
	order.Side = Side(side)
	order.Type = OrderType(orderType)
	order.Status = OrderStatus(status)
	order.StopLoss = floatPointerFromNull(stopLoss)
	order.TakeProfit = floatPointerFromNull(takeProfit)
	if err := assignTime("order.created_at", createdAt, &order.CreatedAt); err != nil {
		return nil, err
	}
	if filledAt.Valid {
		var value time.Time
		if err := assignTime("order.filled_at", filledAt.String, &value); err != nil {
			return nil, err
		}
		order.FilledAt = &value
	}
	return &order, nil
}

// SaveTrade inserts an immutable closed-trade record.
func (s *Store) SaveTrade(ctx context.Context, accountID string, trade Trade) error {
	return saveTrade(ctx, s.db, accountID, trade)
}

// SaveTrade inserts an immutable closed-trade record in this transaction.
func (tx *Tx) SaveTrade(ctx context.Context, accountID string, trade Trade) error {
	return saveTrade(ctx, tx.runner, accountID, trade)
}

// SaveAuditEvent appends a safety-relevant event in this transaction.
func (tx *Tx) SaveAuditEvent(ctx context.Context, event AuditEvent) error {
	_, err := tx.runner.ExecContext(ctx, `INSERT INTO paper_audit_events
		(id, account_id, action, entity_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		event.ID, event.AccountID, event.Action, event.EntityID, formatTime(event.CreatedAt))
	if err != nil {
		return fmt.Errorf("paper: save audit event %q: %w", event.ID, err)
	}
	return nil
}

// ListAuditEvents returns newest events first.
func (s *Store) ListAuditEvents(ctx context.Context, accountID string, limit int) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, account_id, action, entity_id, created_at
		FROM paper_audit_events WHERE account_id = ? ORDER BY created_at DESC LIMIT ?`, accountID, boundedLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("paper: list audit events: %w", err)
	}
	defer rows.Close()
	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		var created string
		if err := rows.Scan(&event.ID, &event.AccountID, &event.Action, &event.EntityID, &created); err != nil {
			return nil, err
		}
		if err := assignTime("audit.created_at", created, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func saveTrade(ctx context.Context, runner sqlRunner, accountID string, trade Trade) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO paper_trades
		    (id, account_id, position_id, open_order_id, close_order_id, symbol, side,
		     volume_lots, contract_size, entry_price, exit_price, gross_pnl, fees,
		     realized_pnl, close_reason, opened_at, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trade.ID, accountID, trade.PositionID, trade.OpenOrderID, trade.CloseOrderID,
		trade.Symbol, trade.Side, trade.VolumeLots, trade.ContractSize,
		trade.EntryPrice, trade.ExitPrice, trade.GrossPnL, trade.Fees,
		trade.RealizedPnL, trade.CloseReason,
		formatTime(trade.OpenedAt), formatTime(trade.ClosedAt),
	)
	if err != nil {
		return fmt.Errorf("paper: save trade %q: %w", trade.ID, err)
	}
	return nil
}

// GetTrade returns one trade by ID.
func (s *Store) GetTrade(ctx context.Context, id string) (*Trade, error) {
	trade, err := scanTrade(s.db.QueryRowContext(ctx, tradeSelect+" WHERE id = ?", id).Scan)
	if err != nil {
		return nil, recordError("trade", id, err)
	}
	return trade, nil
}

const tradeSelect = `
	SELECT id, position_id, open_order_id, close_order_id, symbol, side, volume_lots,
	       contract_size, entry_price, exit_price, gross_pnl, fees, realized_pnl,
	       close_reason, opened_at, closed_at
	FROM paper_trades`

// ListRecentTrades returns newest closed trades first.
func (s *Store) ListRecentTrades(ctx context.Context, accountID string, limit int) ([]Trade, error) {
	limit = boundedLimit(limit)
	rows, err := s.db.QueryContext(ctx,
		tradeSelect+" WHERE account_id = ? ORDER BY closed_at DESC LIMIT ?",
		accountID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("paper: list recent trades: %w", err)
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		trade, err := scanTrade(rows.Scan)
		if err != nil {
			return nil, err
		}
		trades = append(trades, *trade)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("paper: list recent trades: %w", err)
	}
	return trades, nil
}

func scanTrade(scan func(...any) error) (*Trade, error) {
	var trade Trade
	var side, closeReason, openedAt, closedAt string
	if err := scan(
		&trade.ID, &trade.PositionID, &trade.OpenOrderID, &trade.CloseOrderID,
		&trade.Symbol, &side, &trade.VolumeLots, &trade.ContractSize,
		&trade.EntryPrice, &trade.ExitPrice, &trade.GrossPnL, &trade.Fees,
		&trade.RealizedPnL, &closeReason, &openedAt, &closedAt,
	); err != nil {
		return nil, err
	}
	trade.Side = Side(side)
	trade.CloseReason = CloseReason(closeReason)
	if err := assignTime("trade.opened_at", openedAt, &trade.OpenedAt); err != nil {
		return nil, err
	}
	if err := assignTime("trade.closed_at", closedAt, &trade.ClosedAt); err != nil {
		return nil, err
	}
	return &trade, nil
}

func recordError(kind, id string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s %q", ErrNotFound, kind, id)
	}
	return fmt.Errorf("paper: read %s %q: %w", kind, id, err)
}

func isUniqueConstraint(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUniqueCode
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func floatPointerFromNull(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func assignTime(field, raw string, destination *time.Time) error {
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return fmt.Errorf("paper: parse %s: %w", field, err)
	}
	*destination = value
	return nil
}
