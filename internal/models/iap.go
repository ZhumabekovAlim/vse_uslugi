package models

import "fmt"

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
	switch t.Type {
	case IAPTargetTypeSubscription:
		if _, err := ParseSubscriptionType(t.SubscriptionType); err != nil {
			return fmt.Errorf("subscription_type: %w", err)
		}
		if t.Months <= 0 {
			return fmt.Errorf("months must be positive")
		}
	case IAPTargetTypeResponses:
		if t.Quantity < 0 {
			return fmt.Errorf("quantity must be non-negative")
		}
	case IAPTargetTypeTop:
		if t.ID <= 0 {
			return fmt.Errorf("id must be positive")
		}
		if t.DurationDays <= 0 {
			return fmt.Errorf("duration_days must be positive")
		}
		if _, ok := AllowedTopTypes()[t.ListingType]; !ok {
			return fmt.Errorf("invalid listing_type: %s", t.ListingType)
		}
	case IAPTargetTypeBusiness:
		if t.Seats <= 0 {
			return fmt.Errorf("seats must be positive")
		}
	default:
		return fmt.Errorf("unsupported target type: %s", t.Type)
	}
	return nil
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
