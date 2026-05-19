package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xxkmi/biscoli-backend/internal/database"
	"github.com/xxkmi/biscoli-backend/internal/notifications"
	"github.com/xxkmi/biscoli-backend/internal/wompi"
)

type CheckoutRequest struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount_in_cents"` // e.g. 5000000 for 50.000,00 COP
	Currency  string `json:"currency"`        // e.g. "COP"
}

type CheckoutResponse struct {
	Signature string `json:"signature"`
	Reference string `json:"reference"`
}

type PaymentHandler struct {
	wompiService *wompi.Service
	db           *database.DB
	notifier     *notifications.Service
}

func NewPaymentHandler(ws *wompi.Service, db *database.DB, notifier *notifications.Service) *PaymentHandler {
	return &PaymentHandler{
		wompiService: ws,
		db:           db,
		notifier:     notifier,
	}
}

func (h *PaymentHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Aquí se guardaría el pedido en la base de datos con estado "Pendiente"
	// err := database.SaveOrder(req.Reference, req.Amount)

	// 2. Generar la firma de integridad de Wompi
	signature := h.wompiService.GenerateIntegritySignature(req.Reference, req.Amount, req.Currency)

	// 3. Devolver la firma a Angular
	resp := CheckoutResponse{
		Signature: signature,
		Reference: req.Reference,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
