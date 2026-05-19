package notifications

import (
	"fmt"
	"log"
	"sync"
)

// OrderItem represents a single item in the order (a cookie flavor).
type OrderItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// OrderNotification contains all the details needed to notify about an order.
type OrderNotification struct {
	Reference    string      `json:"reference"`
	CustomerName string      `json:"customer_name"`
	Email        string      `json:"email"`
	Phone        string      `json:"phone"`
	City         string      `json:"city"`
	Address      string      `json:"address"`
	Neighborhood string      `json:"neighborhood"`
	Notes        string      `json:"notes"`
	Items        []OrderItem `json:"items"`
	TotalAmount  int64       `json:"total_amount"` // in cents
	Currency     string      `json:"currency"`
	PaymentMethod string     `json:"payment_method"`
}

// Notifier is the interface that notification channels must implement.
type Notifier interface {
	Send(order OrderNotification) error
	Name() string
}

// Service orchestrates sending notifications through multiple channels.
type Service struct {
	notifiers []Notifier
}

// NewService creates a new notification service with the given notifiers.
func NewService(notifiers ...Notifier) *Service {
	return &Service{notifiers: notifiers}
}

// NotifyAll sends the order notification through all registered channels in parallel.
// It logs errors but never fails — notifications should not block the webhook response.
func (s *Service) NotifyAll(order OrderNotification) {
	var wg sync.WaitGroup

	for _, n := range s.notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			if err := notifier.Send(order); err != nil {
				log.Printf("⚠️ Error enviando notificación por %s: %v", notifier.Name(), err)
			} else {
				log.Printf("✅ Notificación enviada por %s para referencia %s", notifier.Name(), order.Reference)
			}
		}(n)
	}

	wg.Wait()
}

// FormatOrderText returns a plain text summary of the order for notifications.
func FormatOrderText(order OrderNotification) string {
	total := fmt.Sprintf("$%s", formatMoney(order.TotalAmount))

	text := fmt.Sprintf("🍪 NUEVO PEDIDO BISCOLI\n")
	text += fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━\n")
	text += fmt.Sprintf("📋 Referencia: %s\n", order.Reference)
	text += fmt.Sprintf("💰 Total: %s %s\n", total, order.Currency)
	text += fmt.Sprintf("💳 Método: %s\n\n", order.PaymentMethod)

	text += fmt.Sprintf("👤 CLIENTE\n")
	text += fmt.Sprintf("Nombre: %s\n", order.CustomerName)
	text += fmt.Sprintf("Email: %s\n", order.Email)
	text += fmt.Sprintf("Teléfono: %s\n\n", order.Phone)

	text += fmt.Sprintf("📦 ENTREGA\n")
	text += fmt.Sprintf("Ciudad: %s\n", order.City)
	text += fmt.Sprintf("Dirección: %s\n", order.Address)
	text += fmt.Sprintf("Barrio: %s\n", order.Neighborhood)
	if order.Notes != "" {
		text += fmt.Sprintf("Notas: %s\n", order.Notes)
	}

	text += fmt.Sprintf("\n🛒 PRODUCTOS\n")
	for _, item := range order.Items {
		text += fmt.Sprintf("  • %s x%d\n", item.Name, item.Quantity)
	}

	return text
}

// formatMoney converts cents to a formatted string (e.g. 1500000 -> "15,000")
func formatMoney(cents int64) string {
	amount := cents / 100
	if amount < 1000 {
		return fmt.Sprintf("%d", amount)
	}

	// Simple thousands separator
	s := fmt.Sprintf("%d", amount)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}
