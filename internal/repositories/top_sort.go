package repositories

import (
	"naimuBack/internal/models"
	"sort"
	"time"
)

type listingTopState struct {
	active      bool
	activatedAt time.Time
}

func sortServicesByTop(services []models.Service) {
	sortListingsByTop(services, func(s models.Service) string { return s.Top }, func(s models.Service) time.Time { return s.CreatedAt })
}

func sortFilteredServicesByTop(services []models.FilteredService) {
	sortListingsByTop(services, func(s models.FilteredService) string { return s.Top }, func(s models.FilteredService) time.Time { return s.CreatedAt })
}

func sortAdsByTop(ads []models.Ad) {
	sortListingsByTop(ads, func(a models.Ad) string { return a.Top }, func(a models.Ad) time.Time { return a.CreatedAt })
}

func sortFilteredAdsByTop(ads []models.FilteredAd) {
	sortListingsByTop(ads, func(a models.FilteredAd) string { return a.Top }, func(a models.FilteredAd) time.Time { return a.CreatedAt })
}

func sortWorksByTop(works []models.Work) {
	sortListingsByTop(works, func(w models.Work) string { return w.Top }, func(w models.Work) time.Time { return w.CreatedAt })
}

func sortFilteredWorksByTop(works []models.FilteredWork) {
	sortListingsByTop(works, func(w models.FilteredWork) string { return w.Top }, func(w models.FilteredWork) time.Time { return w.CreatedAt })
}

func sortWorkAdsByTop(works []models.WorkAd) {
	sortListingsByTop(works, func(w models.WorkAd) string { return w.Top }, func(w models.WorkAd) time.Time { return w.CreatedAt })
}

func sortFilteredWorkAdsByTop(works []models.FilteredWorkAd) {
	sortListingsByTop(works, func(w models.FilteredWorkAd) string { return w.Top }, func(w models.FilteredWorkAd) time.Time { return w.CreatedAt })
}

func sortRentsByTop(rents []models.Rent) {
	sortListingsByTop(rents, func(r models.Rent) string { return r.Top }, func(r models.Rent) time.Time { return r.CreatedAt })
}

func sortFilteredRentsByTop(rents []models.FilteredRent) {
	sortListingsByTop(rents, func(r models.FilteredRent) string { return r.Top }, func(r models.FilteredRent) time.Time { return r.CreatedAt })
}

func sortRentAdsByTop(rents []models.RentAd) {
	sortListingsByTop(rents, func(r models.RentAd) string { return r.Top }, func(r models.RentAd) time.Time { return r.CreatedAt })
}

func sortFilteredRentAdsByTop(rents []models.FilteredRentAd) {
	sortListingsByTop(rents, func(r models.FilteredRentAd) string { return r.Top }, func(r models.FilteredRentAd) time.Time { return r.CreatedAt })
}

func sortListingsByTop[T any](items []T, getTop func(item T) string, getCreatedAt func(item T) time.Time) {
	if len(items) < 2 {
		return
	}
	now := time.Now().UTC()
	sort.SliceStable(items, func(i, j int) bool {
		stateI := computeTopState(getTop(items[i]), now)
		stateJ := computeTopState(getTop(items[j]), now)
		createdAtI := getCreatedAt(items[i]).UTC()
		createdAtJ := getCreatedAt(items[j]).UTC()
		return lessByTopState(stateI, createdAtI, stateJ, createdAtJ)
	})
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

func liftListingsTopOnly[T any](items []T, getTop func(item T) string) {
	if len(items) < 2 {
		return
	}

	now := time.Now().UTC()

	sort.SliceStable(items, func(i, j int) bool {
		si := computeTopState(getTop(items[i]), now)
		sj := computeTopState(getTop(items[j]), now)

		// Единственное правило: активный TOP должен быть выше неактивного
		if si.active != sj.active {
			return si.active && !sj.active
		}

		// Всё остальное НЕ трогаем — сохраняем исходный порядок из SQL
		return false
	})
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
