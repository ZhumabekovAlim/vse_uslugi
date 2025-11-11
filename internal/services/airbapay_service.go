package services

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AirbapayConfig struct {
	Username   string
	Password   string
	TerminalID string

	// База эквайринга AirbaPay (прод)
	// Пример: https://ps.airbapay.kz/acquiring-api
	BaseURL string

	// Куда вернуть пользователя после оплаты (фронт)
	SuccessBackURL string
	FailureBackURL string

	// Куда шлётся вебхук (бэк)
	CallbackURL string

	// Не обязательно, но удобно, если хочешь прокидывать в create
	DefaultEmail     string
	DefaultPhone     string // 11 цифр, 7XXXXXXXXXX
	DefaultAccountID string

	Client *http.Client
	Logger *slog.Logger
}

type AirbapayService struct {
	username   string
	password   string
	terminalID string
	baseURL    *url.URL

	successBackURL string
	failureBackURL string
	callbackURL    string

	defEmail     string
	defPhone     string
	defAccountID string

	httpClient *http.Client
	logger     *slog.Logger

	// jwt cache
	mu          sync.Mutex
	accessToken string
	tokenExp    time.Time

	// публичный ключ для верификации подписи вебхука
	pubKeyOnce sync.Once
	pubKeyErr  error
	pubKey     *rsa.PublicKey
}

func NewAirbapayService(cfg AirbapayConfig) (*AirbapayService, error) {
	if strings.TrimSpace(cfg.Username) == "" ||
		strings.TrimSpace(cfg.Password) == "" ||
		strings.TrimSpace(cfg.TerminalID) == "" ||
		strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("airbapay: username/password/terminal_id/base_url are required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	s := &AirbapayService{
		username:       cfg.Username,
		password:       cfg.Password,
		terminalID:     cfg.TerminalID,
		baseURL:        u,
		successBackURL: cfg.SuccessBackURL,
		failureBackURL: cfg.FailureBackURL,
		callbackURL:    cfg.CallbackURL,
		defEmail:       cfg.DefaultEmail,
		defPhone:       cfg.DefaultPhone,
		defAccountID:   cfg.DefaultAccountID,
		httpClient:     client,
		logger:         logger,
	}
	logger.Info("AirbaPay initialized",
		"baseURL", safeURL(s.baseURL),
		"successBackURL_set", s.successBackURL != "",
		"failureBackURL_set", s.failureBackURL != "",
		"callbackURL_set", s.callbackURL != "",
	)
	return s, nil
}

// ------- AUTH (JWT) -------

func (s *AirbapayService) ensureToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && time.Until(s.tokenExp) > 2*time.Minute {
		return s.accessToken, nil
	}
	type signInReq struct {
		User       string `json:"user"`
		Password   string `json:"password"`
		TerminalID string `json:"terminal_id"`
		// ВАЖНО: НЕ добавлять payment_id сюда при создании платежа!
	}
	type signInResp struct {
		AccessToken string `json:"access_token"`
		// Иногда присылают ttl/exp — если нет, используем 55 минут.
	}

	endpoint := *s.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/api/v1/auth/sign-in")
	body, _ := json.Marshal(signInReq{
		User:       s.username,
		Password:   s.password,
		TerminalID: s.terminalID,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var out signInResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("auth decode: %w", err)
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", errors.New("auth: empty access_token")
	}
	s.accessToken = out.AccessToken
	s.tokenExp = time.Now().Add(55 * time.Minute)
	return s.accessToken, nil
}

// ------- PAYMENTS v2 -------

type paymentV2Request struct {
	InvoiceID       string  `json:"invoice_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"` // "KZT"
	Description     string  `json:"description,omitempty"`
	Email           string  `json:"email,omitempty"`
	Phone           string  `json:"phone,omitempty"`    // 7XXXXXXXXXX
	Language        string  `json:"language,omitempty"` // "ru","kz","en"
	AccountID       string  `json:"account_id,omitempty"`
	CardSave        bool    `json:"card_save"`
	AutoCharge      int     `json:"auto_charge"` // 1=one-stage, 0=two-stage
	SuccessBackURL  string  `json:"success_back_url"`
	FailureBackURL  string  `json:"failure_back_url"`
	SuccessCallback string  `json:"success_callback"`
	FailureCallback string  `json:"failure_callback"`
}

