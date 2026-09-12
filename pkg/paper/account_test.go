package paper

import (
	"context"
	"testing"
	"time"
)

func TestNewStoreCreatesDefaultAccount(t *testing.T) {
	store, _ := newTestStore(t)

	account, err := store.GetAccount(context.Background(), DefaultAccountID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if account.Currency != DefaultAccountCurrency {
		t.Fatalf("Currency = %q, want %q", account.Currency, DefaultAccountCurrency)
	}
	if account.StartingBalance != DefaultStartingBalance || account.Balance != DefaultStartingBalance {
		t.Fatalf("balances = starting %.2f/current %.2f, want %.2f", account.StartingBalance, account.Balance, DefaultStartingBalance)
	}
	if account.CreatedAt.IsZero() || account.UpdatedAt.IsZero() {
		t.Fatalf("timestamps must be populated: %#v", account)
	}
}

func TestEnsureDefaultAccountPreservesExistingState(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	account, err := store.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	account.Balance = 9_125.50
	account.UpdatedAt = testTime()
	if err := store.SaveAccount(ctx, *account); err != nil {
		t.Fatalf("SaveAccount() error = %v", err)
	}

	got, err := store.EnsureDefaultAccount(ctx, testTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("EnsureDefaultAccount() error = %v", err)
	}
	if got.Balance != account.Balance || !got.UpdatedAt.Equal(account.UpdatedAt) {
		t.Fatalf("EnsureDefaultAccount() = %#v, want existing state %#v", got, account)
	}
}

func TestResetDefaultAccountClearsLedgerAndRestoresBalance(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	seedPaperLedger(t, store)
	resetAt := testTime().Add(2 * time.Hour)

	account, err := store.ResetDefaultAccount(ctx, resetAt)
	if err != nil {
		t.Fatalf("ResetDefaultAccount() error = %v", err)
	}
	if account.Balance != DefaultStartingBalance || account.StartingBalance != DefaultStartingBalance {
		t.Fatalf("reset balances = %#v", account)
	}
	if !account.CreatedAt.Equal(resetAt) || !account.UpdatedAt.Equal(resetAt) {
		t.Fatalf("reset timestamps = created %v/updated %v, want %v", account.CreatedAt, account.UpdatedAt, resetAt)
	}
	assertLedgerCounts(t, store, 0, 0, 0)
	var audits int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM paper_audit_events WHERE account_id = ?", DefaultAccountID).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("audit count = %d, error %v", audits, err)
	}
}

func TestResetDefaultAccountRollsBackAtomically(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	seedPaperLedger(t, store)
	before, err := store.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if _, err := store.db.Exec(`
		CREATE TRIGGER fail_paper_order_delete
		BEFORE DELETE ON paper_orders
		BEGIN SELECT RAISE(ABORT, 'forced reset failure'); END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if _, err := store.ResetDefaultAccount(ctx, testTime().Add(3*time.Hour)); err == nil {
		t.Fatal("ResetDefaultAccount() error = nil, want forced failure")
	}
	after, err := store.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		t.Fatalf("GetAccount() after failed reset error = %v", err)
	}
	if after.Balance != before.Balance || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("account changed after rollback: before %#v, after %#v", before, after)
	}
	assertLedgerCounts(t, store, 1, 2, 1)
}

func seedPaperLedger(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	account, err := store.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	account.Balance = 9_500
	account.UpdatedAt = testTime()
	position := testPosition("reset-position", PositionStatusClosed)
	openOrder := testOrder("reset-open", "reset-request-open", position.ID, SideBuy)
	closeOrder := testOrder("reset-close", "reset-request-close", position.ID, SideSell)
	trade := Trade{
		ID: "reset-trade", PositionID: position.ID, OpenOrderID: openOrder.ID, CloseOrderID: closeOrder.ID,
		Symbol: SymbolXAUUSD, Side: SideBuy, VolumeLots: 0.01, ContractSize: 100,
		EntryPrice: 2400, ExitPrice: 2405, GrossPnL: 5, RealizedPnL: 5,
		CloseReason: CloseReasonManual, OpenedAt: testTime(), ClosedAt: testTime().Add(time.Hour),
	}
	if err := store.WithTx(ctx, func(tx *Tx) error {
		if err := tx.SaveAccount(ctx, *account); err != nil {
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
		t.Fatalf("seed ledger: %v", err)
	}
}

func assertLedgerCounts(t *testing.T, store *Store, positions, orders, trades int) {
	t.Helper()
	for table, want := range map[string]int{
		"paper_positions": positions, "paper_orders": orders, "paper_trades": trades,
	} {
		var got int
		if err := store.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE account_id = ?", DefaultAccountID).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s count = %d, want %d", table, got, want)
		}
	}
}
