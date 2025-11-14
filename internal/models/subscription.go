package models

import (
	"fmt"
	"strings"
	"time"
)

type SubscriptionType string

const (
	SubscriptionTypeService SubscriptionType = "service"
	SubscriptionTypeRent    SubscriptionType = "rent"
	SubscriptionTypeWork    SubscriptionType = "work"
)

func ParseSubscriptionType(raw string) (SubscriptionType, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case string(SubscriptionTypeService):
		return SubscriptionTypeService, nil
	case string(SubscriptionTypeRent):
		return SubscriptionTypeRent, nil
	case string(SubscriptionTypeWork):
		return SubscriptionTypeWork, nil
	default:
		return "", fmt.Errorf("unsupported subscription type: %s", raw)
	}
}

func (t SubscriptionType) IsZero() bool {
	return string(t) == ""
}

type ExecutorSubscription struct {
	ID        int              `json:"id"`
	UserID    int              `json:"user_id"`
	Type      SubscriptionType `json:"type"`
	ExpiresAt time.Time        `json:"expires_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt *time.Time       `json:"updated_at,omitempty"`
}

type SubscriptionResponses struct {
	ID                     int        `json:"id"`
	UserID                 int        `json:"user_id"`
	Packs                  int        `json:"packs"`
	Status                 string     `json:"status"`
	RenewsAt               time.Time  `json:"renews_at"`
	MonthlyQuota           int        `json:"monthly_quota"`
	Remaining              int        `json:"remaining"`
	ProviderSubscriptionID *string    `json:"provider_subscription_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              *time.Time `json:"updated_at,omitempty"`
}

type SubscriptionInfo struct {
	Type          SubscriptionType `json:"type"`
	Active        bool             `json:"active"`
	ExpiresAt     *time.Time       `json:"expires_at,omitempty"`
	RemainingDays int              `json:"remaining_days"`
}

type SubscriptionResponsesSummary struct {
	Remaining    int        `json:"remaining"`
	MonthlyQuota int        `json:"monthly_quota"`
	Status       string     `json:"status"`
	RenewsAt     *time.Time `json:"renews_at,omitempty"`
}

type SubscriptionProfile struct {
	Service                     SubscriptionInfo             `json:"service"`
	Rent                        SubscriptionInfo             `json:"rent"`
	Work                        SubscriptionInfo             `json:"work"`
	Responses                   SubscriptionResponsesSummary `json:"responses"`
	ActiveExecutorListingsCount int                          `json:"active_executor_listings_count"`
}

type SubscriptionSummary struct {
	ServiceActive      bool       `json:"service_active"`
	RentActive         bool       `json:"rent_active"`
	WorkActive         bool       `json:"work_active"`
	ResponsesRemaining int        `json:"responses_remaining"`
	RenewDate          *time.Time `json:"renew_date,omitempty"`
}
