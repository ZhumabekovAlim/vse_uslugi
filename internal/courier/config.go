package courier

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPricePerKM        = 120
	defaultMinPrice          = 500
	defaultOfferTTL          = 15 * time.Minute
	defaultSearchTimeout     = 15 * time.Minute
	defaultSearchRadiusStart = 500
	defaultSearchRadiusStep  = 500
	defaultSearchRadiusMax   = 5000
	defaultDispatchTick      = 10 * time.Second
	defaultRedisCity         = "astana"
)

// Config holds runtime configuration for the courier module.
type Config struct {
	PricePerKM        int
	MinPrice          int
	OfferTTL          time.Duration
	SearchTimeout     time.Duration
	SearchRadiusStart int
	SearchRadiusStep  int
	SearchRadiusMax   int
	DispatchTick      time.Duration
	RedisCity         string
}

// LoadConfig reads courier configuration from environment variables and applies defaults.
func LoadConfig() (Config, error) {
	cfg := Config{
		PricePerKM:        defaultPricePerKM,
		MinPrice:          defaultMinPrice,
		OfferTTL:          defaultOfferTTL,
		SearchTimeout:     defaultSearchTimeout,
		SearchRadiusStart: defaultSearchRadiusStart,
		SearchRadiusStep:  defaultSearchRadiusStep,
		SearchRadiusMax:   defaultSearchRadiusMax,
		DispatchTick:      defaultDispatchTick,
		RedisCity:         defaultRedisCity,
	}

	if v, err := readIntEnv("COURIER_PRICE_PER_KM"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_PRICE_PER_KM: %w", err)
	} else if v != nil {
		cfg.PricePerKM = *v
	}

	if v, err := readIntEnv("COURIER_MIN_PRICE"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_MIN_PRICE: %w", err)
	} else if v != nil {
		cfg.MinPrice = *v
	}

	if v := os.Getenv("COURIER_OFFER_TTL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse COURIER_OFFER_TTL_SECONDS: %w", err)
		}
		cfg.OfferTTL = time.Duration(secs) * time.Second
	}

	if v := os.Getenv("COURIER_SEARCH_TIMEOUT_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse COURIER_SEARCH_TIMEOUT_SECONDS: %w", err)
		}
		cfg.SearchTimeout = time.Duration(secs) * time.Second
	} else if v := os.Getenv("COURIER_SEARCH_TTL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse COURIER_SEARCH_TTL_SECONDS: %w", err)
		}
		cfg.SearchTimeout = time.Duration(secs) * time.Second
	}

	if v, err := readIntEnv("COURIER_SEARCH_RADIUS_STEP"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_SEARCH_RADIUS_STEP: %w", err)
	} else if v != nil {
		cfg.SearchRadiusStep = *v
	}

	if v, err := readIntEnv("COURIER_SEARCH_RADIUS_MAX"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_SEARCH_RADIUS_MAX: %w", err)
	} else if v != nil {
		cfg.SearchRadiusMax = *v
	}

	if v := os.Getenv("COURIER_DISPATCH_TICK_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse COURIER_DISPATCH_TICK_SECONDS: %w", err)
		}
		cfg.DispatchTick = time.Duration(secs) * time.Second
	}

	if v := os.Getenv("COURIER_REDIS_CITY"); strings.TrimSpace(v) != "" {
		cfg.RedisCity = strings.ToLower(strings.TrimSpace(v))
	}

	if v, err := readIntEnv("COURIER_SEARCH_RADIUS_START"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_SEARCH_RADIUS_START: %w", err)
	} else if v != nil {
		cfg.SearchRadiusStart = *v
	}

	if v, err := readIntEnv("COURIER_SEARCH_RADIUS_START"); err != nil {
		return Config{}, fmt.Errorf("parse COURIER_SEARCH_RADIUS_START: %w", err)
	} else if v != nil {
		cfg.SearchRadiusStart = *v
	}

	if cfg.PricePerKM <= 0 {
		return Config{}, fmt.Errorf("COURIER_PRICE_PER_KM must be positive")
	}
	if cfg.MinPrice <= 0 {
		return Config{}, fmt.Errorf("COURIER_MIN_PRICE must be positive")
	}
	if cfg.OfferTTL <= 0 {
		return Config{}, fmt.Errorf("COURIER_OFFER_TTL_SECONDS must be positive")
	}
	if cfg.SearchTimeout <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_TIMEOUT_SECONDS must be positive")
	}
	if cfg.SearchRadiusStart <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_RADIUS_START must be positive")
	}
	if cfg.SearchRadiusStep <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_RADIUS_STEP must be positive")
	}
	if cfg.SearchRadiusMax <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_RADIUS_MAX must be positive")
	}
	if cfg.SearchRadiusStart > cfg.SearchRadiusMax {
		return Config{}, fmt.Errorf("COURIER_SEARCH_RADIUS_START must be <= COURIER_SEARCH_RADIUS_MAX")
	}
	if cfg.DispatchTick <= 0 {
		return Config{}, fmt.Errorf("COURIER_DISPATCH_TICK_SECONDS must be positive")
	}
	if cfg.SearchRadiusStart <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_RADIUS_START must be positive")
	}

	return cfg, nil
}

func readIntEnv(name string) (*int, error) {
	val := os.Getenv(name)
	if val == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
