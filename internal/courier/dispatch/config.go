package dispatch

import "time"

// ConfigAdapter bridges courier.Config with dispatcher.Config interface.
type ConfigAdapter struct {
	PricePerKM        int
	MinPrice          int
	SearchRadiusStart int
	SearchRadiusStep  int
	SearchRadiusMax   int
	DispatchTick      time.Duration
	OfferTTL          time.Duration
	SearchTimeout     time.Duration
	RegionKey         string
}

func (c ConfigAdapter) GetPricePerKM() int        { return c.PricePerKM }
func (c ConfigAdapter) GetMinPrice() int          { return c.MinPrice }
func (c ConfigAdapter) GetSearchRadiusStart() int { return c.SearchRadiusStart }
func (c ConfigAdapter) GetSearchRadiusStep() int  { return c.SearchRadiusStep }
func (c ConfigAdapter) GetSearchRadiusMax() int   { return c.SearchRadiusMax }
func (c ConfigAdapter) GetDispatchTick() time.Duration {
	return c.DispatchTick
}
func (c ConfigAdapter) GetOfferTTL() time.Duration      { return c.OfferTTL }
func (c ConfigAdapter) GetSearchTimeout() time.Duration { return c.SearchTimeout }
func (c ConfigAdapter) GetRegionKey() string            { return c.RegionKey }
