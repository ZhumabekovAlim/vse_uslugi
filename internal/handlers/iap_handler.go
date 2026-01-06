package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

// IAPHandler orchestrates Apple IAP verification and entitlement application.
type IAPHandler struct {
	Service             *services.AppleIAPService
	Repo                *repositories.IAPRepository
	SubscriptionRepo    *repositories.SubscriptionRepository
	SubscriptionService *services.SubscriptionService
	TopService          *services.TopService
	BusinessService     *services.BusinessService
}

func NewIAPHandler(service *services.AppleIAPService, repo *repositories.IAPRepository, subRepo *repositories.SubscriptionRepository, subService *services.SubscriptionService, topService *services.TopService, businessService *services.BusinessService) *IAPHandler {
	return &IAPHandler{
		Service:             service,
		Repo:                repo,
		SubscriptionRepo:    subRepo,
		SubscriptionService: subService,
		TopService:          topService,
		BusinessService:     businessService,
	}
}

// VerifyIOSPurchase handles client verification requests after StoreKit completes a purchase.
func (h *IAPHandler) VerifyIOSPurchase(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Repo == nil {
		http.Error(w, "iap is not configured", http.StatusNotImplemented)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		TransactionID string           `json:"transaction_id"`
		Target        models.IAPTarget `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.Target.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	txn, err := h.Service.VerifyTransaction(r.Context(), req.TransactionID)
	if err != nil {
		http.Error(w, "apple verify: "+err.Error(), http.StatusBadGateway)
		return
	}

	processed, err := h.Repo.IsProcessed(r.Context(), txn.TransactionID)
	if err != nil {
		http.Error(w, "idempotency check: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !processed {
		if err := h.applyTarget(r.Context(), userID, req.Target, txn); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.Repo.Save(r.Context(), txn, userID, req.Target); err != nil {
			http.Error(w, "store transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp := map[string]any{
		"status":                  "ok",
		"transaction_id":          txn.TransactionID,
		"original_transaction_id": txn.OriginalTransactionID,
		"environment":             txn.Environment,
		"already_processed":       processed,
		"entitlements":            h.entitlements(r, userID),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// AppleNotificationsV2 handles server-to-server notifications from Apple.
func (h *IAPHandler) AppleNotificationsV2(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Repo == nil {
		http.Error(w, "iap is not configured", http.StatusNotImplemented)
		return
	}

	var req struct {
		SignedPayload string `json:"signedPayload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	notif, err := h.Service.ParseNotification(r.Context(), req.SignedPayload)
	if err != nil {
		http.Error(w, "verify notification: "+err.Error(), http.StatusBadRequest)
		return
	}

	var txn models.AppleTransaction
	switch {
	case notif.Data.SignedTransactionInfo != "":
		txn, err = h.Service.DecodeSignedTransaction(r.Context(), notif.Data.SignedTransactionInfo)
	case notif.Data.SignedRenewalInfo != "":
		txn, err = h.Service.DecodeSignedTransaction(r.Context(), notif.Data.SignedRenewalInfo)
	default:
		err = errors.New("notification missing transaction info")
	}
	if err != nil {
		http.Error(w, "decode transaction: "+err.Error(), http.StatusBadRequest)
		return
	}

	target, userID, err := h.Repo.FindByOriginalTransactionID(r.Context(), txn.OriginalTransactionID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		http.Error(w, "load original transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if userID == 0 {
		http.Error(w, "transaction owner missing", http.StatusInternalServerError)
		return
	}

	processed, err := h.Repo.IsProcessed(r.Context(), txn.TransactionID)
	if err != nil {
		http.Error(w, "idempotency check: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !processed {
		if err := h.applyTarget(r.Context(), userID, target, txn); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.Repo.Save(r.Context(), txn, userID, target); err != nil {
			http.Error(w, "store transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetEntitlements returns the current entitlements for the authenticated user.
func (h *IAPHandler) GetEntitlements(w http.ResponseWriter, r *http.Request) {
	if h.SubscriptionService == nil {
		http.Error(w, "subscriptions not configured", http.StatusNotImplemented)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entitlements := h.entitlements(r, userID)
	if entitlements == nil {
		http.Error(w, "failed to load entitlements", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(entitlements)
}

func (h *IAPHandler) entitlements(r *http.Request, userID int) *models.SubscriptionProfile {
	if h.SubscriptionService == nil {
		return nil
	}
	profile, err := h.SubscriptionService.GetProfile(r.Context(), userID)
	if err != nil {
		fmt.Println("iap: load entitlements:", err)
		return nil
	}
	return &profile
}

func (h *IAPHandler) applyTarget(ctx context.Context, userID int, target models.IAPTarget, txn models.AppleTransaction) error {
	switch target.Type {
	case models.IAPTargetTypeResponses:
		if target.Quantity <= 0 {
			target.Quantity = 10
		}
		if h.SubscriptionRepo == nil {
			return errors.New("subscription repo is not configured")
		}
		return h.SubscriptionRepo.AddResponsesBalance(ctx, userID, target.Quantity)
	case models.IAPTargetTypeSubscription:
		if h.SubscriptionRepo == nil {
			return errors.New("subscription repo is not configured")
		}
		subType, err := models.ParseSubscriptionType(target.SubscriptionType)
		if err != nil {
			return err
		}
		if _, err := h.SubscriptionRepo.ExtendSubscription(ctx, userID, subType, target.Months); err != nil {
			return err
		}
		return nil
	case models.IAPTargetTypeTop:
		if h.TopService == nil {
			return errors.New("top service not configured")
		}
		req := models.TopActivationRequest{
			Type:         target.ListingType,
			ID:           int(target.ID),
			DurationDays: target.DurationDays,
		}
		_, err := h.TopService.ActivateTop(ctx, userID, req)
		return err
	case models.IAPTargetTypeBusiness:
		if h.BusinessService == nil {
			return errors.New("business service not configured")
		}
		provider := "apple_iap"
		state := "paid"
		req := services.PurchaseRequest{
			Seats:         target.Seats,
			Provider:      &provider,
			ProviderTxnID: &txn.TransactionID,
			State:         &state,
		}
		_, err := h.BusinessService.PurchaseSeats(ctx, userID, req)
		return err
	default:
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
}
