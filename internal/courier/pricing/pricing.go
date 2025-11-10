package pricing

// Recommended calculates the suggested delivery price based on distance and configuration.
func Recommended(distanceMeters, pricePerKM, minPrice int) int {
	if distanceMeters <= 0 {
		if minPrice > 0 {
			return minPrice
		}
		return 0
	}
	price := (distanceMeters * pricePerKM) / 1000
	if price < minPrice {
		return minPrice
	}
	return price
}
