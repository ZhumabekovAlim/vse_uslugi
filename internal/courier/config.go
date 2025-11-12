package courier

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultPricePerKM        = 120
	defaultMinPrice          = 500
	defaultOfferTTL          = 15 * time.Minute
	defaultSearchTTL         = 15 * time.Minute
	defaultSearchRadiusStart = 500
)

// Config holds runtime configuration for the courier module.
type Config struct {
	PricePerKM        int
	MinPrice          int
	OfferTTL          time.Duration
	SearchTTL         time.Duration
	SearchRadiusStart int
}

// LoadConfig reads courier configuration from environment variables and applies defaults.
func LoadConfig() (Config, error) {
	cfg := Config{
		PricePerKM:        defaultPricePerKM,
		MinPrice:          defaultMinPrice,
		OfferTTL:          defaultOfferTTL,
		SearchTTL:         defaultSearchTTL,
		SearchRadiusStart: defaultSearchRadiusStart,
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

	if v := os.Getenv("COURIER_SEARCH_TTL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse COURIER_SEARCH_TTL_SECONDS: %w", err)
		}
		cfg.SearchTTL = time.Duration(secs) * time.Second
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
	if cfg.SearchTTL <= 0 {
		return Config{}, fmt.Errorf("COURIER_SEARCH_TTL_SECONDS must be positive")
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
