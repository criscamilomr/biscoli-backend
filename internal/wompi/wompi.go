package wompi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Service provides methods to interact with Wompi's security requirements.
type Service struct {
	IntegritySecret string
	EventsSecret    string // Used for webhook verification
	PrivateKey      string // Used for REST API requests
	PublicKey       string // Used for public REST API requests
	BaseURL         string // Sandbox or Production URL
}

// NewService creates a new Wompi service with the necessary secrets.
func NewService(integritySecret, eventsSecret, privateKey, publicKey, environment string) *Service {
	baseURL := "https://sandbox.wompi.co/v1"
	if environment == "production" {
		baseURL = "https://production.wompi.co/v1"
	}

	return &Service{
		IntegritySecret: integritySecret,
		EventsSecret:    eventsSecret,
		PrivateKey:      privateKey,
		PublicKey:       publicKey,
		BaseURL:         baseURL,
	}
}

// GenerateIntegritySignature creates the SHA256 hash required by Wompi's Web Widget.
// The string to hash is composed of: reference + amountInCents + currency + integritySecret
func (s *Service) GenerateIntegritySignature(reference string, amountInCents int64, currency string) string {
	rawString := fmt.Sprintf("%s%d%s%s", reference, amountInCents, currency, s.IntegritySecret)
	
	hash := sha256.New()
	hash.Write([]byte(rawString))
	
	return hex.EncodeToString(hash.Sum(nil))
}

// VerifyWebhookSignature verifies if the incoming webhook actually comes from Wompi
// using the Events Secret and the dynamic properties array.
func (s *Service) VerifyWebhookSignature(payload []byte) (bool, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return false, fmt.Errorf("error parsing webhook payload: %v", err)
	}

	// Extract timestamp
	timestampFloat, ok := raw["timestamp"].(float64)
	if !ok {
		return false, fmt.Errorf("missing or invalid timestamp")
	}
	timestamp := fmt.Sprintf("%.0f", timestampFloat)

	// Extract signature object
	signatureRaw, ok := raw["signature"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("missing signature object")
	}

	checksum, ok := signatureRaw["checksum"].(string)
	if !ok {
		return false, fmt.Errorf("missing checksum in signature")
	}

	propertiesRaw, ok := signatureRaw["properties"].([]interface{})
	if !ok {
		return false, fmt.Errorf("missing properties in signature")
	}

	// Extract data object
	dataRaw, ok := raw["data"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("missing data object")
	}

	// Concatenate values
	var concatenated string
	for _, propRaw := range propertiesRaw {
		prop, ok := propRaw.(string)
		if !ok {
			return false, fmt.Errorf("invalid property in signature.properties")
		}

		// The property is typically something like "transaction.id" or "transaction.status"
		// We need to traverse the 'data' map
		parts := strings.Split(prop, ".")
		
		var currentVal interface{} = dataRaw
		for _, part := range parts {
			if m, isMap := currentVal.(map[string]interface{}); isMap {
				currentVal = m[part]
			} else {
				currentVal = nil
				break
			}
		}

		// Convert currentVal to string based on its type
		if currentVal != nil {
			switch v := currentVal.(type) {
			case string:
				concatenated += v
			case float64:
				// Typically IDs or amount_in_cents might be numbers
				// Wompi docs say concatenate them as they are
				// Use %.0f to avoid scientific notation and decimals
				concatenated += fmt.Sprintf("%.0f", v)
			default:
				concatenated += fmt.Sprintf("%v", v)
			}
		}
	}

	// Add timestamp and events secret
	concatenated += timestamp + s.EventsSecret

	// Calculate SHA256
	hash := sha256.New()
	hash.Write([]byte(concatenated))
	calculatedChecksum := hex.EncodeToString(hash.Sum(nil))

	// Return true if the calculated hash matches Wompi's checksum case-insensitively
	return strings.EqualFold(calculatedChecksum, checksum), nil
}
