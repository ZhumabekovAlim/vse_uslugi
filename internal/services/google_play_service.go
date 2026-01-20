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

func (s *GooglePlayService) VerifySubscriptionPurchaseV2(ctx context.Context, token string) (models.GooglePurchase, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return models.GooglePurchase{}, errors.New("purchase_token is required")
	}

	// ВАЖНО: subscriptionsv2.get НЕ принимает subscriptionID (sku), только token
	resp, err := s.svc.Purchases.Subscriptionsv2.Get(s.cfg.PackageName, token).
		Context(ctx).
		Do()
	if err != nil {
		return models.GooglePurchase{}, fmt.Errorf("google subscriptionsv2.get: %w", err)
	}

	raw, _ := json.Marshal(resp)

	// Определим "active" максимально безопасно
	// В v2 структура сложнее (lineItems), но минимально:
	// если есть lineItems и у него expiryTime > now => ACTIVE
	nowMillis := time.Now().UnixMilli()

	derivedState := int64(2)
	status := "UNKNOWN"
	orderID := ""

	// resp.LineItems может быть nil/empty
	if len(resp.LineItems) > 0 {
		li := resp.LineItems[0]
		// OrderId может быть в разных местах, но часто есть latestOrderId
		if li.LatestPurchaseId != nil {
			orderID = *li.LatestPurchaseId
		}
		if li.ExpiryTime != "" {
			// expiryTime приходит как RFC3339 (примерно), попробуем распарсить
			if t, perr := time.Parse(time.RFC3339Nano, li.ExpiryTime); perr == nil {
				exp := t.UnixMilli()
				if exp > nowMillis {
					derivedState = 0
					status = "ACTIVE"
				} else {
					derivedState = 1
					status = "EXPIRED"
				}
			}
		}
	}

	return models.GooglePurchase{
		Kind:          "subscription",
		ProductID:     "", // в v2 sku может быть в li.OfferDetails/BasePlanId, можно допарсить позже
		PurchaseToken: token,
		OrderID:       orderID,
		PackageName:   s.cfg.PackageName,
		PurchaseState: derivedState,
		Status:        status,
		Raw:           string(raw),
	}, nil
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
