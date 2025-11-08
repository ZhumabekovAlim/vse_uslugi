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
	// нормализуем город, чтобы везде один формат
	return fmt.Sprintf("drivers:%s:%s", strings.ToLower(city), status)
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

// ---- Debug helper -----------------------------------------------------------

func (l *DriverLocator) DebugDumpFree(ctx context.Context, city string) {
	key := redisKey(city, "free")
	members, _ := l.rdb.ZRange(ctx, key, 0, -1).Result()
	log.Printf("[debug] %s total=%d", key, len(members))
	for _, m := range members {
		pos, _ := l.rdb.GeoPos(ctx, key, m).Result()
		if len(pos) > 0 && pos[0] != nil {
			log.Printf("  %s lon=%.6f lat=%.6f", m, pos[0].Longitude, pos[0].Latitude)
		} else {
			log.Printf("  %s (no pos)", m)
		}
	}
}

// ---- Geo updates ------------------------------------------------------------

// СЫРОЙ низкоуровневый апдейт: ТОЛЬКО GeoAdd. Никакой валидации, никакой логики.
func (l *DriverLocator) updateDriverRaw(ctx context.Context, driverID int64, lon, lat float64, city, status string) error {
	key := redisKey(city, status)
	return l.rdb.GeoAdd(ctx, key, &redis.GeoLocation{
		Name:      memberName(driverID),
		Longitude: lon,
		Latitude:  lat,
	}).Err()
}

// БЕЗОПАСНЫЙ апдейт: валидация + лог, вызывает raw.
func (l *DriverLocator) SafeUpdateDriver(ctx context.Context, driverID int64, lon, lat float64, city, status string) error {
	city = strings.ToLower(strings.TrimSpace(city))
	if city == "" {
		return fmt.Errorf("SafeUpdateDriver: empty city")
	}
	if status == "" {
		status = "free"
	}
	if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return fmt.Errorf("SafeUpdateDriver: invalid coords lon=%.8f lat=%.8f", lon, lat)
	}
	if math.Abs(lon) < 1e-4 && math.Abs(lat) < 1e-4 {
		return fmt.Errorf("SafeUpdateDriver: near-zero coords lon=%.8f lat=%.8f", lon, lat)
	}

	if err := l.updateDriverRaw(ctx, driverID, lon, lat, city, status); err != nil {
		return err
	}
	log.Printf("GeoAdd OK driver=%d city=%s status=%s lon=%.6f lat=%.6f",
		driverID, city, status, lon, lat)
	return nil
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
			fmt.Println("redis Nearby: no free drivers found")
			return nil, nil
		}
		return nil, err
	}

	drivers := make([]NearbyDriver, 0, len(res))

	for _, item := range res {
		id, err := parseDriverMember(item.Name)
		if err != nil {
			fmt.Printf("redis Nearby: skip invalid member %s: %v\n", item.Name, err)
			continue
		}

		d := NearbyDriver{
			ID:   id,
			Dist: item.Dist,
			Lon:  item.Longitude,
			Lat:  item.Latitude,
		}
		drivers = append(drivers, d)

		fmt.Printf("  driverID=%d dist=%.1fm lon=%.6f lat=%.6f\n", d.ID, d.Dist, d.Lon, d.Lat)
	}

	return drivers, nil
}
