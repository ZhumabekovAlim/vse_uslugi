package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type BusinessService struct {
	BusinessRepo *repositories.BusinessRepository
	UserRepo     *repositories.UserRepository
	ChatRepo     *repositories.ChatRepository
}

type PurchaseRequest struct {
	Seats         int      `json:"seats"`
	Provider      *string  `json:"provider,omitempty"`
	ProviderTxnID *string  `json:"provider_txn_id,omitempty"`
	State         *string  `json:"state,omitempty"`
	Amount        *float64 `json:"amount,omitempty"`
	Payload       any      `json:"payload,omitempty"`
}

type CreateWorkerRequest struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type UpdateWorkerRequest struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

// AttachListingRequest binds an existing listing to a worker.
type AttachListingRequest struct {
	ListingType string `json:"listing_type"`
	ListingID   int    `json:"listing_id"`
}

func (s *BusinessService) GetOrCreateAccount(ctx context.Context, businessUserID int) (models.BusinessAccount, error) {
	acc, err := s.BusinessRepo.GetAccountByUserID(ctx, businessUserID)
	if err != nil {
		return models.BusinessAccount{}, err
	}
	if acc.ID == 0 {
		return s.BusinessRepo.CreateAccount(ctx, businessUserID)
	}
	return acc, nil
}

func (s *BusinessService) PurchaseSeats(ctx context.Context, businessUserID int, req PurchaseRequest) (models.BusinessAccount, error) {
	if req.Seats <= 0 {
		return models.BusinessAccount{}, fmt.Errorf("seats must be greater than zero")
	}
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return models.BusinessAccount{}, err
	}

	amount := float64(req.Seats * 1000)
	if req.Amount != nil {
		amount = *req.Amount
	}

	purchase := models.BusinessSeatPurchase{
		BusinessUserID: businessUserID,
		Seats:          req.Seats,
		Amount:         amount,
		Provider:       req.Provider,
		ProviderTxnID:  req.ProviderTxnID,
		State:          req.State,
		PayloadJSON:    req.Payload,
	}
	if err := s.BusinessRepo.SaveSeatPurchase(ctx, purchase); err != nil {
		return models.BusinessAccount{}, err
	}

	if err := s.BusinessRepo.AddSeats(ctx, businessUserID, req.Seats); err != nil {
		return models.BusinessAccount{}, err
	}

	_, err = s.UserRepo.UpdateUser(ctx, models.User{ID: businessUserID, Role: "business"})
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return models.BusinessAccount{}, err
	}

	return s.BusinessRepo.GetAccountByUserID(ctx, acc.BusinessUserID)
}

func (s *BusinessService) validateSeatAvailability(acc models.BusinessAccount) error {
	if acc.Status == "suspended" {
		return repositories.ErrBusinessAccountSuspended
	}
	if acc.SeatsUsed >= acc.SeatsTotal {
		return repositories.ErrNoFreeSeats
	}
	return nil
}

func (s *BusinessService) CreateWorker(ctx context.Context, businessUserID int, req CreateWorkerRequest) (models.BusinessWorker, error) {
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	if err := s.validateSeatAvailability(acc); err != nil {
		return models.BusinessWorker{}, err
	}

	existing, err := s.BusinessRepo.GetWorkerByLogin(ctx, req.Login)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	if existing.ID != 0 {
		return models.BusinessWorker{}, fmt.Errorf("login already taken")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.BusinessWorker{}, err
	}

	user := models.User{ //nolint:exhaustruct
		Name:     req.Name,
		Surname:  req.Surname,
		Phone:    req.Phone,
		Password: string(hashedPassword),
		Role:     "business_worker",
	}
	createdUser, err := s.UserRepo.CreateUser(ctx, user)
	if err != nil {
		return models.BusinessWorker{}, err
	}

	chatID, err := s.ChatRepo.CreateChat(ctx, models.Chat{User1ID: businessUserID, User2ID: createdUser.ID})
	if err != nil {
		return models.BusinessWorker{}, err
	}

	worker, err := s.BusinessRepo.CreateWorker(ctx, models.BusinessWorker{
		BusinessUserID: businessUserID,
		WorkerUserID:   createdUser.ID,
		Login:          req.Login,
		ChatID:         chatID,
		Status:         "active",
	})
	if err != nil {
		return models.BusinessWorker{}, err
	}

	if err := s.BusinessRepo.IncrementSeatsUsed(ctx, businessUserID); err != nil {
		return models.BusinessWorker{}, err
	}
	worker.User = &createdUser
	return worker, nil
}

