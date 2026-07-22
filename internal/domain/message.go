package domain

import "time"

type MessageType string

const (
	MessageTypeNormal  MessageType = "normal"
	MessageTypeOTP     MessageType = "otp"
	MessageTypeExpress MessageType = "express"
)

func (t MessageType) Valid() bool {
	switch t {
	case MessageTypeNormal, MessageTypeOTP, MessageTypeExpress:
		return true
	}
	return false
}

// MessageStatus tracks the lifecycle of a message from acceptance to
// final delivery outcome. A message only ever moves forward through
// these states.
type MessageStatus string

const (
	// StatusQueued: balance has been reserved/debited and the message is
	// waiting to be handed off to an operator.
	StatusQueued MessageStatus = "queued"
	// StatusSent: successfully handed off to (accepted by) the operator.
	StatusSent MessageStatus = "sent"
	// StatusDelivered: operator confirmed final delivery to the handset.
	StatusDelivered MessageStatus = "delivered"
	// StatusFailed: could not be delivered; in this implementation the
	// only failure path is operator rejection, since balance is checked
	// up front (a message is never accepted without funds to cover it).
	StatusFailed MessageStatus = "failed"
)

// Message is a single outbound SMS record, owned by exactly one Customer.
type Message struct {
	ID         string        `json:"id"`
	CustomerID string        `json:"customer_id"`
	Sender     string        `json:"sender"`
	Receiver   string        `json:"receiver"`
	Body       string        `json:"body"`
	Type       MessageType   `json:"type"`
	Price      int64         `json:"price"` // amount actually debited, Rials
	Status     MessageStatus `json:"status"`
	Operator   string        `json:"operator"`
	// DeliveryDeadline is only meaningful for Express messages: the time
	// by which the operator handoff is contractually guaranteed to happen.
	DeliveryDeadline *time.Time `json:"delivery_deadline,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	SentAt           *time.Time `json:"sent_at,omitempty"`
}

type PriceList struct {
	Normal  int64
	OTP     int64
	Express int64
}

func (p PriceList) For(t MessageType) int64 {
	switch t {
	case MessageTypeOTP:
		return p.OTP
	case MessageTypeExpress:
		return p.Express
	default:
		return p.Normal
	}
}

// DefaultPriceList gives sane defaults; overridable via configuration.
func DefaultPriceList() PriceList {
	return PriceList{
		Normal:  100, // Rials
		OTP:     150,
		Express: 300,
	}
}
