package domain

import "errors"

var (
	ErrCustomerNotFound    = errors.New("customer not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidMessageType  = errors.New("invalid message type")
	ErrInvalidInput        = errors.New("invalid input")
	ErrMessageNotFound     = errors.New("message not found")
)
