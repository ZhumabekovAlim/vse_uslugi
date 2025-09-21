package models

import "time"

// SubscriptionSlots represents monthly executor listing slots.
type SubscriptionSlots struct {
	ID                     int        `json:"id"`
	UserID                 int        `json:"user_id"`
	Slots                  int        `json:"slots"`
	Status                 string     `json:"status"`
	RenewsAt               time.Time  `json:"renews_at"`
	ProviderSubscriptionID *string    `json:"provider_subscription_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              *time.Time `json:"updated_at,omitempty"`
}

// SubscriptionResponses represents monthly response packs.
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

// SubscriptionProfile aggregates subscription information for profile endpoint.
type SubscriptionProfile struct {
	ExecutorListingSlots        int `json:"executor_listing_slots"`
	ActiveExecutorListingsCount int `json:"active_executor_listings_count"`
	ResponsePacks               int `json:"response_packs"`
	MonthlyResponsesQuota       int `json:"monthly_responses_quota"`
	RemainingResponses          int `json:"remaining_responses"`
	MonthlyAmount               int `json:"monthly_amount"`
	Status                      struct {
		Slots     string `json:"slots"`
		Responses string `json:"responses"`
	} `json:"status"`
	RenewsAt      *time.Time `json:"renews_at,omitempty"`
	GraceUntil    *time.Time `json:"grace_until,omitempty"`
	BillingNotice *string    `json:"billing_notice,omitempty"`
}

// SubscriptionSummary represents a lightweight subscription snapshot for the
// profile page.
type SubscriptionSummary struct {
	ActivePaidListings int        `json:"active_paid_listings"`
	PurchasedListings  int        `json:"purchased_listings"`
	ResponsesCount     int        `json:"responses_count"`
	RenewDate          *time.Time `json:"renew_date,omitempty"`
	MonthlyPayment     int        `json:"monthly_payment"`
}
