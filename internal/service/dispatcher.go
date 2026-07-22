package service

import (
	"context"
	"log"
	"time"

	"github.com/iman/sms-gateway/internal/domain"
)

type Dispatcher struct {
	router       *OperatorRouter
	messages     domain.MessageRepository
	normalQueue  chan *domain.Message
	expressQueue chan *domain.Message
	now          func() time.Time
}

func NewDispatcher(router *OperatorRouter, messages domain.MessageRepository, normalWorkers, expressWorkers, queueSize int) *Dispatcher {
	d := &Dispatcher{
		router:       router,
		messages:     messages,
		normalQueue:  make(chan *domain.Message, queueSize),
		expressQueue: make(chan *domain.Message, queueSize),
		now:          time.Now,
	}
	for i := 0; i < normalWorkers; i++ {
		go d.worker(d.normalQueue)
	}
	for i := 0; i < expressWorkers; i++ {
		go d.worker(d.expressQueue)
	}
	return d
}

func (d *Dispatcher) Enqueue(m *domain.Message) {
	if m.Type == domain.MessageTypeExpress {
		d.expressQueue <- m
		return
	}
	d.normalQueue <- m
}

func (d *Dispatcher) worker(queue chan *domain.Message) {
	for m := range queue {
		d.process(m)
	}
}

func (d *Dispatcher) process(m *domain.Message) {
	ctx := context.Background()
	if m.DeliveryDeadline != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, *m.DeliveryDeadline)
		defer cancel()
	}

	operator, err := d.router.Route(m)
	if err != nil {
		d.markFailed(ctx, m)
		return
	}

	if err := operator.Send(ctx, m); err != nil {
		log.Printf("dispatcher: operator %s rejected message %s: %v", operator.Name(), m.ID, err)
		d.markFailed(ctx, m)
		return
	}

	if err := d.messages.MarkSent(ctx, m.ID, operator.Name(), d.now()); err != nil {
		log.Printf("dispatcher: failed to persist sent status for %s: %v", m.ID, err)
	}
}

func (d *Dispatcher) markFailed(ctx context.Context, m *domain.Message) {
	if err := d.messages.MarkFailed(ctx, m.ID); err != nil {
		log.Printf("dispatcher: failed to persist failed status for %s: %v", m.ID, err)
	}
}
