package database

import (
	"context"
	"fmt"
)

// CreateOrder inserts a new order into the database with a PENDING status.
func (db *DB) CreateOrder(ctx context.Context, reference string, amount int64, currency string, customerEmail string, orderDetails []byte) error {
	query := `
		INSERT INTO orders (reference, amount_in_cents, currency, customer_email, order_details, status)
		VALUES ($1, $2, $3, $4, $5, 'PENDING')
	`
	_, err := db.Pool.Exec(ctx, query, reference, amount, currency, customerEmail, orderDetails)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}
	return nil
}

// UpdateOrderWompiID updates the wompi_transaction_id for a specific order.
func (db *DB) UpdateOrderWompiID(ctx context.Context, reference string, wompiTransactionID string) error {
	query := `
		UPDATE orders
		SET wompi_transaction_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE reference = $2
	`
	_, err := db.Pool.Exec(ctx, query, wompiTransactionID, reference)
	if err != nil {
		return fmt.Errorf("failed to update order wompi id: %w", err)
	}
	return nil
}

// UpdateOrderStatus updates the status of an order based on the wompi_transaction_id.
// Common statuses: PENDING, APPROVED, DECLINED, ERROR
func (db *DB) UpdateOrderStatus(ctx context.Context, wompiTransactionID string, status string) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE wompi_transaction_id = $2
	`
	_, err := db.Pool.Exec(ctx, query, status, wompiTransactionID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// UpdateOrderStatusByReference updates the status of an order based on our internal reference.
func (db *DB) UpdateOrderStatusByReference(ctx context.Context, reference string, status string) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE reference = $2
	`
	_, err := db.Pool.Exec(ctx, query, status, reference)
	if err != nil {
		return fmt.Errorf("failed to update order status by reference: %w", err)
	}
	return nil
}

// SaveWompiWebhook stores the raw webhook payload.
func (db *DB) SaveWompiWebhook(ctx context.Context, transactionID string, eventType string, payload []byte) error {
	query := `
		INSERT INTO wompi_webhooks (transaction_id, event_type, payload)
		VALUES ($1, $2, $3)
	`
	_, err := db.Pool.Exec(ctx, query, transactionID, eventType, payload)
	if err != nil {
		return fmt.Errorf("failed to save webhook: %w", err)
	}
	return nil
}

// GetOrderByReference retrieves an order by its reference, including order_details.
func (db *DB) GetOrderByReference(ctx context.Context, reference string) (*Order, error) {
	query := `
		SELECT id, reference, wompi_transaction_id, amount_in_cents, currency, status, customer_email, order_details, created_at, updated_at
		FROM orders
		WHERE reference = $1
	`
	row := db.Pool.QueryRow(ctx, query, reference)

	var order Order
	err := row.Scan(
		&order.ID,
		&order.Reference,
		&order.WompiTransactionID,
		&order.AmountInCents,
		&order.Currency,
		&order.Status,
		&order.CustomerEmail,
		&order.OrderDetails,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order by reference: %w", err)
	}
	return &order, nil
}
