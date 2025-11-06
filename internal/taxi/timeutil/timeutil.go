package timeutil

import "time"

var almatyLocation = loadLocation()

func loadLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Asia/Almaty", 5*60*60)
	}
	return loc
}

// Now returns the current time in Asia/Almaty timezone.
func Now() time.Time {
	return time.Now().In(almatyLocation)
}

// InAlmaty converts provided time to Asia/Almaty timezone.
func InAlmaty(t time.Time) time.Time {
	return t.In(almatyLocation)
}

// Location returns Asia/Almaty location instance.
func Location() *time.Location {
	return almatyLocation
}
