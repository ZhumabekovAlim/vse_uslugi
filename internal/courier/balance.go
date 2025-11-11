package courier

import (
	"context"
	"fmt"

	"naimuBack/internal/courier/repo"
)

// DepositBalance credits courier balance with the specified amount of tenge.
func DepositBalance(ctx context.Context, deps *Deps, courierID int64, amount int) error {
	if deps == nil {
		return fmt.Errorf("courier deps are nil")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if deps.DB == nil {
		return fmt.Errorf("courier deps DB is required")
	}
	couriersRepo := repo.NewCouriersRepo(deps.DB)
	_, err := couriersRepo.DepositBalance(ctx, courierID, amount)
	return err
}
