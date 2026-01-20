package models

type GooglePurchase struct {
	Kind          string
	ProductID     string
	PurchaseToken string
	OrderID       string
	PackageName   string

	// Subscription-only
	ExpiryTimeMillis int64
	PaymentState     *int64 // <-- ВАЖНО: pointer (как в v0.247.0)
	CancelReason     int64
	AutoRenewing     bool

	// 0 = OK/ACTIVE, 1 = EXPIRED, 2 = PENDING/UNKNOWN
	PurchaseState int64

	Acknowledged bool
	Consumed     bool

	Status string
	Raw    string
}
