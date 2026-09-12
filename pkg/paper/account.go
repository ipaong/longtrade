package paper

import (
	"context"
	"fmt"
	"time"
)

const (
	// DefaultAccountID identifies the single local paper account used by the MVP.
	DefaultAccountID = "paper-main"
	// DefaultAccountCurrency is the denomination of the MVP paper account.
	DefaultAccountCurrency = "USD"
	// DefaultStartingBalance is the cash restored by a paper-account reset.
	DefaultStartingBalance = 10_000.0
)

// EnsureDefaultAccount creates the MVP paper account when it does not exist.
// Existing account state is preserved so opening the store cannot reset a user.
func (s *Store) EnsureDefaultAccount(ctx context.Context, now time.Time) (*Account, error) {
	account := newDefaultAccount(now)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO paper_accounts
		    (id, currency, starting_balance, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		account.ID, account.Currency, account.StartingBalance, account.Balance,
		formatTime(account.CreatedAt), formatTime(account.UpdatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("paper: initialize default account: %w", err)
	}

	result, err := s.GetAccount(ctx, DefaultAccountID)
	if err != nil {
		return nil, fmt.Errorf("paper: read default account: %w", err)
	}
	return result, nil
}

// ResetDefaultAccount atomically removes the paper ledger and restores the
// default account to its initial USD 10,000 balance.
func (s *Store) ResetDefaultAccount(ctx context.Context, now time.Time) (*Account, error) {
	account := newDefaultAccount(now)
	err := s.WithTx(ctx, func(tx *Tx) error {
		// Delete children in foreign-key order. If any statement fails, WithTx
		// rolls every deletion and the account update back together.
		for _, table := range []string{"paper_audit_events", "paper_trades", "paper_orders", "paper_positions"} {
			if _, err := tx.runner.ExecContext(ctx,
				"DELETE FROM "+table+" WHERE account_id = ?", DefaultAccountID,
			); err != nil {
				return fmt.Errorf("paper: reset default account %s: %w", table, err)
			}
		}
		return tx.SaveAccount(ctx, account)
	})
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func newDefaultAccount(now time.Time) Account {
	now = now.UTC()
	return Account{
		ID:              DefaultAccountID,
		Currency:        DefaultAccountCurrency,
		StartingBalance: DefaultStartingBalance,
		Balance:         DefaultStartingBalance,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
