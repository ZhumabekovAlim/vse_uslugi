package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAirbapayCreateInvoiceResponse_UnmarshalJSON_SnakeCase(t *testing.T) {
	payload := []byte(`{
        "invoice_id": "inv-123",
        "order_id": "ord-456",
        "payment_url": "https://pay.example/123",
        "status": "ok",
        "message": "created"
    }`)

	var resp AirbapayCreateInvoiceResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.InvoiceID != "inv-123" {
		t.Errorf("invoice id mismatch: %q", resp.InvoiceID)
	}
	if resp.OrderID != "ord-456" {
		t.Errorf("order id mismatch: %q", resp.OrderID)
	}
	if resp.PaymentURL != "https://pay.example/123" {
		t.Errorf("payment url mismatch: %q", resp.PaymentURL)
	}
	if resp.Status != "ok" {
		t.Errorf("status mismatch: %q", resp.Status)
	}
	if resp.Message != "created" {
		t.Errorf("message mismatch: %q", resp.Message)
	}
	if string(resp.Raw) != string(payload) {
		t.Errorf("raw payload mismatch: %q", string(resp.Raw))
	}
}

func TestAirbapayCreateInvoiceResponse_UnmarshalJSON_CamelCase(t *testing.T) {
	payload := []byte(`{
        "invoiceId": "inv-123",
        "orderId": "ord-456",
        "paymentUrl": "https://pay.example/123",
        "status": "ok",
        "message": "created"
    }`)

	var resp AirbapayCreateInvoiceResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.InvoiceID != "inv-123" {
		t.Errorf("invoice id mismatch: %q", resp.InvoiceID)
	}
	if resp.OrderID != "ord-456" {
		t.Errorf("order id mismatch: %q", resp.OrderID)
	}
	if resp.PaymentURL != "https://pay.example/123" {
		t.Errorf("payment url mismatch: %q", resp.PaymentURL)
	}
	if resp.Status != "ok" {
		t.Errorf("status mismatch: %q", resp.Status)
	}
	if resp.Message != "created" {
		t.Errorf("message mismatch: %q", resp.Message)
	}
	if string(resp.Raw) != string(payload) {
		t.Errorf("raw payload mismatch: %q", string(resp.Raw))
	}
}

func TestCreatePaymentLink_Non2xxReturnsAirbapayError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer ts.Close()

	svc, err := NewAirbapayService(AirbapayConfig{
		Username:   "user",
		Password:   "pass",
		TerminalID: "terminal",
		BaseURL:    ts.URL,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = svc.CreatePaymentLink(context.Background(), 1, 10.0, "test")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	apiErr, ok := err.(*AirbapayError)
	if !ok {
		t.Fatalf("expected AirbapayError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status code: %d", apiErr.StatusCode)
	}
	if apiErr.Status != "404 Not Found" {
		t.Errorf("unexpected status: %q", apiErr.Status)
	}
	if apiErr.Body == "" {
		t.Errorf("expected body to be populated")
	}
}
