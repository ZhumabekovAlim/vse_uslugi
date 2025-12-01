package repositories

import (
	"math"
	"strconv"
)

func calculateDistanceKm(userLat, userLon *float64, listingLat, listingLon *string) *float64 {
	if userLat == nil || userLon == nil || listingLat == nil || listingLon == nil {
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

	distance := haversineDistanceKm(*userLat, *userLon, latValue, lonValue)
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
