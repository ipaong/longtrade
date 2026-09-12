package paper

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const testAccountID = "paper-main"

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	workspace := t.TempDir()
	store, err := NewStore(workspace)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return store, workspace
}

func testTime() time.Time {
	return time.Date(2026, time.September, 12, 10, 11, 12, 123456000, time.UTC)
}

func testAccount() Account {
	now := testTime()
	return Account{
		ID:              testAccountID,
		Currency:        "USD",
		StartingBalance: 10000,
		Balance:         10000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func testPosition(id string, status PositionStatus) Position {
	now := testTime()
	return Position{
		ID:            id,
		Symbol:        SymbolXAUUSD,
		Side:          SideBuy,
		VolumeLots:    0.01,
		ContractSize:  100,
		EntryPrice:    2400.20,
		CurrentPrice:  2401.80,
		StopLoss:      floatPointer(2380),
		TakeProfit:    floatPointer(2450),
		UnrealizedPnL: 1.60,
		Status:        status,
		OpenedAt:      now,
		UpdatedAt:     now.Add(time.Minute),
	}
}

func testOrder(id, clientRequestID, positionID string, side Side) Order {
	now := testTime()
	return Order{
		ID:              id,
		ClientRequestID: clientRequestID,
		PositionID:      positionID,
		Symbol:          SymbolXAUUSD,
		Side:            side,
		Type:            OrderTypeMarket,
		VolumeLots:      0.01,
		FillPrice:       2400.20,
		StopLoss:        floatPointer(2380),
		TakeProfit:      floatPointer(2450),
		Status:          OrderStatusFilled,
		CreatedAt:       now,
		FilledAt:        &now,
	}
}

func TestNewStoreCreatesVersionedPrivateDatabase(t *testing.T) {
	store, workspace := newTestStore(t)

	if got, want := store.Path(), DatabasePath(workspace); got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("Stat(database) error = %v", err)
	}

	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, currentSchemaVersion)
	}

	var foreignKeys int
	if err := store.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(store.Path())
		if err != nil {
			t.Fatalf("Stat(database) error = %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("database permissions = %04o, want 0600", got)
		}
	}
}

