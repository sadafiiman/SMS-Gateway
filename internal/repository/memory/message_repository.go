package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
)

type MessageRepository struct {
	mu   sync.RWMutex
	data map[string]*domain.Message
	byCustomer map[string][]string
}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{
		data:       make(map[string]*domain.Message),
		byCustomer: make(map[string][]string),
	}
}

func (r *MessageRepository) Save(ctx context.Context, m *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *m
	r.data[m.ID] = &cp
	r.byCustomer[m.CustomerID] = append([]string{m.ID}, r.byCustomer[m.CustomerID]...)
	return nil
}

func (r *MessageRepository) MarkSent(ctx context.Context, id string, operator string, sentAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.data[id]
	if !ok {
		return domain.ErrMessageNotFound
	}
	m.Status = domain.StatusSent
	m.Operator = operator
	m.SentAt = &sentAt
	return nil
}

func (r *MessageRepository) MarkFailed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.data[id]
	if !ok {
		return domain.ErrMessageNotFound
	}
	m.Status = domain.StatusFailed
	return nil
}

func (r *MessageRepository) Get(ctx context.Context, id string) (*domain.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.data[id]
	if !ok {
		return nil, domain.ErrMessageNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *MessageRepository) List(ctx context.Context, filter domain.MessageFilter) ([]*domain.Message, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byCustomer[filter.CustomerID]
	matched := make([]*domain.Message, 0, len(ids))
	for _, id := range ids {
		m := r.data[id]
		if m == nil {
			continue
		}
		if filter.Type != "" && m.Type != filter.Type {
			continue
		}
		if filter.Status != "" && m.Status != filter.Status {
			continue
		}
		cp := *m
		matched = append(matched, &cp)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	total := len(matched)

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []*domain.Message{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matched[offset:end], total, nil
}