func (s *BusinessService) ListWorkers(ctx context.Context, businessUserID int) ([]models.BusinessWorker, error) {
	return s.BusinessRepo.GetWorkersByBusiness(ctx, businessUserID)
}

func (s *BusinessService) UpdateWorker(ctx context.Context, businessUserID, workerID int, req UpdateWorkerRequest) (models.BusinessWorker, error) {
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	if acc.Status == "suspended" {
		return models.BusinessWorker{}, repositories.ErrBusinessAccountSuspended
	}

	existing, err := s.BusinessRepo.GetWorkerByID(ctx, workerID)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	if existing.ID == 0 || existing.BusinessUserID != businessUserID {
		return models.BusinessWorker{}, sql.ErrNoRows
	}

	if req.Login != "" {
		if taken, err := s.BusinessRepo.GetWorkerByLogin(ctx, req.Login); err == nil && taken.ID != 0 && taken.ID != workerID {
			return models.BusinessWorker{}, fmt.Errorf("login already taken")
		}
	}

	worker := models.BusinessWorker{ID: workerID, BusinessUserID: businessUserID, Login: req.Login, Status: req.Status}
	if worker.Login == "" {
		worker.Login = existing.Login
	}
	if worker.Status == "" {
		worker.Status = "active"
	}
	if err := s.BusinessRepo.UpdateWorker(ctx, worker); err != nil {
		return models.BusinessWorker{}, err
	}

	userUpdates := models.User{ID: existing.WorkerUserID, Name: req.Name, Surname: req.Surname, Phone: req.Phone}
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return models.BusinessWorker{}, err
		}
		userUpdates.Password = string(hashed)
	}
	if req.Name != "" || req.Surname != "" || req.Phone != "" || req.Password != "" {
		if _, err := s.UserRepo.UpdateUser(ctx, userUpdates); err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
			return models.BusinessWorker{}, err
		}
	}
	workers, err := s.BusinessRepo.GetWorkersByBusiness(ctx, businessUserID)
	if err != nil {
		return models.BusinessWorker{}, err
	}
	for _, w := range workers {
		if w.ID == workerID {
			return w, nil
		}
	}
	return models.BusinessWorker{}, sql.ErrNoRows
}

func (s *BusinessService) DisableWorker(ctx context.Context, businessUserID, workerID int) error {
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return err
	}
	if acc.Status == "suspended" {
		return repositories.ErrBusinessAccountSuspended
	}
	return s.BusinessRepo.DisableWorker(ctx, workerID, businessUserID)
}

// AttachListing links a listing to a worker under the given business account.
func (s *BusinessService) AttachListing(ctx context.Context, businessUserID, workerID int, req AttachListingRequest) error {
	if req.ListingID == 0 || strings.TrimSpace(req.ListingType) == "" {
		return fmt.Errorf("listing_type and listing_id are required")
	}
	worker, err := s.BusinessRepo.GetWorkerByID(ctx, workerID)
	if err != nil {
		return err
	}
	if worker.ID == 0 || worker.BusinessUserID != businessUserID {
		return sql.ErrNoRows
	}

	attachment := models.BusinessWorkerListing{ //nolint:exhaustruct
		BusinessUserID: businessUserID,
		WorkerUserID:   worker.WorkerUserID,
		ListingType:    strings.TrimSpace(strings.ToLower(req.ListingType)),
		ListingID:      req.ListingID,
	}
	return s.BusinessRepo.UpsertWorkerListing(ctx, attachment)
}

// DetachListing removes a listing binding from a worker.
func (s *BusinessService) DetachListing(ctx context.Context, businessUserID, workerID int, req AttachListingRequest) error {
	worker, err := s.BusinessRepo.GetWorkerByID(ctx, workerID)
	if err != nil {
		return err
	}
	if worker.ID == 0 || worker.BusinessUserID != businessUserID {
		return sql.ErrNoRows
	}

	attachment := models.BusinessWorkerListing{ //nolint:exhaustruct
		BusinessUserID: businessUserID,
		WorkerUserID:   worker.WorkerUserID,
		ListingType:    strings.TrimSpace(strings.ToLower(req.ListingType)),
		ListingID:      req.ListingID,
	}
	return s.BusinessRepo.DeleteWorkerListing(ctx, attachment)
}

// ListWorkerListings returns workers with attached listings map keyed by worker user ID.
func (s *BusinessService) ListWorkerListings(ctx context.Context, businessUserID int) (map[int][]models.BusinessWorkerListing, error) {
	if _, err := s.GetOrCreateAccount(ctx, businessUserID); err != nil {
		return nil, err
	}
	return s.BusinessRepo.ListWorkerListings(ctx, businessUserID)
}