func TestStoreRoundTripAndReopen(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	store, err := NewStore(workspace)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	account := testAccount()
	position := testPosition("position-1", PositionStatusClosed)
	openOrder := testOrder("order-open", "request-open", position.ID, SideBuy)
	closeOrder := testOrder("order-close", "request-close", position.ID, SideSell)
	closeOrder.FillPrice = 2410
	closedAt := testTime().Add(time.Hour)
	trade := Trade{
		ID:           "trade-1",
		PositionID:   position.ID,
		OpenOrderID:  openOrder.ID,
		CloseOrderID: closeOrder.ID,
		Symbol:       SymbolXAUUSD,
		Side:         SideBuy,
		VolumeLots:   0.01,
		ContractSize: 100,
		EntryPrice:   2400.20,
		ExitPrice:    2410,
		GrossPnL:     9.80,
		Fees:         0.50,
		RealizedPnL:  9.30,
		CloseReason:  CloseReasonManual,
		OpenedAt:     testTime(),
		ClosedAt:     closedAt,
	}

	if err := store.WithTx(ctx, func(tx *Tx) error {
		if err := tx.SaveAccount(ctx, account); err != nil {
			return err
		}
		if err := tx.SavePosition(ctx, account.ID, position); err != nil {
			return err
		}
		if err := tx.SaveOrder(ctx, account.ID, openOrder); err != nil {
			return err
		}
		if err := tx.SaveOrder(ctx, account.ID, closeOrder); err != nil {
			return err
		}
		return tx.SaveTrade(ctx, account.ID, trade)
	}); err != nil {
		t.Fatalf("WithTx() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	store, err = NewStore(workspace)
	if err != nil {
		t.Fatalf("reopen NewStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	gotAccount, err := store.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if *gotAccount != account {
		t.Fatalf("GetAccount() = %#v, want %#v", *gotAccount, account)
	}

	gotPosition, err := store.GetPosition(ctx, position.ID)
	if err != nil {
		t.Fatalf("GetPosition() error = %v", err)
	}
	if gotPosition.ID != position.ID || gotPosition.Status != PositionStatusClosed {
		t.Fatalf("GetPosition() = %#v", gotPosition)
	}
	if gotPosition.StopLoss == nil || *gotPosition.StopLoss != *position.StopLoss {
		t.Fatalf("StopLoss = %v, want %v", gotPosition.StopLoss, position.StopLoss)
	}
	if !gotPosition.OpenedAt.Equal(position.OpenedAt) {
		t.Fatalf("OpenedAt = %v, want %v", gotPosition.OpenedAt, position.OpenedAt)
	}

	gotOrder, err := store.GetOrderByClientRequestID(ctx, openOrder.ClientRequestID)
	if err != nil {
		t.Fatalf("GetOrderByClientRequestID() error = %v", err)
	}
	if gotOrder.ID != openOrder.ID || gotOrder.FilledAt == nil || !gotOrder.FilledAt.Equal(*openOrder.FilledAt) {
		t.Fatalf("GetOrderByClientRequestID() = %#v", gotOrder)
	}

	gotTrade, err := store.GetTrade(ctx, trade.ID)
	if err != nil {
		t.Fatalf("GetTrade() error = %v", err)
	}
	if gotTrade.RealizedPnL != trade.RealizedPnL || !gotTrade.ClosedAt.Equal(trade.ClosedAt) {
		t.Fatalf("GetTrade() = %#v, want realized P/L %.2f at %v", gotTrade, trade.RealizedPnL, trade.ClosedAt)
	}

	recentTrades, err := store.ListRecentTrades(ctx, account.ID, 1)
	if err != nil {
		t.Fatalf("ListRecentTrades() error = %v", err)
	}
	if len(recentTrades) != 1 || recentTrades[0].ID != trade.ID {
		t.Fatalf("ListRecentTrades() = %#v, want %q", recentTrades, trade.ID)
	}
}

func TestStoreListsOpenPositionsAndRecentRecords(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	account := testAccount()
	if err := store.SaveAccount(ctx, account); err != nil {
		t.Fatalf("SaveAccount() error = %v", err)
	}

	openPosition := testPosition("position-open", PositionStatusOpen)
	closedPosition := testPosition("position-closed", PositionStatusClosed)
	if err := store.SavePosition(ctx, account.ID, openPosition); err != nil {
		t.Fatalf("SavePosition(open) error = %v", err)
	}
	if err := store.SavePosition(ctx, account.ID, closedPosition); err != nil {
		t.Fatalf("SavePosition(closed) error = %v", err)
	}

	positions, err := store.ListOpenPositions(ctx, account.ID)
	if err != nil {
		t.Fatalf("ListOpenPositions() error = %v", err)
	}
	if len(positions) != 1 || positions[0].ID != openPosition.ID {
		t.Fatalf("ListOpenPositions() = %#v, want only %q", positions, openPosition.ID)
	}

	for index, id := range []string{"order-1", "order-2"} {
		order := testOrder(id, "request-"+id, openPosition.ID, SideBuy)
		order.CreatedAt = order.CreatedAt.Add(time.Duration(index) * time.Minute)
		if err := store.SaveOrder(ctx, account.ID, order); err != nil {
			t.Fatalf("SaveOrder(%q) error = %v", id, err)
		}
	}
	orders, err := store.ListRecentOrders(ctx, account.ID, 1)
	if err != nil {
		t.Fatalf("ListRecentOrders() error = %v", err)
	}
	if len(orders) != 1 || orders[0].ID != "order-2" {
		t.Fatalf("ListRecentOrders() = %#v, want order-2", orders)
	}
}

func TestStoreRejectsDuplicateClientRequestID(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	account := testAccount()
	position := testPosition("position-1", PositionStatusOpen)
	if err := store.SaveAccount(ctx, account); err != nil {
		t.Fatalf("SaveAccount() error = %v", err)
	}
	if err := store.SavePosition(ctx, account.ID, position); err != nil {
		t.Fatalf("SavePosition() error = %v", err)
	}
	if err := store.SaveOrder(ctx, account.ID, testOrder("order-1", "same-request", position.ID, SideBuy)); err != nil {
		t.Fatalf("SaveOrder(first) error = %v", err)
	}

	err := store.SaveOrder(ctx, account.ID, testOrder("order-2", "same-request", position.ID, SideBuy))
	if !errors.Is(err, ErrDuplicateClientRequestID) {
		t.Fatalf("SaveOrder(duplicate) error = %v, want ErrDuplicateClientRequestID", err)
	}
}

func TestStoreTransactionRollsBack(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	wantErr := errors.New("stop transaction")

	err := store.WithTx(ctx, func(tx *Tx) error {
		if err := tx.SaveAccount(ctx, testAccount()); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithTx() error = %v, want %v", err, wantErr)
	}

	if _, err := store.GetAccount(ctx, testAccountID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAccount() error = %v, want ErrNotFound", err)
	}
}

func TestStoreEnforcesForeignKeys(t *testing.T) {
	store, _ := newTestStore(t)
	err := store.SavePosition(context.Background(), "missing-account", testPosition("position-1", PositionStatusOpen))
	if err == nil {
		t.Fatal("SavePosition() error = nil, want foreign-key error")
	}
}

func TestStoreRejectsNewerSchema(t *testing.T) {
	workspace := t.TempDir()
	store, err := NewStore(workspace)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.db.Exec("PRAGMA user_version = 999"); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err = NewStore(workspace)
	if err == nil {
		t.Fatal("NewStore() error = nil, want newer-schema error")
	}

	// The failed open must leave the existing database in place for a newer build.
	if _, statErr := os.Stat(filepath.Join(workspace, "memory", "paper", "paper.db")); statErr != nil {
		t.Fatalf("Stat(database) error = %v", statErr)
	}
}

func TestStoreReturnsNotFound(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	for name, call := range map[string]func() error{
		"account": func() error {
			_, err := store.GetAccount(ctx, "missing")
			return err
		},
		"position": func() error {
			_, err := store.GetPosition(ctx, "missing")
			return err
		},
		"order": func() error {
			_, err := store.GetOrder(ctx, "missing")
			return err
		},
		"trade": func() error {
			_, err := store.GetTrade(ctx, "missing")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
		})
	}
}
