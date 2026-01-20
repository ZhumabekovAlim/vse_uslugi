package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type BusinessService struct {
	BusinessRepo *repositories.BusinessRepository
	UserRepo     *repositories.UserRepository
	ChatRepo     *repositories.ChatRepository
	ServiceRepo  *repositories.ServiceRepository
	WorkRepo     *repositories.WorkRepository
	RentRepo     *repositories.RentRepository
	AdRepo       *repositories.AdRepository
	WorkAdRepo   *repositories.WorkAdRepository
	RentAdRepo   *repositories.RentAdRepository
}

type PurchaseRequest struct {
	Seats         int      `json:"seats"`
	DurationDays  *int     `json:"duration_days,omitempty"`
	Provider      *string  `json:"provider,omitempty"`
	ProviderTxnID *string  `json:"provider_txn_id,omitempty"`
	State         *string  `json:"state,omitempty"`
	Amount        *float64 `json:"amount,omitempty"`
	Payload       any      `json:"payload,omitempty"`
}

type CreateWorkerRequest struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	CanRespond *bool  `json:"can_respond,omitempty"`
}

type UpdateWorkerRequest struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	Status     string `json:"status"`
	CanRespond *bool  `json:"can_respond,omitempty"`
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
	return s.normalizeAccount(acc), nil
}

func (s *BusinessService) PurchaseSeats(ctx context.Context, businessUserID int, req PurchaseRequest) (models.BusinessAccount, error) {
	if req.Seats <= 0 {
		return models.BusinessAccount{}, fmt.Errorf("seats must be greater than zero")
	}
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return models.BusinessAccount{}, err
	}

	durationDays := DefaultBusinessSeatDuration() // 30 дней по умолчанию

	if req.DurationDays != nil {
		if *req.DurationDays <= 0 {
			return models.BusinessAccount{}, fmt.Errorf("duration_days must be greater than zero")
		}
		durationDays = *req.DurationDays
	}

	expiresAt := time.Now().UTC().Add(time.Duration(durationDays) * 24 * time.Hour)

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

	if err := s.BusinessRepo.SetSeats(ctx, businessUserID, req.Seats, expiresAt); err != nil {
		return models.BusinessAccount{}, err
	}

	_, err = s.UserRepo.UpdateUser(ctx, models.User{ID: businessUserID, Role: "business"})
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return models.BusinessAccount{}, err
	}

	return s.GetOrCreateAccount(ctx, acc.BusinessUserID)
}

