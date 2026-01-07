package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"naimuBack/internal/courier"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
	"naimuBack/internal/taxi"
)

const (
	productUnknown             = ""
	productResponses           = "responses"
	invoiceTargetTaxiBalance   = "taxi_driver_balance"
	invoiceTargetCourierFunds  = "courier_balance"
	invoiceTargetSubscription  = "subscription_purchase"
	invoiceTargetTopActivation = "top_activation"
	invoiceTargetBusiness      = "business_purchase"
)

type AirbapayHandler struct {
	Service          *services.AirbapayService
	InvoiceRepo      *repositories.InvoiceRepo
	SubscriptionRepo *repositories.SubscriptionRepository
	TopService       *services.TopService
	BusinessService  *services.BusinessService
	TaxiMux          http.Handler
	TaxiDeps         *taxi.TaxiDeps
	CourierDeps      *courier.Deps
}

func NewAirbapayHandler(s *services.AirbapayService, r *repositories.InvoiceRepo, sub *repositories.SubscriptionRepository) *AirbapayHandler {
	return &AirbapayHandler{Service: s, InvoiceRepo: r, SubscriptionRepo: sub}
}

// SetTaxiWebhookHandler wires taxi webhook handler so that callbacks with HMAC signature are delegated to the taxi module.
func (h *AirbapayHandler) SetTaxiWebhookHandler(handler http.Handler) {
	h.TaxiMux = handler
}

// SetTaxiDeps injects taxi dependencies for balance operations.
func (h *AirbapayHandler) SetTaxiDeps(deps *taxi.TaxiDeps) {
	h.TaxiDeps = deps
}

// SetCourierDeps injects courier dependencies for balance operations.
func (h *AirbapayHandler) SetCourierDeps(deps *courier.Deps) {
	h.CourierDeps = deps
}

