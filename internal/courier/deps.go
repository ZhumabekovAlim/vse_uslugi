package courier

import (
	"database/sql"
	"fmt"
	"net/http"

	"naimuBack/internal/courier/ws"
)

// Logger is the minimal logging interface required by the courier module.
type Logger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

// Deps aggregates runtime dependencies for the courier module.
type Deps struct {
	DB         *sql.DB
	Logger     Logger
	Config     Config
	HTTPClient *http.Client
	CourierHub *ws.CourierHub
	SenderHub  *ws.SenderHub
}

// Validate ensures that the deps struct contains the essentials before bootstrapping services.
func (d *Deps) Validate() error {
	if d == nil {
		return fmt.Errorf("courier deps are nil")
	}
	if d.DB == nil {
		return fmt.Errorf("courier deps DB is required")
	}
	if d.Logger == nil {
		return fmt.Errorf("courier deps Logger is required")
	}
	if d.HTTPClient == nil {
		d.HTTPClient = &http.Client{}
	}
	return nil
}
