package taxi

import (
	"context"
	"fmt"
)

// DepositDriverBalance increases driver's balance by the provided amount in tenge.
func DepositDriverBalance(ctx context.Context, deps *TaxiDeps, driverID int64, amount int) error {
	if deps == nil {
		return fmt.Errorf("taxi deps are nil")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	module, err := ensureModule(deps)
	if err != nil {
		return err
	}
	if module == nil || module.driversRepo == nil {
		return fmt.Errorf("taxi drivers repo is not initialised")
	}
	_, err = module.driversRepo.Deposit(ctx, driverID, amount)
	return err
}
