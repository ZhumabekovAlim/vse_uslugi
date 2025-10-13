package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// AirbapayConfig describes the configuration required for authenticating against
// the Airbapay acquiring API.
type AirbapayConfig struct {
	Username         string
	Password         string
	TerminalID       string
	BaseURL          string
	SuccessURL       string
	FailureURL       string
	CreateInvoiceURI string
	Client           *http.Client
}

// AirbapayService is responsible for communication with the Airbapay API.
type AirbapayService struct {
	username         string
	password         string
	terminalID       string
	baseURL          *url.URL
	successURL       string
	failureURL       string
	createInvoiceURI string
	httpClient       *http.Client
}

// AirbapayAmount represents the payment amount in the request payload.
type AirbapayAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// AirbapayCreateInvoiceRequest encapsulates data sent to Airbapay when creating
// a payment invoice.
type AirbapayCreateInvoiceRequest struct {
	TerminalID   string            `json:"terminal_id"`
	OrderID      string            `json:"order_id"`
	Description  string            `json:"description,omitempty"`
	Amount       AirbapayAmount    `json:"amount"`
	SuccessURL   string            `json:"success_url,omitempty"`
	FailureURL   string            `json:"failure_url,omitempty"`
	ExtraFields  map[string]string `json:"extra_fields,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
	CustomerInfo map[string]string `json:"customer_info,omitempty"`
}

// AirbapayCreateInvoiceResponse mirrors the relevant part of the Airbapay
// response when a payment invoice is created.
type AirbapayCreateInvoiceResponse struct {
	InvoiceID  string          `json:"invoice_id"`
	OrderID    string          `json:"order_id"`
	PaymentURL string          `json:"payment_url"`
	Status     string          `json:"status"`
	Message    string          `json:"message"`
	Raw        json.RawMessage `json:"-"`
}

func (r *AirbapayCreateInvoiceResponse) UnmarshalJSON(data []byte) error {
	type responseAlias struct {
		InvoiceIDSnake  string `json:"invoice_id"`
		InvoiceIDCamel  string `json:"invoiceId"`
		OrderIDSnake    string `json:"order_id"`
		OrderIDCamel    string `json:"orderId"`
		PaymentURLSnake string `json:"payment_url"`
		PaymentURLCamel string `json:"paymentUrl"`
		Status          string `json:"status"`
		Message         string `json:"message"`
	}

	var alias responseAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	r.InvoiceID = firstNonEmpty(alias.InvoiceIDSnake, alias.InvoiceIDCamel)
	r.OrderID = firstNonEmpty(alias.OrderIDSnake, alias.OrderIDCamel)
	r.PaymentURL = firstNonEmpty(alias.PaymentURLSnake, alias.PaymentURLCamel)
	r.Status = alias.Status
	r.Message = alias.Message
	r.Raw = append(r.Raw[:0], data...)

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// AirbapayCallbackPayload describes the payload sent by Airbapay to the result
// callback endpoint.
type AirbapayCallbackPayload struct {
	OperationID string          `json:"operation_id"`
	OrderID     string          `json:"order_id"`
	InvoiceID   string          `json:"invoice_id"`
	Status      string          `json:"status"`
	Amount      AirbapayAmount  `json:"amount"`
	Signature   string          `json:"signature"`
	Extra       map[string]any  `json:"extra_fields"`
	Metadata    map[string]any  `json:"metadata"`
	Raw         json.RawMessage `json:"-"`
}

// NewAirbapayService creates an instance of AirbapayService configured with the
// supplied options. Missing optional fields are initialised with sensible defaults.
func NewAirbapayService(cfg AirbapayConfig) (*AirbapayService, error) {
	if strings.TrimSpace(cfg.Username) == "" {
		return nil, fmt.Errorf("airbapay username is required")
	}
	if strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("airbapay password is required")
	}
	if strings.TrimSpace(cfg.TerminalID) == "" {
		return nil, fmt.Errorf("airbapay terminal_id is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("airbapay base url is required")
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	createURI := cfg.CreateInvoiceURI
	if strings.TrimSpace(createURI) == "" {
		createURI = "/v1/invoice/create"
	}

	return &AirbapayService{
		username:         cfg.Username,
		password:         cfg.Password,
		terminalID:       cfg.TerminalID,
		baseURL:          parsed,
		successURL:       cfg.SuccessURL,
		failureURL:       cfg.FailureURL,
		createInvoiceURI: createURI,
		httpClient:       client,
	}, nil
}

// CreatePaymentLink creates an invoice in Airbapay and returns the payment URL
// provided by the API.
func (s *AirbapayService) CreatePaymentLink(ctx context.Context, invoiceID int, amount float64, description string) (*AirbapayCreateInvoiceResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("airbapay service is not initialised")
	}

	var requestURL string
	if strings.HasPrefix(strings.ToLower(s.createInvoiceURI), "http://") || strings.HasPrefix(strings.ToLower(s.createInvoiceURI), "https://") {
		requestURL = s.createInvoiceURI
	} else {
		endpoint := *s.baseURL
		relPath := strings.TrimPrefix(s.createInvoiceURI, "/")
		endpoint.Path = path.Join(endpoint.Path, relPath)
		requestURL = endpoint.String()
	}

	payload := AirbapayCreateInvoiceRequest{
		TerminalID:  s.terminalID,
		OrderID:     fmt.Sprintf("%d", invoiceID),
		Description: description,
		Amount: AirbapayAmount{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "KZT",
		},
	}

	if s.successURL != "" {
		payload.SuccessURL = s.successURL
	}
	if s.failureURL != "" {
		payload.FailureURL = s.failureURL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.username, s.password)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("airbapay responded with status %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	var result AirbapayCreateInvoiceResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result.Raw = json.RawMessage(responseBody)

	if strings.TrimSpace(result.PaymentURL) == "" {
		return nil, fmt.Errorf("airbapay response does not contain payment_url")
	}

	return &result, nil
}

// ValidateCallbackSignature attempts to verify the callback signature.
// Airbapay documentation states that callbacks should be signed, however the
// exact signature algorithm may vary. The implementation here merely ensures a
// non-empty signature is present. More specific validation can be added once the
// exact signature algorithm is confirmed.
func (s *AirbapayService) ValidateCallbackSignature(payload *AirbapayCallbackPayload) bool {
	if payload == nil {
		return false
	}
	return strings.TrimSpace(payload.Signature) != ""
}

// ParseCallback decodes Airbapay callback requests into a structured payload.
func (s *AirbapayService) ParseCallback(r io.Reader) (*AirbapayCallbackPayload, error) {
	if s == nil {
		return nil, fmt.Errorf("airbapay service is not initialised")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read callback body: %w", err)
	}
	var payload AirbapayCallbackPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode callback: %w", err)
	}
	payload.Raw = json.RawMessage(data)
	return &payload, nil
}
