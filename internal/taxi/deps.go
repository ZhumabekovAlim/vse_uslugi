package taxi

import (
    "database/sql"
    "errors"
    "net/http"

    "github.com/redis/go-redis/v9"
)

// Logger provides minimal logging required by the Taxi module.
type Logger interface {
    Infof(format string, args ...interface{})
    Errorf(format string, args ...interface{})
}

// TaxiDeps groups external dependencies needed by the Taxi module.
type TaxiDeps struct {
    DB         *sql.DB
    RDB        *redis.Client
    Logger     Logger
    Config     TaxiConfig
    HTTPClient *http.Client
    module     *moduleState
}

// Validate ensures required dependencies are provided.
func (d *TaxiDeps) Validate() error {
    if d.DB == nil {
        return errors.New("taxi deps: DB is required")
    }
    if d.RDB == nil {
        return errors.New("taxi deps: RDB is required")
    }
    if d.Logger == nil {
        return errors.New("taxi deps: Logger is required")
    }
    if d.HTTPClient == nil {
        d.HTTPClient = http.DefaultClient
    }
    return nil
}
