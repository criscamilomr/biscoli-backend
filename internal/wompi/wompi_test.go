package wompi

import (
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	// Payload directly from Wompi's documentation
	payload := []byte(`{
		"event": "transaction.updated",
		"data": {
			"transaction": {
				"id": "1234-1610641025-49201",
				"amount_in_cents": 4490000,
				"reference": "MZQ3X2DE2SMX",
				"customer_email": "juan.perez@gmail.com",
				"currency": "COP",
				"payment_method_type": "NEQUI",
				"redirect_url": "https://mitienda.com.co/pagos/redireccion",
				"status": "APPROVED",
				"shipping_address": null,
				"payment_link_id": null,
				"payment_source_id": null
			}
		},
		"environment": "prod",
		"signature": {
			"properties": [
				"transaction.id",
				"transaction.status",
				"transaction.amount_in_cents"
			],
			"checksum": "5A18EC5E8FDB7DF463E9F94774CBA8F583BA21BD04A09CEFF2EA68A4BC0AEFBE"
		},
		"timestamp": 1530291411,
		"sent_at": "2018-07-20T16:45:05.000Z"
	}`)

	// "prod_events_OcHnIzeBl5socpwByQ4hA52Em3USQ93Z"
	s := NewService("dummy_integrity", "prod_events_OcHnIzeBl5socpwByQ4hA52Em3USQ93Z")

	isValid, err := s.VerifyWebhookSignature(payload)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !isValid {
		t.Errorf("Expected signature to be valid, but got invalid")
	}
}
