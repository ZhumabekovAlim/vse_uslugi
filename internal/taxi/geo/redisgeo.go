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

// UpdateDriver stores driver coordinates in the appropriate GEO set.
func (l *DriverLocator) UpdateDriver(ctx context.Context, driverID int64, lon, lat float64, city, status string) error {
    key := redisKey(city, status)
    member := fmt.Sprintf("driver:%d", driverID)
    return l.rdb.GeoAdd(ctx, key, &redis.GeoLocation{Longitude: lon, Latitude: lat, Name: member}).Err()
}

// MoveDriver moves a driver between status sets.
func (l *DriverLocator) MoveDriver(ctx context.Context, driverID int64, city, fromStatus, toStatus string) error {
    member := fmt.Sprintf("driver:%d", driverID)
    if err := l.rdb.ZRem(ctx, redisKey(city, fromStatus), member).Err(); err != nil {
        return err
    }
    return l.rdb.GeoAdd(ctx, redisKey(city, toStatus), &redis.GeoLocation{Name: member}).Err()
}

// Nearby returns drivers within radius sorted by distance.
func (l *DriverLocator) Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]NearbyDriver, error) {
    res, err := l.rdb.GeoSearchLocation(ctx, redisKey(city, "free"), &redis.GeoSearchLocationQuery{
        GeoSearchQuery: redis.GeoSearchQuery{
            Radius:      radiusMeters,
            RadiusUnit:  "m",
            Sort:        "ASC",
            Count:       limit,
            WithCoord:   true,
            WithDist:    true,
            Longitude:   lon,
            Latitude:    lat,
            FromLonLat:  true,
        },
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
        drivers = append(drivers, NearbyDriver{ID: id, Dist: item.Dist, Lon: item.Longitude, Lat: item.Latitude})
    }
    return drivers, nil
}

func parseDriverMember(member string) (int64, error) {
    parts := strings.Split(member, ":")
    if len(parts) != 2 {
        return 0, fmt.Errorf("invalid member")
    }
    return strconv.ParseInt(parts[1], 10, 64)
}
