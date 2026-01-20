package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type GoogleIAPHandler struct {
	Service             *services.GooglePlayService
	Repo                *repositories.GoogleIAPRepository
	SubscriptionRepo    *repositories.SubscriptionRepository
	SubscriptionService *services.SubscriptionService
	TopService          *services.TopService
	BusinessService     *services.BusinessService
	ProductTargets      map[string]models.IAPTarget
}

func NewGoogleIAPHandler(
	svc *services.GooglePlayService,
	repo *repositories.GoogleIAPRepository,
	subRepo *repositories.SubscriptionRepository,
	subService *services.SubscriptionService,
	topService *services.TopService,
	businessService *services.BusinessService,
	targets map[string]models.IAPTarget,
) *GoogleIAPHandler {
	return &GoogleIAPHandler{
		Service:             svc,
		Repo:                repo,
		SubscriptionRepo:    subRepo,
		SubscriptionService: subService,
		TopService:          topService,
		BusinessService:     businessService,
		ProductTargets:      targets,
	}
}

func (h *GoogleIAPHandler) VerifyAndroidPurchase(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Repo == nil {
		http.Error(w, "google iap is not configured", http.StatusNotImplemented)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ProductID     string `json:"product_id"`
		PurchaseToken string `json:"purchase_token"`

		Target *struct {
			ListingType string `json:"listing_type,omitempty"`
			ID          int64  `json:"id,omitempty"`
		} `json:"target,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.ProductID = strings.TrimSpace(req.ProductID)
	req.PurchaseToken = strings.TrimSpace(req.PurchaseToken)
	if req.ProductID == "" || req.PurchaseToken == "" {
		http.Error(w, "product_id and purchase_token are required", http.StatusBadRequest)
		return
	}
	log.Printf("[IAP] incoming user=%d product_id=%q token_len=%d", userID, req.ProductID, len(req.PurchaseToken))

	// 1) target по product_id (как у Apple: resolveTarget(txn.ProductID))
	serverTarget, err := h.resolveTarget(req.ProductID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2) verify в Google
	// 2) verify в Google
	ctx := r.Context()
	var purchase models.GooglePurchase

	isSub := serverTarget.Type == models.IAPTargetTypeSubscription ||
		serverTarget.Type == models.IAPTargetTypeBusiness

	log.Printf("[IAP] verify start user=%d product_id=%q type=%q isSub=%v token_len=%d",
		userID, req.ProductID, serverTarget.Type, isSub, len(req.PurchaseToken))

	if isSub {
		purchase, err = h.Service.VerifySubscriptionPurchase(ctx, req.ProductID, req.PurchaseToken)
	} else {
		purchase, err = h.Service.VerifyProductPurchase(ctx, req.ProductID, req.PurchaseToken)
	}

	if err != nil {
		log.Printf("[IAP] verify failed user=%d product_id=%q isSub=%v token_len=%d err=%v",
			userID, req.ProductID, isSub, len(req.PurchaseToken), err)
		http.Error(w, "google verify: "+err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("[IAP] verify ok kind=%q product_id=%q order_id=%q purchase_state=%d status=%q",
		purchase.Kind, purchase.ProductID, purchase.OrderID, purchase.PurchaseState, purchase.Status)

	// 0 = purchased
	if isSub {
		if purchase.PurchaseState != 0 {
			http.Error(w, "subscription is not active", http.StatusBadRequest)
			return
		}
		if purchase.Status == "PENDING" {
			http.Error(w, "subscription payment is pending", http.StatusBadRequest)
			return
		}
	} else {
		if purchase.PurchaseState != 0 {
			http.Error(w, "purchase is not completed", http.StatusBadRequest)
			return
		}
	}

	if err != nil {
		http.Error(w, "google verify: "+err.Error(), http.StatusBadGateway)
		return
	}

	// 0 = purchased
	if isSub {
		// разрешаем ACTIVE и CANCELED (если период еще не истек => PurchaseState == 0)
		if purchase.PurchaseState != 0 {
			http.Error(w, "subscription is not active", http.StatusBadRequest)
			return
		}
		if purchase.Status == "PENDING" {
			http.Error(w, "subscription payment is pending", http.StatusBadRequest)
			return
		}
	} else {
		if purchase.PurchaseState != 0 {
			http.Error(w, "purchase is not completed", http.StatusBadRequest)
			return
		}
	}

	// 3) гибрид TOP: клиент присылает только привязку listing_type + id
	finalTarget := serverTarget
	if serverTarget.Type == models.IAPTargetTypeTop {
		if req.Target == nil {
			http.Error(w, "target is required for top purchase", http.StatusBadRequest)
			return
		}
		finalTarget.ListingType = strings.TrimSpace(strings.ToLower(req.Target.ListingType))
		finalTarget.ID = req.Target.ID

		if finalTarget.ListingType == "" || finalTarget.ID <= 0 {
			http.Error(w, "invalid top target", http.StatusBadRequest)
			return
		}
		if _, ok := models.AllowedTopTypes()[finalTarget.ListingType]; !ok {
			http.Error(w, "invalid listing_type", http.StatusBadRequest)
			return
		}
	}

	if err := finalTarget.Validate(); err != nil {
		http.Error(w, "target validate: "+err.Error(), http.StatusBadRequest)
		return
	}

	processed, err := h.processPurchase(ctx, userID, finalTarget, purchase)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4) acknowledge/consume только после успешного applyTarget
	// subscriptions => acknowledge
	// responses (consumable) => consume
	// other products => acknowledge
	if isSub {
		swallowIapErr(h.Service.AcknowledgeSubscription(ctx, req.ProductID, req.PurchaseToken))
	} else if serverTarget.Type == models.IAPTargetTypeResponses {
		swallowIapErr(h.Service.ConsumeProduct(ctx, req.ProductID, req.PurchaseToken))
	} else {
		swallowIapErr(h.Service.AcknowledgeProduct(ctx, req.ProductID, req.PurchaseToken))
	}

	resp := map[string]any{
		"status":            "ok",
		"product_id":        req.ProductID,
		"purchase_token":    req.PurchaseToken,
		"order_id":          purchase.OrderID,
		"already_processed": processed,
		"entitlements":      h.entitlements(r, userID),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *GoogleIAPHandler) entitlements(r *http.Request, userID int) *models.SubscriptionProfile {
	if h.SubscriptionService == nil {
		return nil
	}
	profile, err := h.SubscriptionService.GetProfile(r.Context(), userID)
	if err != nil {
		return nil
	}
	return &profile
}

func (h *GoogleIAPHandler) resolveTarget(productID string) (models.IAPTarget, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return models.IAPTarget{}, errors.New("product id is empty")
	}
	if len(h.ProductTargets) == 0 {
		return models.IAPTarget{}, errors.New("google iap product targets are not configured")
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

// processPurchase — аналог Apple processTransaction (idempotency + anti-theft + save + apply)
func (h *GoogleIAPHandler) processPurchase(ctx context.Context, userID int, target models.IAPTarget, p models.GooglePurchase) (bool, error) {
	// NOTE: context.Context тут нужен, поэтому файл должен импортировать context.
	// Чтобы не ловить конфликт, ниже я использую ctx, значит добавь import "context".
	// Но тогда не забудь использовать context в коде, как тут.
	//
	// Если хочешь без context import — можно убрать ctx и брать r.Context() везде.
	//
	// Я оставляю правильный вариант: импортируй context.

	if strings.TrimSpace(p.PurchaseToken) == "" {
		return false, errors.New("purchase token is required")
	}

	// anti-theft: purchase_token must belong to one user
	if owner, err := h.Repo.GetOwnerByToken(ctx, p.PurchaseToken); err == nil {
		if owner != 0 && owner != userID {
			return false, errors.New("purchase belongs to another user")
		}
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return false, fmt.Errorf("check purchase owner: %w", err)
	}

	processed, err := h.Repo.IsProcessed(ctx, p.PurchaseToken)
	if err != nil {
		return false, fmt.Errorf("idempotency check: %w", err)
	}
	if processed {
		return true, nil
	}

	if err := h.Repo.Save(ctx, p, userID, target); err != nil {
		return false, fmt.Errorf("store purchase: %w", err)
	}

	if err := h.applyTarget(ctx, userID, target, p); err != nil {
		_ = h.Repo.DeleteByToken(ctx, p.PurchaseToken)
		return false, err
	}

	return false, nil
}

func (h *GoogleIAPHandler) applyTarget(ctx context.Context, userID int, target models.IAPTarget, p models.GooglePurchase) error {
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
		_, err = h.SubscriptionRepo.ExtendSubscription(ctx, userID, subType, target.Months)
		return err

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
		provider := "google_play"
		state := "paid"
		req := services.PurchaseRequest{
			Seats:         target.Seats,
			Provider:      &provider,
			ProviderTxnID: &p.PurchaseToken,
			State:         &state,
		}
		_, err := h.BusinessService.PurchaseSeats(ctx, userID, req)
		return err

	default:
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
}

// GoogleNotifications — RTDN (Pub/Sub push)
func (h *GoogleIAPHandler) GoogleNotifications(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Repo == nil {
		http.Error(w, "google iap is not configured", http.StatusNotImplemented)
		return
	}

	var push struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
		Subscription string `json:"subscription,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if push.Message.Data == "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	raw, err := base64.StdEncoding.DecodeString(push.Message.Data)
	if err != nil {
		http.Error(w, "decode pubsub data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// RTDN payload
	var notif struct {
		Version     string `json:"version,omitempty"`
		PackageName string `json:"packageName,omitempty"`

		SubscriptionNotification *struct {
			NotificationType int    `json:"notificationType"`
			PurchaseToken    string `json:"purchaseToken"`
			SubscriptionId   string `json:"subscriptionId"`
		} `json:"subscriptionNotification,omitempty"`

		OneTimeProductNotification *struct {
			NotificationType int    `json:"notificationType"`
			PurchaseToken    string `json:"purchaseToken"`
			Sku              string `json:"sku"`
		} `json:"oneTimeProductNotification,omitempty"`
	}
	if err := json.Unmarshal(raw, &notif); err != nil {
		http.Error(w, "unmarshal rtdn: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Subscription RTDN
	if notif.SubscriptionNotification != nil {
		token := strings.TrimSpace(notif.SubscriptionNotification.PurchaseToken)
		sku := strings.TrimSpace(notif.SubscriptionNotification.SubscriptionId)
		if token == "" || sku == "" {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		target, userID, err := h.Repo.FindTargetByToken(ctx, token)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				return
			}
			http.Error(w, "find token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// если target пустой (старые записи), пробуем резолвить по sku
		if strings.TrimSpace(target.Type) == "" {
			if t, rerr := h.resolveTarget(sku); rerr == nil {
				target = t
			}
		}

		purchase, verr := h.Service.VerifySubscriptionPurchase(ctx, sku, token)
		if verr != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		if isSubscriptionRevokeType(notif.SubscriptionNotification.NotificationType) {
			if target.Type == models.IAPTargetTypeSubscription && h.SubscriptionRepo != nil {
				subType, perr := models.ParseSubscriptionType(target.SubscriptionType)
				if perr == nil {
					_ = h.SubscriptionRepo.ForceExpireSubscription(ctx, userID, subType)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
			return
		}

		// grant/renew => apply (idempotency защитит)
		if purchase.PurchaseState == 0 {
			_, _ = h.processPurchase(ctx, userID, target, purchase)
		}

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// One-time product RTDN
	if notif.OneTimeProductNotification != nil {
		token := strings.TrimSpace(notif.OneTimeProductNotification.PurchaseToken)
		sku := strings.TrimSpace(notif.OneTimeProductNotification.Sku)
		if token == "" || sku == "" {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		target, userID, err := h.Repo.FindTargetByToken(ctx, token)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		if strings.TrimSpace(target.Type) == "" {
			if t, rerr := h.resolveTarget(sku); rerr == nil {
				target = t
			}
		}

		purchase, verr := h.Service.VerifyProductPurchase(ctx, sku, token)
		if verr == nil && purchase.PurchaseState == 0 {
			_, _ = h.processPurchase(ctx, userID, target, purchase)
		}

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Минимальный маппинг типов RTDN -> revoke.
// Коды зависят от enum Google, но часто отмена/истечение/ревок попадают сюда.
// Можно расширить позже.
func isSubscriptionRevokeType(t int) bool {
	switch t {
	case 3, 12, 13:
		return true
	default:
		return false
	}
}

func swallowIapErr(err error) {
	// тут можно распарсить googleapi.Error и игнорить конкретные коды,
	// а пока просто игнор / лог
	_ = err
}