type paymentV2Response struct {
	ID          string `json:"id"`
	InvoiceID   string `json:"invoice_id"`
	Status      string `json:"status"`
	RedirectURL string `json:"redirect_url"`
}

type AirbapayCreateInvoiceResponse struct {
	InvoiceID  string          `json:"invoice_id"`
	OrderID    string          `json:"order_id"`
	PaymentURL string          `json:"payment_url"`
	Status     string          `json:"status"`
	Message    string          `json:"message"`
	Raw        json.RawMessage `json:"-"`
}

// «Сделать платеж»: создаём сущность и получаем redirect_url
func (s *AirbapayService) CreatePaymentLink(ctx context.Context, invoiceID int, amount float64, description string) (*AirbapayCreateInvoiceResponse, error) {
	logger := s.logger.With("op", "CreatePaymentLink")
	token, err := s.ensureToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := *s.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/api/v2/payments")

	reqBody := paymentV2Request{
		InvoiceID:       strconv.Itoa(invoiceID),
		Amount:          amount,
		Currency:        "KZT",
		Description:     description,
		Email:           s.defEmail,
		Phone:           s.defPhone,
		Language:        "ru",
		AccountID:       s.defAccountID,
		CardSave:        true,
		AutoCharge:      1, // двухстадийный по умолчанию
		SuccessBackURL:  s.successBackURL,
		FailureBackURL:  s.failureBackURL,
		SuccessCallback: s.callbackURL,
		FailureCallback: s.callbackURL,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("payments v2 request: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	logger.Debug("payments v2 raw", "status", resp.Status, "body", trim(string(b), 2000))

	if resp.StatusCode != http.StatusCreated {
		return nil, &AirbapayError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(b)}
	}

	var out paymentV2Response
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode payments v2: %w", err)
	}
	if strings.TrimSpace(out.RedirectURL) == "" || strings.TrimSpace(out.ID) == "" {
		return nil, fmt.Errorf("payments v2: empty redirect_url or id")
	}

	// Приводим к прежнему ответу хендлера (чтобы не переписывать много кода)
	return &AirbapayCreateInvoiceResponse{
		InvoiceID:  out.InvoiceID,
		OrderID:    strconv.Itoa(invoiceID),
		PaymentURL: out.RedirectURL,
		Status:     out.Status,
		Raw:        json.RawMessage(b),
	}, nil
}

// ------- CHARGE / RETURN (опционально, если используешь двухстадийный) -------

func (s *AirbapayService) Charge(ctx context.Context, amount float64) error {
	token, err := s.ensureToken(ctx)
	if err != nil {
		return err
	}
	endpoint := *s.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/api/v1/payments/charge")
	body, _ := json.Marshal(map[string]any{"amount": amount})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return &AirbapayError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(b)}
	}
	return nil
}

func (s *AirbapayService) Return(ctx context.Context, extID string, amount *float64) error {
	token, err := s.ensureToken(ctx)
	if err != nil {
		return err
	}
	endpoint := *s.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/api/v1/payments/return")
	body := map[string]any{"ext_id": extID}
	if amount != nil {
		body["amount"] = *amount
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return &AirbapayError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(rb)}
	}
	return nil
}

// ------- CALLBACK (webhook) -------

type WebhookPayload struct {
	ID          string          `json:"id"`
	InvoiceID   string          `json:"invoice_id"`
	Amount      float64         `json:"amount"`
	Currency    string          `json:"currency"`
	Status      string          `json:"status"`
	Description string          `json:"description"`
	Sign        string          `json:"sign"`
	Raw         json.RawMessage `json:"-"`
}

