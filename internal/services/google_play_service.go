package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"naimuBack/internal/models"
)

type GooglePlayConfig struct {
	PackageName        string
	ServiceAccountJSON string
}

type GooglePlayService struct {
	cfg GooglePlayConfig
	svc *androidpublisher.Service
}

func NewGooglePlayService(cfg GooglePlayConfig) (*GooglePlayService, error) {
	cfg.PackageName = strings.TrimSpace(cfg.PackageName)
	if cfg.PackageName == "" {
		return nil, errors.New("GOOGLE_PLAY_PACKAGE_NAME is empty")
	}
	if strings.TrimSpace(cfg.ServiceAccountJSON) == "" {
		return nil, errors.New("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON is empty")
	}

	ctx := context.Background()
	s, err := androidpublisher.NewService(ctx,
		option.WithCredentialsJSON([]byte(cfg.ServiceAccountJSON)),
		option.WithScopes(androidpublisher.AndroidpublisherScope),
	)
	if err != nil {
		return nil, fmt.Errorf("androidpublisher.NewService: %w", err)
	}

	return &GooglePlayService{cfg: cfg, svc: s}, nil
}

func (s *GooglePlayService) VerifyProductPurchase(ctx context.Context, productID, token string) (models.GooglePurchase, error) {
	productID = strings.TrimSpace(productID)
	token = strings.TrimSpace(token)
	if productID == "" || token == "" {
		return models.GooglePurchase{}, errors.New("product_id and purchase_token are required")
	}

	resp, err := s.svc.Purchases.Products.Get(s.cfg.PackageName, productID, token).
		Context(ctx).
		Do()
	if err != nil {
		return models.GooglePurchase{}, fmt.Errorf("google products.get: %w", err)
	}

	raw, _ := json.Marshal(resp)

	return models.GooglePurchase{
		Kind:          "product",
		ProductID:     productID,
		PurchaseToken: token,
		OrderID:       resp.OrderId,
		PackageName:   s.cfg.PackageName,

		PurchaseState: resp.PurchaseState,
		Acknowledged:  resp.AcknowledgementState == 1,
		Consumed:      resp.ConsumptionState == 1,

		Raw: string(raw),
	}, nil
}

func (s *GooglePlayService) VerifySubscriptionPurchase(ctx context.Context, subscriptionID, token string) (models.GooglePurchase, error) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	token = strings.TrimSpace(token)
	if subscriptionID == "" || token == "" {
		return models.GooglePurchase{}, errors.New("subscription_id and purchase_token are required")
	}

	resp, err := s.svc.Purchases.Subscriptions.Get(s.cfg.PackageName, subscriptionID, token).
		Context(ctx).
		Do()
	if err != nil {
		return models.GooglePurchase{}, fmt.Errorf("google subscriptions.get: %w", err)
	}

	raw, _ := json.Marshal(resp)

	nowMillis := time.Now().UnixMilli()

	derivedState := int64(2)
	status := "UNKNOWN"

	// PaymentState: 0 pending, 1 received, 2 free trial, 3 deferred
	if int64PtrEq(resp.PaymentState, 0) {
		status = "PENDING"
		derivedState = 2
	} else if resp.ExpiryTimeMillis > 0 && resp.ExpiryTimeMillis > nowMillis {
		derivedState = 0
		status = "ACTIVE"

		// Автопродление выключено => "canceled", но период может быть еще активен
		if !resp.AutoRenewing {
			status = "CANCELED"
		}
	} else if resp.ExpiryTimeMillis > 0 && resp.ExpiryTimeMillis <= nowMillis {
		status = "EXPIRED"
		derivedState = 1
	}

	p := models.GooglePurchase{
		Kind:          "subscription",
		ProductID:     subscriptionID,
		PurchaseToken: token,
		OrderID:       resp.OrderId,
		PackageName:   s.cfg.PackageName,

		ExpiryTimeMillis: resp.ExpiryTimeMillis,
		PaymentState:     resp.PaymentState, // <-- теперь тип совпадает
		CancelReason:     resp.CancelReason,
		AutoRenewing:     resp.AutoRenewing,

		PurchaseState: derivedState,
		Acknowledged:  resp.AcknowledgementState == 1,
		Status:        status,
		Raw:           string(raw),
	}

	return p, nil
}

func (s *GooglePlayService) AcknowledgeProduct(ctx context.Context, productID, token string) error {
	productID = strings.TrimSpace(productID)
	token = strings.TrimSpace(token)
	if productID == "" || token == "" {
		return errors.New("product_id and purchase_token are required")
	}

	req := &androidpublisher.ProductPurchasesAcknowledgeRequest{}
	if err := s.svc.Purchases.Products.Acknowledge(s.cfg.PackageName, productID, token, req).
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("google products.acknowledge: %w", err)
	}
	return nil
}

func (s *GooglePlayService) ConsumeProduct(ctx context.Context, productID, token string) error {
	productID = strings.TrimSpace(productID)
	token = strings.TrimSpace(token)
	if productID == "" || token == "" {
		return errors.New("product_id and purchase_token are required")
	}

	if err := s.svc.Purchases.Products.Consume(s.cfg.PackageName, productID, token).
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("google products.consume: %w", err)
	}
	return nil
}

func (s *GooglePlayService) AcknowledgeSubscription(ctx context.Context, subscriptionID, token string) error {
	subscriptionID = strings.TrimSpace(subscriptionID)
	token = strings.TrimSpace(token)
	if subscriptionID == "" || token == "" {
		return errors.New("subscription_id and purchase_token are required")
	}

	req := &androidpublisher.SubscriptionPurchasesAcknowledgeRequest{}
	if err := s.svc.Purchases.Subscriptions.Acknowledge(s.cfg.PackageName, subscriptionID, token, req).
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("google subscriptions.acknowledge: %w", err)
	}
	return nil
}

func int64PtrEq(v *int64, want int64) bool {
	return v != nil && *v == want
}
