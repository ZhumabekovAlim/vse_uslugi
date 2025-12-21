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
	liftListingsTopOnly(services, func(s models.Service) string { return s.Top })
}

func sortFilteredServicesByTop(services []models.FilteredService) {
	liftListingsTopOnly(services, func(s models.FilteredService) string { return s.Top })
}

func sortAdsByTop(ads []models.Ad) {
	liftListingsTopOnly(ads, func(a models.Ad) string { return a.Top })
}

func sortFilteredAdsByTop(ads []models.FilteredAd) {
	liftListingsTopOnly(ads, func(a models.FilteredAd) string { return a.Top })
}

func sortWorksByTop(works []models.Work) {
	liftListingsTopOnly(works, func(w models.Work) string { return w.Top })
}

func sortFilteredWorksByTop(works []models.FilteredWork) {
	liftListingsTopOnly(works, func(w models.FilteredWork) string { return w.Top })
}

func sortWorkAdsByTop(works []models.WorkAd) {
	liftListingsTopOnly(works, func(w models.WorkAd) string { return w.Top })
}

func sortFilteredWorkAdsByTop(works []models.FilteredWorkAd) {
	liftListingsTopOnly(works, func(w models.FilteredWorkAd) string { return w.Top })
}

func sortRentsByTop(rents []models.Rent) {
	liftListingsTopOnly(rents, func(r models.Rent) string { return r.Top })
}

func sortFilteredRentsByTop(rents []models.FilteredRent) {
	liftListingsTopOnly(rents, func(r models.FilteredRent) string { return r.Top })
}

func sortRentAdsByTop(rents []models.RentAd) {
	liftListingsTopOnly(rents, func(r models.RentAd) string { return r.Top })
}

func sortFilteredRentAdsByTop(rents []models.FilteredRentAd) {
	liftListingsTopOnly(rents, func(r models.FilteredRentAd) string { return r.Top })
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
