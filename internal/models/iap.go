package models

import (
	"errors"
	"fmt"
	"strings"
)

const (
	IAPTargetTypeSubscription = "subscription"
	IAPTargetTypeResponses    = "responses"
	IAPTargetTypeTop          = "top"
	IAPTargetTypeBusiness     = "business"
)

// IAPTarget describes what should happen after a successful IAP transaction.
// The client provides this payload alongside the transaction so that the server
// can apply the correct entitlement.
type IAPTarget struct {
	Type             string `json:"type"`
	ID               int64  `json:"id,omitempty"`
	SubscriptionType string `json:"subscription_type,omitempty"`
	Months           int    `json:"months,omitempty"`
	Quantity         int    `json:"quantity,omitempty"`
	ListingType      string `json:"listing_type,omitempty"`
	DurationDays     int    `json:"duration_days,omitempty"`
	Seats            int    `json:"seats,omitempty"`
}

// Validate checks required fields for each target type.
func (t IAPTarget) Validate() error {
	tt := strings.TrimSpace(strings.ToLower(t.Type))
	if tt == "" {
		return errors.New("type is required")
	}

	switch tt {
	case IAPTargetTypeSubscription:
		if strings.TrimSpace(t.SubscriptionType) == "" {
			return errors.New("subscription_type is required")
		}
		if t.Months <= 0 {
			return errors.New("months must be positive")
		}
		return nil

	case IAPTargetTypeResponses:
		if t.Quantity <= 0 {
			return errors.New("quantity must be positive")
		}
		return nil

	case IAPTargetTypeTop:
		// ВНИМАНИЕ:
		// duration_days — обязателен ВСЕГДА (сервер)
		// listing_type + id — могут прийти позже (клиент), поэтому делаем “мягко”:
		if t.DurationDays <= 0 {
			return errors.New("duration_days must be positive")
		}
		// listing_type/id валидируем только если они уже заданы
		if strings.TrimSpace(t.ListingType) != "" || t.ID != 0 {
			if strings.TrimSpace(t.ListingType) == "" {
				return errors.New("listing_type is required")
			}
			if t.ID <= 0 {
				return errors.New("id must be positive")
			}
		}
		return nil

	case IAPTargetTypeBusiness:
		if t.Seats <= 0 {
			return errors.New("seats must be positive")
		}
		// ❗ duration_days тут НЕ нужен
		// months тут НЕ нужен (если ты его не используешь)
		return nil

	default:
		return fmt.Errorf("unsupported type: %s", tt)
	}
}

// AppleTransaction contains decoded transaction fields from Apple's JWS payload.
type AppleTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	ProductID             string `json:"productId"`
	BundleID              string `json:"bundleId"`
	Environment           string `json:"environment"`
	Raw                   string `json:"-"`
}

// AppleRenewalInfo contains decoded renewal fields from Apple's signedRenewalInfo JWS payload.
type AppleRenewalInfo struct {
	OriginalTransactionID string `json:"originalTransactionId"`
	AutoRenewProductID    string `json:"autoRenewProductId,omitempty"`
	Environment           string `json:"environment"`
	SignedDate            int64  `json:"signedDate"`
	BundleID              string `json:"bundleId,omitempty"`
	Raw                   string `json:"-"`
}

// AppleNotification wraps the server notification payload (after signature verification).
type AppleNotification struct {
	NotificationType string `json:"notificationType"`
	Subtype          string `json:"subtype,omitempty"`
	Data             struct {
		AppAppleID            int64  `json:"appAppleId,omitempty"`
		BundleID              string `json:"bundleId,omitempty"`
		Environment           string `json:"environment"`
		SignedTransactionInfo string `json:"signedTransactionInfo,omitempty"`
		SignedRenewalInfo     string `json:"signedRenewalInfo,omitempty"`
		Status                string `json:"status,omitempty"`
	} `json:"data"`
	Version    string `json:"version"`
	SignedDate int64  `json:"signedDate"`
	Raw        string `json:"-"`
}
