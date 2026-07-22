package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
)

type OperatorGateway interface {
	Name() string
	Send(ctx context.Context, m *domain.Message) error
}

type OperatorRouter struct {
	standard []OperatorGateway
	express  []OperatorGateway
}

func NewOperatorRouter(standard, express []OperatorGateway) *OperatorRouter {
	return &OperatorRouter{standard: standard, express: express}
}

func (r *OperatorRouter) Route(m *domain.Message) (OperatorGateway, error) {
	pool := r.standard
	if m.Type == domain.MessageTypeExpress {
		pool = r.express
	}
	if len(pool) == 0 {
		return nil, errors.New("no operator available for message type")
	}
	return pool[rand.Intn(len(pool))], nil
}

type SimulatedOperator struct {
	name        string
	minLatency  time.Duration
	maxLatency  time.Duration
	failureRate float64 // 0..1
}

func NewSimulatedOperator(name string, minLatency, maxLatency time.Duration, failureRate float64) *SimulatedOperator {
	return &SimulatedOperator{name: name, minLatency: minLatency, maxLatency: maxLatency, failureRate: failureRate}
}

func (o *SimulatedOperator) Name() string { return o.name }

func (o *SimulatedOperator) Send(ctx context.Context, m *domain.Message) error {
	delta := o.maxLatency - o.minLatency
	wait := o.minLatency
	if delta > 0 {
		wait += time.Duration(rand.Int63n(int64(delta)))
	}
	select {
	case <-time.After(wait):
	case <-ctx.Done():
		return ctx.Err()
	}
	if o.failureRate > 0 && rand.Float64() < o.failureRate {
		return errors.New("operator rejected message")
	}
	return nil
}