func (h *AirbapayHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		UserID      int     `json:"user_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		Target      *struct {
			Type   string `json:"type"`
			ID     int64  `json:"id"`
			Amount *int   `json:"amount,omitempty"`
		} `json:"target,omitempty"`
		Subscription *struct {
			Type   string `json:"type"`
			Months int    `json:"months"`
		} `json:"subscription,omitempty"`
		Business *struct {
			Seats int `json:"seats"`
		} `json:"business,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Target != nil {
		switch req.Target.Type {
		case invoiceTargetTaxiBalance, invoiceTargetCourierFunds:
			if req.Target.ID <= 0 {
				http.Error(w, "target id must be positive", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unsupported target type", http.StatusBadRequest)
			return
		}
	}

	var subscriptionTarget json.RawMessage
	if req.Subscription != nil {
		subType, err := models.ParseSubscriptionType(req.Subscription.Type)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Subscription.Months <= 0 {
			http.Error(w, "subscription months must be positive", http.StatusBadRequest)
			return
		}
		payload, _ := json.Marshal(map[string]any{
			"type":   subType,
			"months": req.Subscription.Months,
		})
		subscriptionTarget = payload
		if strings.TrimSpace(req.Description) == "" {
			req.Description = fmt.Sprintf("subscription %s x%d", subType, req.Subscription.Months)
		}
	}

	if req.Business != nil {
		if req.Business.Seats <= 0 {
			http.Error(w, "business seats must be positive", http.StatusBadRequest)
			return
		}
		if req.Amount <= 0 {
			http.Error(w, "amount must be positive", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Description) == "" {
			req.Description = "Бизнес аккаунт"
		}
	}

	invID, err := h.InvoiceRepo.CreateInvoice(r.Context(), req.UserID, req.Amount, req.Description)
	if err != nil {
		http.Error(w, "create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Target != nil {
		payload := json.RawMessage(nil)
		if req.Target.Amount != nil {
			b, _ := json.Marshal(map[string]int{"amount": *req.Target.Amount})
			payload = b
		}
		if _, err := h.InvoiceRepo.AddTarget(r.Context(), invID, req.Target.Type, req.Target.ID, payload); err != nil {
			http.Error(w, "store invoice target: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if subscriptionTarget != nil {
		if _, err := h.InvoiceRepo.AddTarget(r.Context(), invID, invoiceTargetSubscription, int64(req.UserID), subscriptionTarget); err != nil {
			http.Error(w, "store invoice target: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if req.Business != nil {
		businessPayload, _ := json.Marshal(map[string]any{
			"seats": req.Business.Seats,
		})
		if _, err := h.InvoiceRepo.AddTarget(r.Context(), invID, invoiceTargetBusiness, int64(req.UserID), businessPayload); err != nil {
			http.Error(w, "store invoice target: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp, err := h.Service.CreatePaymentLink(r.Context(), invID, req.Amount, req.Description)
	if err != nil {
		_ = h.InvoiceRepo.UpdateStatus(r.Context(), invID, "error")
		http.Error(w, "create payment link: "+err.Error(), airbapayErrorStatus(err))
		return
	}

	if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, strings.ToLower(resp.Status)); err != nil {
		fmt.Println("airbapay: failed to update invoice status:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"inv_id":      invID,
		"order_id":    resp.OrderID,
		"invoice_id":  resp.InvoiceID,
		"payment_url": resp.PaymentURL,
		"status":      resp.Status,
	})
}

func airbapayErrorStatus(err error) int {
	var apiErr *services.AirbapayError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return apiErr.StatusCode
		}
	}
	return http.StatusBadGateway
}

func (h *AirbapayHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if sig := strings.TrimSpace(r.Header.Get("X-AirbaPay-Signature")); sig != "" {
		if h.TaxiMux == nil {
			http.Error(w, "taxi webhook handler not configured", http.StatusNotImplemented)
			return
		}
		req := r.Clone(r.Context())
		req.Body = io.NopCloser(bytes.NewReader(body))
		h.TaxiMux.ServeHTTP(w, req)
		return
	}

	payload, err := h.Service.ParseCallback(bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.InvoiceID) == "" {
		http.Error(w, "missing invoice_id", http.StatusBadRequest)
		return
	}
	if !h.Service.ValidateCallbackSignature(payload) {
		http.Error(w, "invalid callback signature", http.StatusBadRequest)
		return
	}

	invID, err := strconv.Atoi(payload.InvoiceID)
	if err != nil {
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}

	status := strings.ToLower(payload.Status)
	switch status {
	case "success", "succeeded", "paid", "done", "approved", "auth":
		invoice, err := h.InvoiceRepo.GetByID(r.Context(), invID)
		if err != nil {
			http.Error(w, "get invoice: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.InvoiceRepo.MarkPaid(r.Context(), invID); err != nil {
			http.Error(w, "mark paid: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.applyInvoiceReward(r.Context(), invoice); err != nil {
			fmt.Println("airbapay: apply reward failed:", err)
		}
		if err := h.processInvoiceTargets(r.Context(), invoice, payload); err != nil {
			fmt.Println("airbapay: process invoice targets failed:", err)
		}
	case "failure", "failed", "cancelled", "rejected", "error":
		if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, "failed"); err != nil {
			http.Error(w, "mark failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, status); err != nil {
			http.Error(w, "update status: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"invoice_id": payload.InvoiceID,
	})
}

func (h *AirbapayHandler) applyInvoiceReward(ctx context.Context, invoice models.Invoice) error {
	if h.SubscriptionRepo == nil {
		return nil
	}

	product, quantity := classifyInvoiceDescription(invoice.Description)
	switch product {
	case productResponses:
		if quantity == 0 {
			quantity = 10
		}
		return h.SubscriptionRepo.AddResponsesBalance(ctx, invoice.UserID, quantity)
	default:
		return nil
	}
}

func (h *AirbapayHandler) processInvoiceTargets(ctx context.Context, invoice models.Invoice, payload *services.WebhookPayload) error {
	targets, err := h.InvoiceRepo.ListTargets(ctx, invoice.ID)
	if err != nil {
		return err
	}

	var firstErr error
	for _, target := range targets {
		if target.ProcessedAt != nil {
			continue
		}

		if err := h.executeInvoiceTarget(ctx, invoice, target, payload); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			fmt.Println("airbapay: execute target failed:", err)
			continue
		}

		if err := h.InvoiceRepo.MarkTargetProcessed(ctx, target.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			fmt.Println("airbapay: mark target processed failed:", err)
		}
	}

	return firstErr
}

func (h *AirbapayHandler) executeInvoiceTarget(ctx context.Context, invoice models.Invoice, target models.InvoiceTarget, payload *services.WebhookPayload) error {
	amount := invoiceTargetAmount(invoice.Amount, target.Payload)
	if amount <= 0 {
		return fmt.Errorf("invalid target amount")
	}

	switch target.TargetType {
	case invoiceTargetTaxiBalance:
		if h.TaxiDeps == nil {
			return fmt.Errorf("taxi deps are not configured")
		}
		if err := taxi.DepositDriverBalance(ctx, h.TaxiDeps, target.TargetID, amount); err != nil {
			return fmt.Errorf("deposit taxi balance: %w", err)
		}
	case invoiceTargetCourierFunds:
		if h.CourierDeps == nil {
			return fmt.Errorf("courier deps are not configured")
		}
		if err := courier.DepositBalance(ctx, h.CourierDeps, target.TargetID, amount); err != nil {
			return fmt.Errorf("deposit courier balance: %w", err)
		}
	case invoiceTargetSubscription:
		if h.SubscriptionRepo == nil {
			return fmt.Errorf("subscription repo not configured")
		}
		var data struct {
			Type   string `json:"type"`
			Months int    `json:"months"`
		}
		if err := json.Unmarshal(target.Payload, &data); err != nil {
			return fmt.Errorf("subscription payload: %w", err)
		}
		subType, err := models.ParseSubscriptionType(data.Type)
		if err != nil {
			return err
		}
		if data.Months <= 0 {
			return fmt.Errorf("subscription months must be positive")
		}
		if _, err := h.SubscriptionRepo.ExtendSubscription(ctx, invoice.UserID, subType, data.Months); err != nil {
			return fmt.Errorf("extend subscription: %w", err)
		}
	case invoiceTargetTopActivation:
		if h.TopService == nil {
			return fmt.Errorf("top service not configured")
		}
		var data models.TopActivationRequest
		if err := json.Unmarshal(target.Payload, &data); err != nil {
			return fmt.Errorf("top activation payload: %w", err)
		}
		if _, err := h.TopService.ActivateTop(ctx, invoice.UserID, data); err != nil {
			return fmt.Errorf("activate top: %w", err)
		}
	case invoiceTargetBusiness:
		if h.BusinessService == nil {
			return fmt.Errorf("business service not configured")
		}
		var data struct {
			Seats        int  `json:"seats"`
			DurationDays *int `json:"duration_days,omitempty"`
		}
		if err := json.Unmarshal(target.Payload, &data); err != nil {
			return fmt.Errorf("business payload: %w", err)
		}
		if data.Seats <= 0 {
			return fmt.Errorf("business payload: seats must be positive")
		}
		duration := services.DefaultBusinessSeatDuration()
		if data.DurationDays != nil {
			if *data.DurationDays <= 0 {
				return fmt.Errorf("business payload: duration_days must be positive")
			}
			duration = *data.DurationDays
		}
		provider := "airbapay"
		state := "paid"
		req := services.PurchaseRequest{ //nolint:exhaustruct
			Seats:         data.Seats,
			DurationDays:  &duration,
			Provider:      &provider,
			ProviderTxnID: nil,
			State:         &state,
			Amount:        &invoice.Amount,
		}
		if payload != nil {
			req.ProviderTxnID = &payload.ID
			req.State = &payload.Status
			req.Payload = payload.Raw
		}
		if _, err := h.BusinessService.PurchaseSeats(ctx, invoice.UserID, req); err != nil {
			return fmt.Errorf("activate business: %w", err)
		}
	default:
		// Unknown target types are ignored to keep backward compatibility.
		return nil
	}
	return nil
}

func invoiceTargetAmount(invoiceAmount float64, payload json.RawMessage) int {
	if len(payload) > 0 {
		var data struct {
			Amount *int `json:"amount"`
		}
		if err := json.Unmarshal(payload, &data); err == nil && data.Amount != nil {
			return *data.Amount
		}
	}
	return int(math.Round(invoiceAmount))
}

func classifyInvoiceDescription(description string) (string, int) {
	normalized := strings.ToLower(strings.TrimSpace(description))
	if normalized == "" {
		return productUnknown, 0
	}

	if strings.Contains(normalized, "10") && (strings.Contains(normalized, "отклик") || strings.Contains(normalized, "жауап")) {
		return productResponses, 10
	}

	return productUnknown, 0
}

func (h *AirbapayHandler) SuccessRedirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *AirbapayHandler) FailureRedirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "failure"})
}

func (h *AirbapayHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	invoices, err := h.InvoiceRepo.GetByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "get invoices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invoices)
}
