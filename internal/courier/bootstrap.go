package courier

import (
	"net/http"

	courierhttp "naimuBack/internal/courier/http"
	"naimuBack/internal/courier/repo"
	"naimuBack/internal/courier/ws"
)

// RegisterCourierRoutes wires the HTTP handlers into the provided mux.
func RegisterCourierRoutes(mux *http.ServeMux, deps *Deps) error {
	if err := deps.Validate(); err != nil {
		return err
	}

	ordersRepo := repo.NewOrdersRepo(deps.DB)
	offersRepo := repo.NewOffersRepo(deps.DB)
	couriersRepo := repo.NewCouriersRepo(deps.DB)
	usersRepo := repo.NewUsersRepo(deps.DB)

	courierHub := deps.CourierHub
	if courierHub == nil {
		courierHub = ws.NewCourierHub(deps.Logger)
		deps.CourierHub = courierHub
	}
	senderHub := deps.SenderHub
	if senderHub == nil {
		senderHub = ws.NewSenderHub(deps.Logger)
		deps.SenderHub = senderHub
	}

	httpCfg := courierhttp.Config{
		PricePerKM:        deps.Config.PricePerKM,
		MinPrice:          deps.Config.MinPrice,
		SearchRadiusStart: deps.Config.SearchRadiusStart,
	}
	server := courierhttp.NewServer(httpCfg, deps.Logger, ordersRepo, offersRepo, couriersRepo, usersRepo, courierHub, senderHub)
	server.Register(mux)
	return nil
}
