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

	resp, err := s.svc.Purchases.Subscriptionsv2.Get(s.cfg.PackageName, token).
		Context(ctx).
		Do()
	if err != nil {
		return models.GooglePurchase{}, fmt.Errorf("google subscriptionsv2.get: %w", err)
	}

	raw, _ := json.Marshal(resp)
	fmt.Printf("[IAP] subsV2 raw=%s\n", string(raw))
	// Парсим только нужные json-поля (через raw, чтобы не зависеть от SDK-структур)
	type v2LineItem struct {
		ProductID               string `json:"productId"`
		ExpiryTime              string `json:"expiryTime"`
		LatestSuccessfulOrderID string `json:"latestSuccessfulOrderId"`
	}
	type v2Payload struct {
		SubscriptionState    string       `json:"subscriptionState"`
		AcknowledgementState int64        `json:"acknowledgementState"`
		LineItems            []v2LineItem `json:"lineItems"`
	}

	var p v2Payload
	_ = json.Unmarshal(raw, &p)

	now := time.Now()
	active := false
	orderID := ""
	productID := ""

	for _, li := range p.LineItems {
		if productID == "" && strings.TrimSpace(li.ProductID) != "" {
			productID = strings.TrimSpace(li.ProductID)
		}
		if orderID == "" && strings.TrimSpace(li.LatestSuccessfulOrderID) != "" {
			orderID = strings.TrimSpace(li.LatestSuccessfulOrderID)
		}
		if strings.TrimSpace(li.ExpiryTime) != "" {
			if t, perr := time.Parse(time.RFC3339Nano, li.ExpiryTime); perr == nil {
				if t.After(now) {
					active = true
				}
			}
		}
	}

	purchaseState := int64(1) // 1 = not active/expired
	status := strings.TrimSpace(p.SubscriptionState)

	if active {
		purchaseState = 0 // 0 = active
		if status == "" {
			status = "ACTIVE"
		}
	}

	ack := p.AcknowledgementState == 1

	return models.GooglePurchase{
		Kind:          "subscription",
		ProductID:     productID, // может быть пустым — это ок, хендлер подставит
		PurchaseToken: token,
		OrderID:       orderID,
		PackageName:   s.cfg.PackageName,
		PurchaseState: purchaseState,
		Status:        status,
		Acknowledged:  ack,
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
