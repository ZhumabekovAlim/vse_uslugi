package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	ProductTargets      map[string]models.IAPTarget
}

func NewIAPHandler(service *services.AppleIAPService, repo *repositories.IAPRepository, subRepo *repositories.SubscriptionRepository, subService *services.SubscriptionService, topService *services.TopService, businessService *services.BusinessService, productTargets map[string]models.IAPTarget) *IAPHandler {
	return &IAPHandler{
		Service:             service,
		Repo:                repo,
		SubscriptionRepo:    subRepo,
		SubscriptionService: subService,
		TopService:          topService,
		BusinessService:     businessService,
		ProductTargets:      productTargets,
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
	txn, err := h.Service.VerifyTransaction(r.Context(), req.TransactionID)
	if err != nil {
		http.Error(w, "apple verify: "+err.Error(), http.StatusBadGateway)
		return
	}

	target, err := h.resolveTarget(txn.ProductID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	processed, err := h.processTransaction(r.Context(), userID, target, txn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		var renewal models.AppleRenewalInfo
		renewal, err = h.Service.DecodeSignedRenewalInfo(r.Context(), notif.Data.SignedRenewalInfo)
		if err == nil {
			txn = transactionFromRenewal(renewal)
			if strings.TrimSpace(txn.OriginalTransactionID) == "" {
				err = errors.New("renewal info missing original transaction id")
			}
		}
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

	action := classifyNotification(notif.NotificationType, notif.Subtype)
	switch action {
	case notificationIgnore:
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	case notificationRevoke:
		if err := h.Repo.DeleteByTransactionID(r.Context(), txn.TransactionID); err != nil {
			http.Error(w, "revoke transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
		return
	}

	if _, err := h.processTransaction(r.Context(), userID, target, txn); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

func (h *IAPHandler) processTransaction(ctx context.Context, userID int, target models.IAPTarget, txn models.AppleTransaction) (bool, error) {
	if strings.TrimSpace(txn.TransactionID) == "" {
		return false, errors.New("transaction id is required")
	}
	processed, err := h.Repo.IsProcessed(ctx, txn.TransactionID)
	if err != nil {
		return false, fmt.Errorf("idempotency check: %w", err)
	}
	if processed {
		return true, nil
	}

	if err := h.Repo.Save(ctx, txn, userID, target); err != nil {
		return false, fmt.Errorf("store transaction: %w", err)
	}
	if err := h.applyTarget(ctx, userID, target, txn); err != nil {
		if rollbackErr := h.Repo.DeleteByTransactionID(ctx, txn.TransactionID); rollbackErr != nil {
			return false, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return false, err
	}
	return false, nil
}

func transactionFromRenewal(renewal models.AppleRenewalInfo) models.AppleTransaction {
	txnID := renewal.OriginalTransactionID
	if renewal.SignedDate > 0 {
		txnID = fmt.Sprintf("renewal:%s:%d", renewal.OriginalTransactionID, renewal.SignedDate)
	}
	return models.AppleTransaction{
		TransactionID:         txnID,
		OriginalTransactionID: renewal.OriginalTransactionID,
		ProductID:             renewal.AutoRenewProductID,
		BundleID:              renewal.BundleID,
		Environment:           renewal.Environment,
		Raw:                   renewal.Raw,
	}
}

func (h *IAPHandler) resolveTarget(productID string) (models.IAPTarget, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return models.IAPTarget{}, errors.New("product id is empty")
	}
	if len(h.ProductTargets) == 0 {
		return models.IAPTarget{}, errors.New("iap product targets are not configured")
	}
	target, ok := h.ProductTargets[productID]
	if !ok {
		return models.IAPTarget{}, fmt.Errorf("unsupported product id: %s", productID)
	}
	if err := target.Validate(); err != nil {
		return models.IAPTarget{}, err
	}
	return target, nil
}

type notificationAction int

const (
	notificationIgnore notificationAction = iota
	notificationGrant
	notificationRevoke
)

func classifyNotification(notificationType, subtype string) notificationAction {
	switch strings.ToUpper(strings.TrimSpace(notificationType)) {
	case "INITIAL_BUY", "DID_RENEW", "DID_RECOVER", "INTERACTIVE_RENEWAL", "REFUND_REVERSED":
		return notificationGrant
	case "REVOKE", "REFUND":
		return notificationRevoke
	// Ignore state transitions that do not change entitlement (grace, billing retry, etc.).
	case "DID_FAIL_TO_RENEW", "EXPIRED", "GRACE_PERIOD_EXPIRED", "BILLING_RETRY", "PRICE_INCREASE_CONSENT":
		return notificationIgnore
	default:
		// Subtypes like VOLUNTARY or BILLING_RECOVERY can be informative but not actionable here.
		if strings.ToUpper(strings.TrimSpace(subtype)) == "VOLUNTARY" {
			return notificationRevoke
		}
	}
	return notificationIgnore
}
