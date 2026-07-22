package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
)

func TestDebitBalance_NeverOversells(t *testing.T) {
	repo := NewCustomerRepository()
	ctx := context.Background()

	const startingBalance = int64(10_000)
	const price = int64(100)
	const attempts = 500

	c := &domain.Customer{ID: "cust-1", Name: "Test", Balance: startingBalance, CreatedAt: time.Now()}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.DebitBalance(ctx, c.ID, price)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else if !errors.Is(err, domain.ErrInsufficientBalance) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	final, err := repo.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}

	if final.Balance < 0 {
		t.Fatalf("balance went negative: %d", final.Balance)
	}

	expectedSuccesses := startingBalance / price
	if successCount != expectedSuccesses {
		t.Fatalf("expected exactly %d successful debits, got %d", expectedSuccesses, successCount)
	}

	expectedBalance := startingBalance - successCount*price
	if final.Balance != expectedBalance {
		t.Fatalf("expected final balance %d, got %d", expectedBalance, final.Balance)
	}
}

func TestDebitBalance_InsufficientFunds(t *testing.T) {
	repo := NewCustomerRepository()
	ctx := context.Background()
	c := &domain.Customer{ID: "cust-2", Name: "Test", Balance: 50, CreatedAt: time.Now()}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	if _, err := repo.DebitBalance(ctx, c.ID, 100); !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
	final, _ := repo.Get(ctx, c.ID)
	if final.Balance != 50 {
		t.Fatalf("balance should be untouched on failed debit, got %d", final.Balance)
	}
}

func TestDebitBalance_UnknownCustomer(t *testing.T) {
	repo := NewCustomerRepository()
	ctx := context.Background()
	if _, err := repo.DebitBalance(ctx, "does-not-exist", 10); !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestCreditBalance(t *testing.T) {
	repo := NewCustomerRepository()
	ctx := context.Background()
	c := &domain.Customer{ID: "cust-3", Name: "Test", Balance: 0, CreatedAt: time.Now()}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	newBal, err := repo.CreditBalance(ctx, c.ID, 500)
	if err != nil {
		t.Fatalf("credit balance: %v", err)
	}
	if newBal != 500 {
		t.Fatalf("expected balance 500, got %d", newBal)
	}
}
