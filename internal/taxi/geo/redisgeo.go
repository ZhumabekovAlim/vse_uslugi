package geo

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// NearbyDriver represents a driver returned from Redis GEO queries.
type NearbyDriver struct {
	ID   int64
	Dist float64
	Lon  float64
	Lat  float64
}

// DriverLocator handles driver geo operations.
type DriverLocator struct {
	rdb *redis.Client
}

// NewDriverLocator creates a new locator.
func NewDriverLocator(rdb *redis.Client) *DriverLocator {
	return &DriverLocator{rdb: rdb}
}

func redisKey(city, status string) string {
	return fmt.Sprintf("drivers:%s:%s", city, status)
}

func memberName(driverID int64) string {
	return fmt.Sprintf("driver:%d", driverID)
}

func parseDriverMember(member string) (int64, error) {
	parts := strings.Split(member, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid member %q", member)
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

// UpdateDriver stores/updates driver coordinates in the appropriate GEO set.
func (l *DriverLocator) UpdateDriver(ctx context.Context, driverID int64, lon, lat float64, city, status string) error {
	key := redisKey(city, status)
	return l.rdb.GeoAdd(ctx, key, &redis.GeoLocation{
		Name:      memberName(driverID),
		Longitude: lon,
		Latitude:  lat,
	}).Err()
}

// MoveDriver moves a driver between status sets, preserving coordinates.
func (l *DriverLocator) MoveDriver(ctx context.Context, driverID int64, city, fromStatus, toStatus string) error {
	if fromStatus == toStatus {
		return nil
	}
	src := redisKey(city, fromStatus)
	dst := redisKey(city, toStatus)
	mem := memberName(driverID)

	// 1) Берём координаты из исходного ключа.
	pos, err := l.rdb.GeoPos(ctx, src, mem).Result()
	if err != nil {
		return err
	}
	if len(pos) == 0 || pos[0] == nil {
		// Если нет в src — ничего страшного: не падаем, но без координат не можем добавить.
		// Можно вернуть ошибку:
		return fmt.Errorf("MoveDriver: coordinates not found for %s in %s", mem, src)
	}
	lon := pos[0].Longitude
	lat := pos[0].Latitude

	// 2) Добавляем в новый ключ с теми же координатами.
	if err := l.rdb.GeoAdd(ctx, dst, &redis.GeoLocation{
		Name:      mem,
		Longitude: lon,
		Latitude:  lat,
	}).Err(); err != nil {
		return err
	}

	// 3) Удаляем из старого ключа.
	if err := l.rdb.ZRem(ctx, src, mem).Err(); err != nil {
		return err
	}
	return nil
}

// GoOffline удаляет водителя из всех статусных наборов конкретного города.
func (l *DriverLocator) GoOffline(ctx context.Context, driverID int64, city string) error {
	mem := memberName(driverID)
	// если статусов у тебя больше — просто добавь сюда.
	statuses := []string{"free", "busy"}
	for _, st := range statuses {
		if err := l.rdb.ZRem(ctx, redisKey(city, st), mem).Err(); err != nil {
			return err
		}
	}
	return nil
}

// Nearby returns drivers within radius sorted by distance (ascending).
func (l *DriverLocator) Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]NearbyDriver, error) {
	key := redisKey(city, "free")
	// временный лог:
	fmt.Printf("redis Nearby key=%s radius=%.0fm lon=%.6f lat=%.6f\n", key, radiusMeters, lon, lat)

	res, err := l.rdb.GeoSearchLocation(ctx, redisKey(city, "free"), &redis.GeoSearchLocationQuery{
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

	drivers := make([]NearbyDriver, 0, len(res))
	for _, item := range res {
		id, err := parseDriverMember(item.Name)
		if err != nil {
			continue
		}
		drivers = append(drivers, NearbyDriver{
			ID:   id,
			Dist: item.Dist, // метры (см. RadiusUnit: "m")
			Lon:  item.Longitude,
			Lat:  item.Latitude,
		})
	}
	return drivers, nil
}