func (s *BusinessService) validateSeatAvailability(acc models.BusinessAccount) error {
	if acc.Status == "suspended" {
		return repositories.ErrBusinessAccountSuspended
	}
	if acc.Expired {
		return repositories.ErrNoFreeSeats
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

	canRespond := false
	if req.CanRespond != nil {
		canRespond = *req.CanRespond
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
		CanRespond:     canRespond,
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

	worker := models.BusinessWorker{ID: workerID, BusinessUserID: businessUserID, Login: req.Login, Status: req.Status, CanRespond: existing.CanRespond}
	if worker.Login == "" {
		worker.Login = existing.Login
	}
	if worker.Status == "" {
		worker.Status = "active"
	}
	if req.CanRespond != nil {
		worker.CanRespond = *req.CanRespond
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

func (s *BusinessService) DeleteWorker(ctx context.Context, businessUserID, workerID int) error {
	acc, err := s.GetOrCreateAccount(ctx, businessUserID)
	if err != nil {
		return err
	}
	if acc.Status == "suspended" {
		return repositories.ErrBusinessAccountSuspended
	}

	worker, err := s.BusinessRepo.GetWorkerByID(ctx, workerID)
	if err != nil {
		return err
	}
	if worker.ID == 0 || worker.BusinessUserID != businessUserID {
		return sql.ErrNoRows
	}

	return s.BusinessRepo.DeleteWorker(ctx, worker)
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
func (s *BusinessService) ListWorkerListings(ctx context.Context, businessUserID int) (map[int][]models.BusinessWorkerListingDetails, error) {
	if _, err := s.GetOrCreateAccount(ctx, businessUserID); err != nil {
		return nil, err
	}
	listings, err := s.BusinessRepo.ListWorkerListings(ctx, businessUserID)
	if err != nil {
		return nil, err
	}

	result := make(map[int][]models.BusinessWorkerListingDetails, len(listings))
	for workerID, items := range listings {
		for _, listing := range items {
			details, err := s.buildListingDetails(ctx, businessUserID, listing)
			if err != nil {
				return nil, err
			}
			result[workerID] = append(result[workerID], details)
		}
	}
	return result, nil
}

func (s *BusinessService) buildListingDetails(ctx context.Context, businessUserID int, listing models.BusinessWorkerListing) (models.BusinessWorkerListingDetails, error) {
	details := models.BusinessWorkerListingDetails{
		BusinessUserID: listing.BusinessUserID,
		WorkerUserID:   listing.WorkerUserID,
		ListingType:    listing.ListingType,
		ListingID:      listing.ListingID,
	}

	switch listing.ListingType {
	case "service":
		service, err := s.ServiceRepo.GetServiceByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = service.Images
		details.Videos = service.Videos
		details.Liked = service.Liked
		details.Negotiable = service.Negotiable
		details.Price = service.Price
		details.PriceTo = service.PriceTo
		details.CreatedAt = service.CreatedAt
	case "work":
		work, err := s.WorkRepo.GetWorkByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = work.Images
		details.Videos = work.Videos
		details.Liked = work.Liked
		details.Negotiable = work.Negotiable
		details.Price = work.Price
		details.PriceTo = work.PriceTo
		details.CreatedAt = work.CreatedAt
	case "rent":
		rent, err := s.RentRepo.GetRentByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = rent.Images
		details.Videos = rent.Videos
		details.Liked = rent.Liked
		details.Negotiable = rent.Negotiable
		details.Price = rent.Price
		details.PriceTo = rent.PriceTo
		details.CreatedAt = rent.CreatedAt
	case "ad":
		ad, err := s.AdRepo.GetAdByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = ad.Images
		details.Videos = ad.Videos
		details.Liked = ad.Liked
		details.Negotiable = ad.Negotiable
		details.Price = ad.Price
		details.PriceTo = ad.PriceTo
		details.CreatedAt = ad.CreatedAt
	case "work_ad":
		workAd, err := s.WorkAdRepo.GetWorkAdByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = workAd.Images
		details.Videos = workAd.Videos
		details.Liked = workAd.Liked
		details.Negotiable = workAd.Negotiable
		details.Price = workAd.Price
		details.PriceTo = workAd.PriceTo
		details.CreatedAt = workAd.CreatedAt
	case "rent_ad":
		rentAd, err := s.RentAdRepo.GetRentAdByID(ctx, listing.ListingID, businessUserID)
		if err != nil {
			return details, err
		}
		details.Images = rentAd.Images
		details.Videos = rentAd.Videos
		details.Liked = rentAd.Liked
		details.Negotiable = rentAd.Negotiable
		details.Price = rentAd.Price
		details.PriceTo = rentAd.PriceTo
		details.CreatedAt = rentAd.CreatedAt
	default:
		details.CreatedAt = listing.CreatedAt
	}
	return details, nil
}

const defaultBusinessSeatDurationDays = 30

// DefaultBusinessSeatDuration returns the default duration in days for business seats.
func DefaultBusinessSeatDuration() int {
	return defaultBusinessSeatDurationDays
}

func (s *BusinessService) normalizeAccount(acc models.BusinessAccount) models.BusinessAccount {
	if acc.SeatsExpiresAt == nil {
		return acc
	}
	now := time.Now()
	if acc.SeatsExpiresAt.Before(now) || acc.SeatsExpiresAt.Equal(now) {
		acc.Expired = true
		acc.SeatsTotal = 0
		acc.SeatsUsed = 0
	}
	return acc
}
