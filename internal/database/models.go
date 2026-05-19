package database

import (
	"time"
)

// Order represents a payment transaction/order in the database.
type Order struct {
	ID                 int       `json:"id"`
	Reference          string    `json:"reference"`
	WompiTransactionID *string   `json:"wompi_transaction_id"` // Can be null initially
	AmountInCents      int64     `json:"amount_in_cents"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	CustomerEmail      string    `json:"customer_email"`
	OrderDetails       []byte    `json:"order_details"` // JSONB with full order info
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// WompiWebhook represents a raw webhook payload received from Wompi.
type WompiWebhook struct {
	ID            int       `json:"id"`
	TransactionID *string   `json:"transaction_id"`
	EventType     string    `json:"event_type"`
	Payload       []byte    `json:"payload"` // JSONB stored as bytes
	CreatedAt     time.Time `json:"created_at"`
}
