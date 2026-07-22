package memory

import (
	"context"
	"sync"

	"github.com/iman/sms-gateway/internal/domain"
)

type customerRecord struct {
	mu       sync.Mutex
	customer domain.Customer
}

type CustomerRepository struct {
	mu   sync.RWMutex
	data map[string]*customerRecord
}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{
		data: make(map[string]*customerRecord),
	}
}

func (r *CustomerRepository) Create(ctx context.Context, c *domain.Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.data[c.ID]; exists {
		return domain.ErrInvalidInput
	}
	r.data[c.ID] = &customerRecord{customer: *c}
	return nil
}

func (r *CustomerRepository) getRecord(id string) *customerRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data[id]
}

func (r *CustomerRepository) Get(ctx context.Context, id string) (*domain.Customer, error) {
	rec := r.getRecord(id)
	if rec == nil {
		return nil, domain.ErrCustomerNotFound
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	cp := rec.customer
	return &cp, nil
}

func (r *CustomerRepository) DebitBalance(ctx context.Context, id string, amount int64) (int64, error) {
	rec := r.getRecord(id)
	if rec == nil {
		return 0, domain.ErrCustomerNotFound
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()

	if rec.customer.Balance < amount {
		return rec.customer.Balance, domain.ErrInsufficientBalance
	}
	rec.customer.Balance -= amount
	return rec.customer.Balance, nil
}

func (r *CustomerRepository) CreditBalance(ctx context.Context, id string, amount int64) (int64, error) {
	rec := r.getRecord(id)
	if rec == nil {
		return 0, domain.ErrCustomerNotFound
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.customer.Balance += amount
	return rec.customer.Balance, nil
}
