package courier

import (
	"net/http"

	courierhttp "naimuBack/internal/courier/http"
	"naimuBack/internal/courier/repo"
)

// RegisterCourierRoutes wires the HTTP handlers into the provided mux.
func RegisterCourierRoutes(mux *http.ServeMux, deps *Deps) error {
	if err := deps.Validate(); err != nil {
		return err
	}

	ordersRepo := repo.NewOrdersRepo(deps.DB)
	offersRepo := repo.NewOffersRepo(deps.DB)

	httpCfg := courierhttp.Config{
		PricePerKM: deps.Config.PricePerKM,
		MinPrice:   deps.Config.MinPrice,
	}
	server := courierhttp.NewServer(httpCfg, deps.Logger, ordersRepo, offersRepo)
	server.Register(mux)
	return nil
}
