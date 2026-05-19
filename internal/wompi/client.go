package wompi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PaymentMethod represent the method used for the transaction.
type PaymentMethod struct {
	Type         string `json:"type"`                   // e.g. "CARD", "NEQUI", "PSE", "BANCOLOMBIA_TRANSFER"
	Token        string `json:"token,omitempty"`        // Used for CARD
	Installments int    `json:"installments,omitempty"` // Used for CARD
	PhoneNumber  string `json:"phone_number,omitempty"` // For Nequi
	// For PSE and BANCOLOMBIA_TRANSFER
	UserType      string `json:"user_type,omitempty"`    // "PERSON" for Bancolombia, "0" or "1" for PSE
	UserDocument  string `json:"user_legal_id,omitempty"`
	UserDocType   string `json:"user_legal_id_type,omitempty"`
	FinancialInst string `json:"financial_institution_code,omitempty"`
	// For BANCOLOMBIA_TRANSFER
	PaymentDescription string `json:"payment_description,omitempty"` // Max 64 chars, no single quotes
	EcommerceURL       string `json:"ecommerce_url,omitempty"`       // URL to redirect after payment
	SandboxStatus      string `json:"sandbox_status,omitempty"`      // For sandbox testing: APPROVED, DECLINED, ERROR
}

// TransactionRequest is the payload to create a new transaction.
type TransactionRequest struct {
	AcceptanceToken string        `json:"acceptance_token"`
	AmountInCents   int64         `json:"amount_in_cents"`
	Currency        string        `json:"currency"`
	Signature       string        `json:"signature"`
	CustomerEmail   string        `json:"customer_email"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	Reference       string        `json:"reference"`
	RedirectURL     string        `json:"redirect_url,omitempty"` // Required for async methods (Bancolombia, PSE)
	IP              string        `json:"ip,omitempty"`
}

// TransactionResponse is the response from Wompi when creating or fetching a transaction.
type TransactionResponse struct {
	Data struct {
		ID                string `json:"id"`
		Reference         string `json:"reference"`
		AmountInCents     int64  `json:"amount_in_cents"`
		Currency          string `json:"currency"`
		Status            string `json:"status"`
		StatusMessage     string `json:"status_message"`
		PaymentMethodType string `json:"payment_method_type"`
		PaymentMethod     struct {
			Type  string `json:"type"`
			Extra struct {
				AsyncPaymentURL string `json:"async_payment_url,omitempty"` // For Bancolombia/PSE redirect
				Name            string `json:"name,omitempty"`
				Brand           string `json:"brand,omitempty"`
				LastFour        string `json:"last_four,omitempty"`
			} `json:"extra"`
		} `json:"payment_method"`
		RedirectURL string `json:"redirect_url"`
	} `json:"data"`
}

// MerchantResponse contains the merchant details, which includes the presigned acceptance token.
type MerchantResponse struct {
	Data struct {
		PresignedAcceptance struct {
			AcceptanceToken string `json:"acceptance_token"`
		} `json:"presigned_acceptance"`
	} `json:"data"`
}

// CreateTransaction sends a POST request to Wompi to process a payment.
func (s *Service) CreateTransaction(req TransactionRequest) (*TransactionResponse, error) {
	// Re-generate signature just to be safe if it wasn't provided
	if req.Signature == "" {
		req.Signature = s.GenerateIntegritySignature(req.Reference, req.AmountInCents, req.Currency)
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction request: %w", err)
	}

	url := fmt.Sprintf("%s/transactions", s.BaseURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.PrivateKey))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wompi API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var transactionResponse TransactionResponse
	if err := json.Unmarshal(bodyBytes, &transactionResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return &transactionResponse, nil
}

// GetTransaction queries Wompi for the current status of a transaction by ID.
func (s *Service) GetTransaction(transactionID string) (*TransactionResponse, error) {
	url := fmt.Sprintf("%s/transactions/%s", s.BaseURL, transactionID)
	
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// For querying transactions, the public key is usually sufficient according to Wompi docs,
	// but private key also works. We use PublicKey here as it's common practice for read-only.
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.PublicKey))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wompi API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var transactionResponse TransactionResponse
	if err := json.Unmarshal(bodyBytes, &transactionResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return &transactionResponse, nil
}

// GetPresignedAcceptanceToken fetches the merchant info to get the acceptance token required for transactions.
func (s *Service) GetPresignedAcceptanceToken() (string, error) {
	url := fmt.Sprintf("%s/merchants/%s", s.BaseURL, s.PublicKey)
	
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wompi API error fetching merchant info (status %d)", resp.StatusCode)
	}

	var merchantResponse MerchantResponse
	if err := json.NewDecoder(resp.Body).Decode(&merchantResponse); err != nil {
		return "", fmt.Errorf("failed to parse merchant response JSON: %w", err)
	}

	return merchantResponse.Data.PresignedAcceptance.AcceptanceToken, nil
}
