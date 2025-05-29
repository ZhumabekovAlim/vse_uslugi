package models

import (
	"errors"
)

var ErrWorkerNotFound = errors.New("worker not found")
var ErrClientNotFound = errors.New("client not found")
var ErrOrderNotFound = errors.New("order not found")
var ErrWorkerLimitExceeded = errors.New("worker limit exceeded for order")
var (
	ErrNoRecord               = errors.New("models: no matching record found")
	ErrInvalidCredentials     = errors.New("models: invalid credentials")
	ErrDuplicateEmail         = errors.New("models: duplicate email")
	ErrDuplicateClubName      = errors.New("models: duplicate club name")
	ErrDuplicatePhone         = errors.New("models: duplicate phone number")
	ErrProductNotFound        = errors.New("models: duplicate product")
	ErrUserNotFound           = errors.New("models: user not found")
	ErrInvalidPassword        = errors.New("models: invalid password")
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrCompanyNotFound        = errors.New("company not found")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrExpenseNotFound        = errors.New("personal expense not found")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrBalanceHistoryNotFound = errors.New("balance history not found")
	ErrNoFriends              = errors.New("not friends")
	ErrAlreadyFriends         = errors.New("already friends")
	ErrCardNotFound           = errors.New("card not found")
	ErrReviewNotFound         = errors.New("review not found")
)
