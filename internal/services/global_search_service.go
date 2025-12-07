package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"sort"
	"strconv"
	"time"
)

var (
	// ErrUnsupportedListingType is returned when the requested listing type is not supported.
	ErrUnsupportedListingType = errors.New("unsupported listing type")
)

// GlobalSearchService aggregates listings across different domains.
type GlobalSearchService struct {
	ServiceRepo *repositories.ServiceRepository
	AdRepo      *repositories.AdRepository
	WorkRepo    *repositories.WorkRepository
	WorkAdRepo  *repositories.WorkAdRepository
	RentRepo    *repositories.RentRepository
	RentAdRepo  *repositories.RentAdRepository
}

// Search returns mixed listings filtered by the provided criteria.
func (s *GlobalSearchService) Search(ctx context.Context, req models.GlobalSearchRequest) (models.GlobalSearchResponse, error) {
	if len(req.Types) == 0 {
		return models.GlobalSearchResponse{}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	hasUserLocation := req.Latitude != nil && req.Longitude != nil
	var userLat, userLon float64
	if hasUserLocation {
		userLat = *req.Latitude
		userLon = *req.Longitude
	}

	perTypeLimit := limit * page
	if perTypeLimit <= 0 {
		perTypeLimit = limit
	}

	subcategoryStrings := make([]string, 0, len(req.SubcategoryIDs))
	for _, id := range req.SubcategoryIDs {
		subcategoryStrings = append(subcategoryStrings, strconv.Itoa(id))
	}

	now := time.Now().UTC()
	entries := make([]globalSearchEntry, 0)

	ratings := req.Ratings
	priceFrom := req.PriceFrom
	priceTo := req.PriceTo
	sortOption := req.SortOption
	for _, listingType := range req.Types {
		switch listingType {
		case "service":
			if s.ServiceRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			services, _, _, err := s.ServiceRepo.GetServicesWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.OnSite, req.Negotiable)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, svc := range services {
				svcCopy := svc
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, svcCopy.Latitude, svcCopy.Longitude)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, Service: &svcCopy}, svc.Top, svc.CreatedAt, now))
			}
		case "ad":
			if s.AdRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			ads, _, _, err := s.AdRepo.GetAdWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.OnSite, req.Negotiable)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, ad := range ads {
				adCopy := ad
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, adCopy.Latitude, adCopy.Longitude)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, Ad: &adCopy}, ad.Top, ad.CreatedAt, now))
			}
		case "work":
			if s.WorkRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			works, _, _, err := s.WorkRepo.GetWorksWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.Negotiable, req.WorkExperience, req.WorkSchedules, req.PaymentPeriods, req.RemoteWork, req.Languages, req.Educations)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, work := range works {
				workCopy := work
				lat, lon := workCopy.Latitude, workCopy.Longitude
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, &lat, &lon)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, Work: &workCopy}, work.Top, work.CreatedAt, now))
			}
		case "work_ad":
			if s.WorkAdRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			workAds, _, _, err := s.WorkAdRepo.GetWorksAdWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.Negotiable, req.WorkExperience, req.WorkSchedules, req.PaymentPeriods, req.RemoteWork, req.Languages, req.Educations)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, workAd := range workAds {
				workAdCopy := workAd
				lat, lon := workAdCopy.Latitude, workAdCopy.Longitude
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, &lat, &lon)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, WorkAd: &workAdCopy}, workAd.Top, workAd.CreatedAt, now))
			}
		case "rent":
			if s.RentRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			rents, _, _, err := s.RentRepo.GetRentsWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.Negotiable, req.RentTypes, req.Deposits)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, rent := range rents {
				rentCopy := rent
				lat, lon := rentCopy.Latitude, rentCopy.Longitude
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, &lat, &lon)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, Rent: &rentCopy}, rent.Top, rent.CreatedAt, now))
			}
		case "rent_ad":
			if s.RentAdRepo == nil {
				return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
			}
			rentAds, _, _, err := s.RentAdRepo.GetRentsAdWithFilters(ctx, req.UserID, 0, req.CategoryIDs, subcategoryStrings, priceFrom, priceTo, ratings, sortOption, perTypeLimit, 0, req.Negotiable, req.RentTypes, req.Deposits)
			if err != nil {
				return models.GlobalSearchResponse{}, err
			}
			for _, rentAd := range rentAds {
				rentAdCopy := rentAd
				lat, lon := rentAdCopy.Latitude, rentAdCopy.Longitude
				distance := calculateDistanceKm(userLat, userLon, hasUserLocation, &lat, &lon)
				if req.RadiusKm != nil && hasUserLocation {
					if distance == nil || *distance > *req.RadiusKm {
						continue
					}
				}
				entries = append(entries, newGlobalSearchEntry(listingType, models.GlobalSearchItem{Type: listingType, Distance: distance, RentAd: &rentAdCopy}, rentAd.Top, rentAd.CreatedAt, now))
			}
		default:
			return models.GlobalSearchResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedListingType, listingType)
		}
	}

	if len(entries) == 0 {
		return models.GlobalSearchResponse{Results: nil, Total: 0, Page: page, Limit: limit}, nil
	}

	useDistanceSort := hasUserLocation && req.RadiusKm != nil
	sortGlobalSearchEntries(entries, useDistanceSort)

	total := len(entries)
	start := (page - 1) * limit
	if start >= total {
		return models.GlobalSearchResponse{Results: []models.GlobalSearchItem{}, Total: total, Page: page, Limit: limit}, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	results := make([]models.GlobalSearchItem, 0, end-start)
	for _, entry := range entries[start:end] {
		results = append(results, entry.item)
	}

	return models.GlobalSearchResponse{Results: results, Total: total, Page: page, Limit: limit}, nil
}

