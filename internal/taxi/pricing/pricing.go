package pricing

import "math"

// Recommended calculates the recommended price for a ride based on distance and pricing rules.
func Recommended(distanceMeters, pricePerKM, minPrice int) int {
    if distanceMeters < 0 {
        distanceMeters = 0
    }
    km := float64(distanceMeters) / 1000.0
    price := int(math.Round(km * float64(pricePerKM)))
    if price < minPrice {
        return minPrice
    }
    return price
}
