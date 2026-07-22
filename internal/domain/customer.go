package domain

import "time"

type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Balance   int64     `json:"balance"` // smallest currency unit (Rial), never a float
	CreatedAt time.Time `json:"created_at"`
}
