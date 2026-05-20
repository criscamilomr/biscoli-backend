package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/xxkmi/biscoli-backend/internal/wompi"
)

// OrderItemDetail represents a single item in the order for notification purposes.
type OrderItemDetail struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// OrderDetails contains customer and delivery info sent from the frontend.
type OrderDetails struct {
	CustomerName string            `json:"customer_name"`
	Phone        string            `json:"phone"`
	City         string            `json:"city"`
	Address      string            `json:"address"`
	Neighborhood string            `json:"neighborhood"`
	Notes        string            `json:"notes"`
	Items        []OrderItemDetail `json:"items"`
}

// CreatePaymentRequest represents the payload from the frontend to create a payment.
type CreatePaymentRequest struct {
	AmountInCents int64               `json:"amount_in_cents"`
	Currency      string              `json:"currency"`
	CustomerEmail string              `json:"customer_email"`
	PaymentMethod wompi.PaymentMethod `json:"payment_method"`
	RedirectURL   string              `json:"redirect_url,omitempty"` // For async methods (Bancolombia, PSE)
	OrderDetails  *OrderDetails       `json:"order_details,omitempty"`
}

// HandleCreatePayment handles the POST /api/payments endpoint.
func (h *PaymentHandler) HandleCreatePayment(w http.ResponseWriter, r *http.Request) {
	// CORS headers should ideally be handled by a middleware, but adding it here for simplicity
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Get Presigned Acceptance Token
	acceptanceToken, err := h.wompiService.GetPresignedAcceptanceToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get acceptance token: %v", err), http.StatusInternalServerError)
		return
	}

	// 2. Generate a unique reference
	reference := fmt.Sprintf("BISCOLI-%s", uuid.New().String())

	// 3. Get User IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		} else {
			ip = host
		}
	} else {
		// X-Forwarded-For can contain multiple IPs, take the first
		ip = strings.Split(ip, ",")[0]
		ip = strings.TrimSpace(ip)
	}
	// Fallback: localhost IPv6 (::1) to IPv4 for Wompi compatibility
	if ip == "::1" || ip == "[::1]" {
		ip = "127.0.0.1"
	}

	// 4. Serialize order details to JSON for DB storage
	var orderDetailsJSON []byte
	if req.OrderDetails != nil {
		orderDetailsJSON, err = json.Marshal(req.OrderDetails)
		if err != nil {
			http.Error(w, "Failed to serialize order details", http.StatusInternalServerError)
			return
		}
	}

	// 5. Save to DB BEFORE calling Wompi
	err = h.db.CreateOrder(r.Context(), reference, req.AmountInCents, req.Currency, req.CustomerEmail, orderDetailsJSON)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save order in database: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Build Transaction Request
	transactionReq := wompi.TransactionRequest{
		AcceptanceToken: acceptanceToken,
		AmountInCents:   req.AmountInCents,
		Currency:        req.Currency,
		CustomerEmail:   req.CustomerEmail,
		PaymentMethod:   req.PaymentMethod,
		Reference:       reference,
		RedirectURL:     req.RedirectURL,
		IP:              ip,
	}

	// 6. Create Transaction in Wompi
	transactionResp, err := h.wompiService.CreateTransaction(transactionReq)
	if err != nil {
		// Update DB to mark as ERROR
		_ = h.db.UpdateOrderStatusByReference(r.Context(), reference, "ERROR")
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// 7. Update DB with Wompi Transaction ID and initial Status
	err = h.db.UpdateOrderWompiID(r.Context(), reference, transactionResp.Data.ID)
	if err != nil {
		// Log error but we already have the transaction in Wompi
		fmt.Printf("Warning: failed to update order Wompi ID: %v\n", err)
	}
	err = h.db.UpdateOrderStatus(r.Context(), transactionResp.Data.ID, transactionResp.Data.Status)
	if err != nil {
		fmt.Printf("Warning: failed to update initial order status: %v\n", err)
	}

	// 8. Return response to frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transactionResp)
}

// HandleGetPaymentStatus handles the GET /api/payments/{id} endpoint.
func (h *PaymentHandler) HandleGetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the ID from the URL path. Assuming route is /api/payments/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	transactionID := pathParts[3]

	if transactionID == "" {
		http.Error(w, "Transaction ID required", http.StatusBadRequest)
		return
	}

	transactionResp, err := h.wompiService.GetTransaction(transactionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get transaction status: %v", err), http.StatusInternalServerError)
		return
	}

	// Fallback/Local Development: Check database status and send notification if APPROVED but still marked PENDING in DB.
	// Since Wompi webhooks cannot reach localhost, this serves as the primary mechanism for local testing.
	order, err := h.db.GetOrderByReference(r.Context(), transactionResp.Data.Reference)
	if err == nil && order != nil {
		if order.Status == "PENDING" && transactionResp.Data.Status != "PENDING" {
			txStatus := transactionResp.Data.Status
			reference := transactionResp.Data.Reference

			log.Printf("Sondeo de transacción final detectado. Referencia: %s | Estado: %s. Actualizando DB.\n", reference, txStatus)
			
			// Update status in DB
			_ = h.db.UpdateOrderStatusByReference(r.Context(), reference, txStatus)

			// If APPROVED, trigger notifications
			if txStatus == "APPROVED" {
				log.Println("✅ ¡Pago aprobado detectado por sondeo! Enviando notificaciones...")
				go h.sendOrderNotifications(reference, transactionResp.Data.AmountInCents)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactionResp)
}
