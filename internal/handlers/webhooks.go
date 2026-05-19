package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/xxkmi/biscoli-backend/internal/notifications"
)

type WebhookRequest struct {
	Event string `json:"event"`
	Data  struct {
		Transaction struct {
			Id        string `json:"id"`
			Status    string `json:"status"`
			Reference string `json:"reference"`
			Amount    int64  `json:"amount_in_cents"`
		} `json:"transaction"`
	} `json:"data"`
	Signature struct {
		Checksum string `json:"checksum"`
	} `json:"signature"`
	Timestamp int64 `json:"timestamp"`
}

func (h *PaymentHandler) HandleWompiWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Read raw body to preserve exact JSON format for signature verification
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// 2. Verify signature to ensure it comes from Wompi
	isValid, err := h.wompiService.VerifyWebhookSignature(payload)
	if err != nil {
		fmt.Printf("Webhook signature verification error: %v\n", err)
		http.Error(w, "Signature verification failed", http.StatusBadRequest)
		return
	}
	
	if !isValid {
		fmt.Println("Warning: Received Wompi webhook with invalid checksum!")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// 3. Unmarshal the payload to our structured request
	var req WebhookRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 4. Save Raw Webhook to DB
	// Extract transaction ID if present
	var transactionID string
	if req.Data.Transaction.Id != "" {
		transactionID = req.Data.Transaction.Id
	}
	
	err = h.db.SaveWompiWebhook(r.Context(), transactionID, req.Event, payload)
	if err != nil {
		fmt.Printf("Warning: failed to save raw webhook to DB: %v\n", err)
		// We continue processing even if this fails
	}

	// 5. Check if the event is a transaction update
	if req.Event == "transaction.updated" {
		txStatus := req.Data.Transaction.Status
		reference := req.Data.Transaction.Reference

		fmt.Printf("Recibido Webhook de Wompi. Referencia: %s | Estado: %s\n", reference, txStatus)

		// Update order status in DB
		err = h.db.UpdateOrderStatusByReference(r.Context(), reference, txStatus)
		if err != nil {
			fmt.Printf("Error: failed to update order status by reference: %v\n", err)
			
			// Fallback: try by transaction ID if reference didn't work
			if transactionID != "" {
				err = h.db.UpdateOrderStatus(r.Context(), transactionID, txStatus)
				if err != nil {
					fmt.Printf("Error: failed to update order status by transaction ID: %v\n", err)
				}
			}
		}

		// 6. Send notifications on APPROVED payments
		if txStatus == "APPROVED" {
			log.Println("✅ ¡Pago aprobado! Enviando notificaciones...")
			go h.sendOrderNotifications(reference, req.Data.Transaction.Amount)
		} else if txStatus == "DECLINED" || txStatus == "ERROR" {
			log.Printf("❌ Pago rechazado/error para referencia: %s\n", reference)
		}
	}

	// Always return 200 OK to acknowledge receipt of the webhook to Wompi
	w.WriteHeader(http.StatusOK)
}

// sendOrderNotifications retrieves order details from DB and sends notifications.
func (h *PaymentHandler) sendOrderNotifications(reference string, amountInCents int64) {
	if h.notifier == nil {
		return
	}

	ctx := context.Background()

	// Get order from DB
	order, err := h.db.GetOrderByReference(ctx, reference)
	if err != nil {
		log.Printf("⚠️ No se pudo obtener la orden %s para notificación: %v\n", reference, err)
		return
	}

	// Parse order details from JSONB
	var details OrderDetails
	if order.OrderDetails != nil {
		if err := json.Unmarshal(order.OrderDetails, &details); err != nil {
			log.Printf("⚠️ Error deserializando order_details: %v\n", err)
		}
	}

	// Build notification items
	items := make([]notifications.OrderItem, len(details.Items))
	for i, item := range details.Items {
		items[i] = notifications.OrderItem{
			Name:     item.Name,
			Quantity: item.Quantity,
		}
	}

	// Determine payment method type
	paymentMethod := "Desconocido"
	if order.OrderDetails != nil {
		paymentMethod = "Tarjeta/Nequi/Bancolombia"
	}

	notification := notifications.OrderNotification{
		Reference:     order.Reference,
		CustomerName:  details.CustomerName,
		Email:         order.CustomerEmail,
		Phone:         details.Phone,
		City:          details.City,
		Address:       details.Address,
		Neighborhood:  details.Neighborhood,
		Notes:         details.Notes,
		Items:         items,
		TotalAmount:   order.AmountInCents,
		Currency:      order.Currency,
		PaymentMethod: paymentMethod,
	}

	h.notifier.NotifyAll(notification)
}

