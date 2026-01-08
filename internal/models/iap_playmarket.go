package models

type GooglePurchase struct {
	Kind          string // "product" | "subscription"
	ProductID     string
	PurchaseToken string
	OrderID       string
	PackageName   string

	// Subscription-only
	ExpiryTimeMillis int64
	PaymentState     *int64 // из SubscriptionPurchase.PaymentState
	CancelReason     int64  // из SubscriptionPurchase.CancelReason
	AutoRenewing     bool   // из SubscriptionPurchase.AutoRenewing

	// Product-only (и “виртуально” для subscription, если ты хочешь)
	// 0 = Purchased, 1 = Canceled, 2 = Pending
	PurchaseState int64

	Acknowledged bool
	Consumed     bool

	// Нормальный единый статус
	Status string // "ACTIVE" | "EXPIRED" | "PENDING" | "CANCELED" | "UNKNOWN"

	Raw string
}
