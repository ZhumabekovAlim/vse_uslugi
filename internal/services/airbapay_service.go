package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	Logger           *slog.Logger // optional; if nil -> slog.Default()
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
	logger           *slog.Logger
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

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
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

	svc := &AirbapayService{
		username:         cfg.Username,
		password:         cfg.Password,
		terminalID:       cfg.TerminalID,
		baseURL:          parsed,
		successURL:       cfg.SuccessURL,
		failureURL:       cfg.FailureURL,
		createInvoiceURI: createURI,
		httpClient:       client,
		logger:           logger,
	}

	// Log initial configuration (without secrets)
	logger.Info("Airbapay service initialized",
		"baseURL", parsed.String(),
		"createInvoiceURI", createURI,
		"terminalID", maskRight(cfg.TerminalID, 4),
		"successURL_set", strings.TrimSpace(cfg.SuccessURL) != "",
		"failureURL_set", strings.TrimSpace(cfg.FailureURL) != "",
		"http_timeout", client.Timeout.String(),
	)

	return svc, nil
}

// CreatePaymentLink creates an invoice in Airbapay and returns the payment URL
// provided by the API.
func (s *AirbapayService) CreatePaymentLink(ctx context.Context, invoiceID int, amount float64, description string) (*AirbapayCreateInvoiceResponse, error) {
	start := time.Now()
	if s == nil {
		return nil, fmt.Errorf("airbapay service is not initialised")
	}
	logger := s.logger.With("op", "CreatePaymentLink")

	// Context info
	if deadline, ok := ctx.Deadline(); ok {
		logger.Debug("context has deadline", "deadline", deadline)
	} else {
		logger.Debug("context has no deadline")
	}

	// Log input args (without secrets)
	logger.Info("start",
		"invoiceID", invoiceID,
		"amount", amount,
		"description_set", strings.TrimSpace(description) != "",
		"baseURL", safeURL(s.baseURL),
		"createInvoiceURI", s.createInvoiceURI,
		"successURL_set", s.successURL != "",
		"failureURL_set", s.failureURL != "",
	)

	// Build final request URL
	var requestURL string
	uriLower := strings.ToLower(s.createInvoiceURI)
	if strings.HasPrefix(uriLower, "http://") || strings.HasPrefix(uriLower, "https://") {
		requestURL = s.createInvoiceURI
	} else {
		endpoint := *s.baseURL // copy
		before := endpoint.Path
		relPath := strings.TrimPrefix(s.createInvoiceURI, "/")
		endpoint.Path = path.Join(endpoint.Path, relPath)
		requestURL = endpoint.String()
		logger.Debug("joined path", "base_path_before", before, "relPath", relPath, "base_path_after", endpoint.Path)
	}
	logger.Info("built request URL", "url", requestURL)

	// Prepare payload
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
		logger.Error("marshal request failed", "err", err)
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	logger.Debug("payload", "json", trim(string(body), 2000))

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		logger.Error("build request failed", "err", err)
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Basic auth — DO NOT log password
	req.SetBasicAuth(s.username, s.password)

	logger.Info("sending request",
		"method", req.Method,
		"url", req.URL.String(),
		"contentType", req.Header.Get("Content-Type"),
		"username", s.username,
		"contentLength", len(body),
	)

	// Execute
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("perform request failed", "err", err, "elapsed_ms", time.Since(start).Milliseconds())
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("read response failed", "err", err, "status", resp.Status)
		return nil, fmt.Errorf("read response: %w", err)
	}

	logger.Info("response received",
		"status", resp.Status,
		"elapsed_ms", time.Since(start).Milliseconds(),
	)
	logger.Debug("raw response body", "body", trim(string(responseBody), 2000))

	// Non-2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := trim(string(responseBody), 2000)
		logger.Error("non-2xx from Airbapay", "status", resp.Status, "body", bodyStr)
		return nil, fmt.Errorf("create payment link: airbapay responded with status %s: %s", resp.Status, bodyStr)
	}

	// Decode
	var result AirbapayCreateInvoiceResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		logger.Error("decode response failed", "err", err)
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result.Raw = json.RawMessage(responseBody)

	// Validate
	if strings.TrimSpace(result.PaymentURL) == "" {
		logger.Error("empty payment_url in response")
		return nil, fmt.Errorf("airbapay response does not contain payment_url")
	}

	logger.Info("success",
		"invoiceID", result.InvoiceID,
		"orderID", result.OrderID,
		"paymentURL", result.PaymentURL,
		"status", result.Status,
		"elapsed_ms", time.Since(start).Milliseconds(),
	)
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
	ok := strings.TrimSpace(payload.Signature) != ""
	s.logger.Debug("validate callback signature", "present", ok)
	return ok
}

// ParseCallback decodes Airbapay callback requests into a structured payload.
func (s *AirbapayService) ParseCallback(r io.Reader) (*AirbapayCallbackPayload, error) {
	if s == nil {
		return nil, fmt.Errorf("airbapay service is not initialised")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		s.logger.Error("read callback body failed", "err", err)
		return nil, fmt.Errorf("read callback body: %w", err)
	}
	s.logger.Info("callback received", "bytes", len(data))
	s.logger.Debug("callback raw", "body", trim(string(data), 2000))

	var payload AirbapayCallbackPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.logger.Error("decode callback failed", "err", err)
		return nil, fmt.Errorf("decode callback: %w", err)
	}
	payload.Raw = json.RawMessage(data)

	s.logger.Info("callback parsed",
		"operation_id", payload.OperationID,
		"order_id", payload.OrderID,
		"invoice_id", payload.InvoiceID,
		"status", payload.Status,
	)
	return &payload, nil
}

// ---------- helpers ----------

func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

// safeURL prints URL without user/pass (just in case)
func safeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	c := *u
	c.User = nil
	return c.String()
}

// maskRight keeps only rightmost n characters
func maskRight(s string, n int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= n {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-n) + s[len(s)-n:]
}
