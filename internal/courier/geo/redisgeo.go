package geo

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// NearbyCourier represents a courier returned from Redis GEO queries.
type NearbyCourier struct {
	ID   int64
	Dist float64
	Lon  float64
	Lat  float64
}

// CourierLocator handles courier geo operations in Redis.
type CourierLocator struct {
	rdb *redis.Client
}

// NewCourierLocator creates a new locator.
func NewCourierLocator(rdb *redis.Client) *CourierLocator {
	return &CourierLocator{rdb: rdb}
}

func redisKey(city, status string) string {
	return fmt.Sprintf("couriers:%s:%s", strings.ToLower(city), status)
}

func memberName(courierID int64) string {
	return fmt.Sprintf("courier:%d", courierID)
}

func parseCourierMember(member string) (int64, error) {
	parts := strings.Split(member, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid member %q", member)
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

// SafeUpdateCourier validates input and updates courier location in Redis GEO set.
func (l *CourierLocator) SafeUpdateCourier(ctx context.Context, courierID int64, lon, lat float64, city, status string) error {
	city = strings.ToLower(strings.TrimSpace(city))
	if city == "" {
		return fmt.Errorf("SafeUpdateCourier: empty city")
	}
	if status == "" {
		status = "free"
	}
	if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return fmt.Errorf("SafeUpdateCourier: invalid coords lon=%.8f lat=%.8f", lon, lat)
	}
	if math.Abs(lon) < 1e-4 && math.Abs(lat) < 1e-4 {
		return fmt.Errorf("SafeUpdateCourier: near-zero coords lon=%.8f lat=%.8f", lon, lat)
	}

	key := redisKey(city, status)
	if err := l.rdb.GeoAdd(ctx, key, &redis.GeoLocation{Name: memberName(courierID), Longitude: lon, Latitude: lat}).Err(); err != nil {
		return err
	}
	log.Printf("Courier GeoAdd OK courier=%d city=%s status=%s lon=%.6f lat=%.6f", courierID, city, status, lon, lat)
	return nil
}

// MoveCourier moves a courier between status sets, preserving coordinates.
func (l *CourierLocator) MoveCourier(ctx context.Context, courierID int64, city, fromStatus, toStatus string) error {
	if fromStatus == toStatus {
		return nil
	}
	src := redisKey(city, fromStatus)
	dst := redisKey(city, toStatus)
	mem := memberName(courierID)

	pos, err := l.rdb.GeoPos(ctx, src, mem).Result()
	if err != nil {
		return err
	}
	if len(pos) == 0 || pos[0] == nil {
		return fmt.Errorf("MoveCourier: coordinates not found for %s in %s", mem, src)
	}
	lon := pos[0].Longitude
	lat := pos[0].Latitude

	if err := l.rdb.GeoAdd(ctx, dst, &redis.GeoLocation{Name: mem, Longitude: lon, Latitude: lat}).Err(); err != nil {
		return err
	}
	if err := l.rdb.ZRem(ctx, src, mem).Err(); err != nil {
		return err
	}
	return nil
}

// GoOffline removes courier from all status sets in a city.
func (l *CourierLocator) GoOffline(ctx context.Context, courierID int64, city string) error {
	mem := memberName(courierID)
	statuses := []string{"free", "busy"}
	for _, st := range statuses {
		if err := l.rdb.ZRem(ctx, redisKey(city, st), mem).Err(); err != nil {
			return err
		}
	}
	return nil
}

// Nearby returns couriers within radius sorted by distance.
func (l *CourierLocator) Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]NearbyCourier, error) {
	key := redisKey(city, "free")
	res, err := l.rdb.GeoSearchLocation(ctx, key, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lon,
			Latitude:   lat,
			Radius:     radiusMeters,
			RadiusUnit: "m",
			Sort:       "ASC",
			Count:      limit,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	couriers := make([]NearbyCourier, 0, len(res))
	for _, item := range res {
		id, err := parseCourierMember(item.Name)
		if err != nil {
			log.Printf("Courier Nearby: skip invalid member %s: %v", item.Name, err)
			continue
		}
		couriers = append(couriers, NearbyCourier{
			ID:   id,
			Dist: item.Dist,
			Lon:  item.Longitude,
			Lat:  item.Latitude,
		})
	}
	return couriers, nil
}
