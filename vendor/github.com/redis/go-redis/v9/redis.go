package redis

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
)

// GeoLocation represents a member in GEO sets.
type GeoLocation struct {
	Name      string
	Longitude float64
	Latitude  float64
	Dist      float64
}

// GeoSearchQuery defines search parameters.
type GeoSearchQuery struct {
	Radius     float64
	RadiusUnit string
	Sort       string
	Count      int
	WithCoord  bool
	WithDist   bool
	Longitude  float64
	Latitude   float64
	FromLonLat bool
}

// GeoSearchLocationQuery wraps GeoSearchQuery for WithCoord.
type GeoSearchLocationQuery struct {
	GeoSearchQuery
}

// Client is a lightweight in-memory substitute for go-redis client.
type Client struct {
	mu   sync.RWMutex
	data map[string]map[string]GeoLocation
}

// Options mimics redis options.
type Options struct {
	Addr string
}

// NewClient creates a new client stub.
func NewClient(opt *Options) *Client {
	return &Client{data: make(map[string]map[string]GeoLocation)}
}

// Close closes the client.
func (c *Client) Close() error { return nil }

// IntCmd mimics redis command returning integer.
type IntCmd struct {
	err error
}

// Err returns error.
func (c *IntCmd) Err() error { return c.err }

// GeoLocationCmd returns locations.
type GeoLocationCmd struct {
	val []GeoLocation
	err error
}

// Result returns locations and error.
func (c *GeoLocationCmd) Result() ([]GeoLocation, error) { return c.val, c.err }

// Nil mimics redis.Nil sentinel error.
var Nil = errors.New("redis: nil")

// GeoAdd adds locations to key.
func (c *Client) GeoAdd(ctx context.Context, key string, locations ...*GeoLocation) *IntCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	bucket, ok := c.data[key]
	if !ok {
		bucket = make(map[string]GeoLocation)
		c.data[key] = bucket
	}
	for _, loc := range locations {
		existing := bucket[loc.Name]
		if loc.Latitude == 0 && loc.Longitude == 0 && existing.Name != "" {
			continue
		}
		bucket[loc.Name] = GeoLocation{Name: loc.Name, Longitude: loc.Longitude, Latitude: loc.Latitude}
	}
	return &IntCmd{}
}

// ZRem removes members from sorted set.
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) *IntCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	bucket, ok := c.data[key]
	if !ok {
		return &IntCmd{}
	}
	for _, m := range members {
		name, _ := m.(string)
		delete(bucket, name)
	}
	return &IntCmd{}
}

// GeoSearchLocation searches nearest drivers.
func (c *Client) GeoSearchLocation(ctx context.Context, key string, query *GeoSearchLocationQuery) *GeoLocationCmd {
	if query == nil {
		return &GeoLocationCmd{err: errors.New("query required")}
	}
	c.mu.RLock()
	bucket := c.data[key]
	c.mu.RUnlock()
	if len(bucket) == 0 {
		return &GeoLocationCmd{val: []GeoLocation{}}
	}
	centerLon := query.Longitude
	centerLat := query.Latitude
	radius := query.Radius
	if query.RadiusUnit == "km" {
		radius *= 1000
	}

	res := make([]GeoLocation, 0, len(bucket))
	for _, loc := range bucket {
		dist := haversine(centerLat, centerLon, loc.Latitude, loc.Longitude)
		if radius > 0 && dist > radius {
			continue
		}
		loc.Dist = dist
		res = append(res, loc)
	}

	if query.Sort == "ASC" {
		sort.Slice(res, func(i, j int) bool { return res[i].Dist < res[j].Dist })
	} else if query.Sort == "DESC" {
		sort.Slice(res, func(i, j int) bool { return res[i].Dist > res[j].Dist })
	}

	if query.Count > 0 && len(res) > query.Count {
		res = res[:query.Count]
	}
	return &GeoLocationCmd{val: res}
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func toRadians(deg float64) float64 { return deg * math.Pi / 180 }