type globalSearchEntry struct {
	item      models.GlobalSearchItem
	state     listingTopState
	createdAt time.Time
}

func sortGlobalSearchEntries(entries []globalSearchEntry, prioritizeDistance bool) {
	if len(entries) < 2 {
		return
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if prioritizeDistance {
			distA := entries[i].item.Distance
			distB := entries[j].item.Distance

			if distA != nil && distB != nil && *distA != *distB {
				return *distA < *distB
			}
		}

		return lessByTopState(entries[i].state, entries[i].createdAt, entries[j].state, entries[j].createdAt)
	})
}

func newGlobalSearchEntry(listingType string, item models.GlobalSearchItem, top string, createdAt, now time.Time) globalSearchEntry {
	return globalSearchEntry{
		item:      item,
		state:     computeTopState(top, now),
		createdAt: createdAt.UTC(),
	}
}

type listingTopState struct {
	active      bool
	activatedAt time.Time
}

func computeTopState(raw string, now time.Time) listingTopState {
	info, err := models.ParseTopInfo(raw)
	if err != nil || info == nil {
		return listingTopState{}
	}
	return listingTopState{
		active:      info.IsActive(now),
		activatedAt: info.ActivatedAt,
	}
}

func lessByTopState(a listingTopState, createdAtA time.Time, b listingTopState, createdAtB time.Time) bool {
	if a.active != b.active {
		return a.active
	}
	if a.active && b.active {
		if !a.activatedAt.Equal(b.activatedAt) {
			return a.activatedAt.After(b.activatedAt)
		}
		return createdAtA.After(createdAtB)
	}
	return createdAtA.After(createdAtB)
}

func calculateDistanceKm(userLat, userLon float64, hasUserLocation bool, listingLat, listingLon *string) *float64 {
	if !hasUserLocation || listingLat == nil || listingLon == nil {
		return nil
	}

	latValue, err := strconv.ParseFloat(*listingLat, 64)
	if err != nil {
		return nil
	}
	lonValue, err := strconv.ParseFloat(*listingLon, 64)
	if err != nil {
		return nil
	}

	distance := haversineDistanceKm(userLat, userLon, latValue, lonValue)
	return &distance
}

func haversineDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
