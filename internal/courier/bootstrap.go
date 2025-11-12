package courier

import (
	"context"
	"net/http"

	"naimuBack/internal/courier/dispatch"
	"naimuBack/internal/courier/geo"
	courierhttp "naimuBack/internal/courier/http"
	"naimuBack/internal/courier/repo"
	"naimuBack/internal/courier/ws"
)

type moduleState struct {
	locator      *geo.CourierLocator
	ordersRepo   *repo.OrdersRepo
	offersRepo   *repo.OffersRepo
	couriersRepo *repo.CouriersRepo
	usersRepo    *repo.UsersRepo
	dispatchRepo *repo.DispatchRepo
	courierHub   *ws.CourierHub
	senderHub    *ws.SenderHub
	dispatcher   *dispatch.Dispatcher
	server       *courierhttp.Server
	cfgAdapter   dispatch.ConfigAdapter
}

func ensureModule(deps *Deps) (*moduleState, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	if deps.module != nil {
		return deps.module, nil
	}

	cfgAdapter := dispatch.ConfigAdapter{
		PricePerKM:        deps.Config.PricePerKM,
		MinPrice:          deps.Config.MinPrice,
		SearchRadiusStart: deps.Config.SearchRadiusStart,
		SearchRadiusStep:  deps.Config.SearchRadiusStep,
		SearchRadiusMax:   deps.Config.SearchRadiusMax,
		DispatchTick:      deps.Config.DispatchTick,
		OfferTTL:          deps.Config.OfferTTL,
		SearchTimeout:     deps.Config.SearchTimeout,
		RegionKey:         deps.Config.RedisCity,
	}

	locator := geo.NewCourierLocator(deps.RDB)
	courierHub := deps.CourierHub
	if courierHub == nil {
		courierHub = ws.NewCourierHub(locator, deps.Logger)
	}
	senderHub := deps.SenderHub
	if senderHub == nil {
		senderHub = ws.NewSenderHub(deps.Logger)
	}

	ordersRepo := repo.NewOrdersRepo(deps.DB)
	offersRepo := repo.NewOffersRepo(deps.DB)
	couriersRepo := repo.NewCouriersRepo(deps.DB)
	usersRepo := repo.NewUsersRepo(deps.DB)
	dispatchRepo := repo.NewDispatchRepo(deps.DB)

	dispatcher := dispatch.New(ordersRepo, dispatchRepo, offersRepo, locator, courierHub, senderHub, deps.Logger, cfgAdapter)
	httpCfg := courierhttp.Config{
		PricePerKM:        deps.Config.PricePerKM,
		MinPrice:          deps.Config.MinPrice,
		SearchRadiusStart: deps.Config.SearchRadiusStart,
	}
	server := courierhttp.NewServer(httpCfg, deps.Logger, ordersRepo, offersRepo, couriersRepo, usersRepo, courierHub, senderHub, dispatcher)

	deps.module = &moduleState{
		locator:      locator,
		ordersRepo:   ordersRepo,
		offersRepo:   offersRepo,
		couriersRepo: couriersRepo,
		usersRepo:    usersRepo,
		dispatchRepo: dispatchRepo,
		courierHub:   courierHub,
		senderHub:    senderHub,
		dispatcher:   dispatcher,
		server:       server,
		cfgAdapter:   cfgAdapter,
	}
	deps.CourierHub = courierHub
	deps.SenderHub = senderHub
	return deps.module, nil
}

// RegisterCourierRoutes wires the HTTP handlers into the provided mux.
func RegisterCourierRoutes(mux *http.ServeMux, deps *Deps) error {
	module, err := ensureModule(deps)
	if err != nil {
		return err
	}
	module.server.Register(mux)
	return nil
}

// StartCourierWorkers launches background dispatcher loop for courier orders.
func StartCourierWorkers(ctx context.Context, deps *Deps) error {
	module, err := ensureModule(deps)
	if err != nil {
		return err
	}
	go module.dispatcher.Run(ctx)
	return nil
}
