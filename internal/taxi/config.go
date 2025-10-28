package taxi

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultPricePerKM        = 300
	defaultMinPrice          = 1200
	defaultSearchRadiusStart = 800
	defaultSearchRadiusStep  = 400
	defaultSearchRadiusMax   = 3000
	defaultDispatchTick      = 10 * time.Second
	defaultOfferTTL          = 20 * time.Second
)

// TaxiConfig holds runtime configuration for the Taxi module.
type TaxiConfig struct {
	PricePerKM        int
	MinPrice          int
	SearchRadiusStart int
	SearchRadiusStep  int
	SearchRadiusMax   int
	DispatchTick      time.Duration
	OfferTTL          time.Duration
	DGISAPIKey        string
	DGISRegionID      string
	AirbaPayMerchant  string
	AirbaPaySecret    string
	AirbaPayCallback  string
}

// LoadTaxiConfig reads configuration from environment variables and applies defaults.
func LoadTaxiConfig() (TaxiConfig, error) {
	cfg := TaxiConfig{
		PricePerKM:        defaultPricePerKM,
		MinPrice:          defaultMinPrice,
		SearchRadiusStart: defaultSearchRadiusStart,
		SearchRadiusStep:  defaultSearchRadiusStep,
		SearchRadiusMax:   defaultSearchRadiusMax,
		DispatchTick:      defaultDispatchTick,
		OfferTTL:          defaultOfferTTL,
	}

	if v, err := readIntEnv("PRICE_PER_KM"); err != nil {
		return TaxiConfig{}, fmt.Errorf("parse PRICE_PER_KM: %w", err)
	} else if v != nil {
		cfg.PricePerKM = *v
	}

	if v, err := readIntEnv("MIN_PRICE"); err != nil {
		return TaxiConfig{}, fmt.Errorf("parse MIN_PRICE: %w", err)
	} else if v != nil {
		cfg.MinPrice = *v
	}

	if v, err := readIntEnv("SEARCH_RADIUS_START"); err != nil {
		return TaxiConfig{}, fmt.Errorf("parse SEARCH_RADIUS_START: %w", err)
	} else if v != nil {
		cfg.SearchRadiusStart = *v
	}

	if v, err := readIntEnv("SEARCH_RADIUS_STEP"); err != nil {
		return TaxiConfig{}, fmt.Errorf("parse SEARCH_RADIUS_STEP: %w", err)
	} else if v != nil {
		cfg.SearchRadiusStep = *v
	}

	if v, err := readIntEnv("SEARCH_RADIUS_MAX"); err != nil {
		return TaxiConfig{}, fmt.Errorf("parse SEARCH_RADIUS_MAX: %w", err)
	} else if v != nil {
		cfg.SearchRadiusMax = *v
	}

	if v := os.Getenv("DISPATCH_TICK_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return TaxiConfig{}, fmt.Errorf("parse DISPATCH_TICK_SECONDS: %w", err)
		}
		cfg.DispatchTick = time.Duration(secs) * time.Second
	}

	if v := os.Getenv("OFFER_TTL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return TaxiConfig{}, fmt.Errorf("parse OFFER_TTL_SECONDS: %w", err)
		}
		cfg.OfferTTL = time.Duration(secs) * time.Second
	}

	cfg.DGISAPIKey = os.Getenv("DGIS_API_KEY")
	if cfg.DGISAPIKey == "" {
		return TaxiConfig{}, fmt.Errorf("DGIS_API_KEY is required")
	}

	cfg.DGISRegionID = os.Getenv("DGIS_REGION_ID")

	cfg.AirbaPayMerchant = os.Getenv("AIRBAPAY_MERCHANT_ID")
	cfg.AirbaPaySecret = os.Getenv("AIRBAPAY_SECRET")
	cfg.AirbaPayCallback = os.Getenv("AIRBAPAY_CALLBACK_URL")

	if cfg.AirbaPayMerchant == "" || cfg.AirbaPaySecret == "" || cfg.AirbaPayCallback == "" {
		return TaxiConfig{}, fmt.Errorf("AIRBAPAY configuration incomplete")
	}

	if cfg.SearchRadiusStart <= 0 || cfg.SearchRadiusStep <= 0 || cfg.SearchRadiusMax <= 0 {
		return TaxiConfig{}, fmt.Errorf("search radius values must be positive")
	}
	if cfg.SearchRadiusStart > cfg.SearchRadiusMax {
		return TaxiConfig{}, fmt.Errorf("SEARCH_RADIUS_START must be <= SEARCH_RADIUS_MAX")
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