func (p *WebhookPayload) UnmarshalJSON(data []byte) error {
	type rawPayload struct {
		ID             string          `json:"id"`
		InvoiceID      string          `json:"invoice_id"`
		InvoiceIDCamel string          `json:"invoiceId"`
		Amount         json.RawMessage `json:"amount"`
		Currency       string          `json:"currency"`
		Status         string          `json:"status"`
		Description    string          `json:"description"`
		Sign           string          `json:"sign"`
	}

	var raw rawPayload
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	invoiceID := strings.TrimSpace(raw.InvoiceID)
	if invoiceID == "" {
		invoiceID = strings.TrimSpace(raw.InvoiceIDCamel)
	}

	var amount float64
	if len(raw.Amount) > 0 {
		if err := json.Unmarshal(raw.Amount, &amount); err != nil {
			var amountStr string
			if err := json.Unmarshal(raw.Amount, &amountStr); err != nil {
				return fmt.Errorf("airbapay: parse webhook amount: %w", err)
			}
			amountStr = strings.TrimSpace(amountStr)
			if amountStr != "" {
				parsed, err := strconv.ParseFloat(amountStr, 64)
				if err != nil {
					return fmt.Errorf("airbapay: parse webhook amount: %w", err)
				}
				amount = parsed
			}
		}
	}

	p.ID = strings.TrimSpace(raw.ID)
	p.InvoiceID = invoiceID
	p.Amount = amount
	p.Currency = strings.TrimSpace(raw.Currency)
	p.Status = strings.TrimSpace(raw.Status)
	p.Description = strings.TrimSpace(raw.Description)
	p.Sign = strings.TrimSpace(raw.Sign)

	return nil
}

// Порядок конкатенации для подписи: id+invoice_id+amount+currency+status+description
func (s *AirbapayService) ValidateCallbackSignature(p *WebhookPayload) bool {
	if p == nil || strings.TrimSpace(p.Sign) == "" {
		return false
	}
	if err := s.loadPublicKeyOnce(); err != nil {
		s.logger.Error("load public key failed", "err", err)
		return false
	}

	// amount → строка без лишних нулей по правилам платы (приведём к 2 знакам и обрежем)
	amountStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", p.Amount), "0"), ".")
	payload := p.ID + p.InvoiceID + amountStr + p.Currency + p.Status + p.Description

	sig, err := base64.StdEncoding.DecodeString(p.Sign)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(payload))
	if err := rsa.VerifyPKCS1v15(s.pubKey, crypto.SHA256, h[:], sig); err != nil {
		return false
	}
	return true
}

func (s *AirbapayService) loadPublicKeyOnce() error {
	s.pubKeyOnce.Do(func() {
		// официальный ключ AirbaPay (prod):
		// https://ps.airbapay.kz/acquiring/sign/public.pem
		u := "https://ps.airbapay.kz/acquiring/sign/public.pem"
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.pubKeyErr = err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			s.pubKeyErr = fmt.Errorf("get public key: %s", resp.Status)
			return
		}
		b, _ := io.ReadAll(resp.Body)
		block, _ := pem.Decode(b)
		if block == nil {
			s.pubKeyErr = errors.New("pem decode failed")
			return
		}
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			s.pubKeyErr = err
			return
		}
		rk, ok := key.(*rsa.PublicKey)
		if !ok {
			s.pubKeyErr = errors.New("not rsa public key")
			return
		}
		s.pubKey = rk
	})
	return s.pubKeyErr
}

func (s *AirbapayService) ParseCallback(r io.Reader) (*WebhookPayload, error) {
	if s == nil {
		return nil, fmt.Errorf("airbapay service is not initialised")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read callback body: %w", err)
	}
	var p WebhookPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode callback: %w", err)
	}
	p.Raw = json.RawMessage(data)
	return &p, nil
}

// ---------- helpers ----------

func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func safeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	c := *u
	c.User = nil
	return c.String()
}

type AirbapayError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *AirbapayError) Error() string {
	if e == nil {
		return "<nil>"
	}
	bt := strings.TrimSpace(e.Body)
	if bt == "" {
		return fmt.Sprintf("airbapay error: %s", e.Status)
	}
	return fmt.Sprintf("airbapay error: %s: %s", e.Status, bt)
}
