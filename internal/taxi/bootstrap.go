package taxi

import (
	"context"
	"net/http"
	"time"

	"naimuBack/internal/taxi/dispatch"
	"naimuBack/internal/taxi/geo"
	taxihttp "naimuBack/internal/taxi/http"
	"naimuBack/internal/taxi/pay"
	"naimuBack/internal/taxi/repo"
	"naimuBack/internal/taxi/timeutil"
	"naimuBack/internal/taxi/ws"
)

type moduleState struct {
	geoClient     *geo.DGISClient
	locator       *geo.DriverLocator
	driversRepo   *repo.DriversRepo
	ordersRepo    *repo.OrdersRepo
	intercityRepo *repo.IntercityOrdersRepo
	dispatchRepo  *repo.DispatchRepo
	offersRepo    *repo.OffersRepo
	paymentsRepo  *repo.PaymentsRepo
	driverHub     *ws.DriverHub
	passengerHub  *ws.PassengerHub
	dispatcher    *dispatch.Dispatcher
	server        *taxihttp.Server
	payClient     *pay.Client
	cfgAdapter    dispatch.ConfigAdapter
}

func ensureModule(deps *TaxiDeps) (*moduleState, error) {
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
		RegionID:          deps.Config.DGISRegionID,
		SearchTimeout:     deps.Config.SearchTimeout,
	}

	geoClient := geo.NewDGISClient(deps.HTTPClient, deps.Config.DGISAPIKey, deps.Config.DGISRegionID)
	locator := geo.NewDriverLocator(deps.RDB)
	driverHub := ws.NewDriverHub(locator, deps.Logger)
	passengerHub := ws.NewPassengerHub(deps.Logger)

	driversRepo := repo.NewDriversRepo(deps.DB)
	ordersRepo := repo.NewOrdersRepo(deps.DB)
	intercityRepo := repo.NewIntercityOrdersRepo(deps.DB)
	dispatchRepo := repo.NewDispatchRepo(deps.DB)
	offersRepo := repo.NewOffersRepo(deps.DB)
	paymentsRepo := repo.NewPaymentsRepo(deps.DB)

	dispatcher := dispatch.New(ordersRepo, dispatchRepo, offersRepo, locator, driverHub, passengerHub, deps.Logger, cfgAdapter)
	payClient := pay.NewClient(deps.HTTPClient, deps.Config.AirbaPayMerchant, deps.Config.AirbaPaySecret, deps.Config.AirbaPayCallback)
	server := taxihttp.NewServer(deps.Logger, cfgAdapter, geoClient, driversRepo, ordersRepo, intercityRepo, offersRepo, paymentsRepo, driverHub, passengerHub, dispatcher, payClient)

	deps.module = &moduleState{
		geoClient:     geoClient,
		locator:       locator,
		driversRepo:   driversRepo,
		ordersRepo:    ordersRepo,
		intercityRepo: intercityRepo,
		dispatchRepo:  dispatchRepo,
		offersRepo:    offersRepo,
		paymentsRepo:  paymentsRepo,
		driverHub:     driverHub,
		passengerHub:  passengerHub,
		dispatcher:    dispatcher,
		server:        server,
		payClient:     payClient,
		cfgAdapter:    cfgAdapter,
	}
	return deps.module, nil
}

// RegisterTaxiRoutes wires HTTP and WebSocket routes into the provided mux.
func RegisterTaxiRoutes(mux *http.ServeMux, deps *TaxiDeps) error {
	module, err := ensureModule(deps)
	if err != nil {
		return err
	}
	module.server.RegisterRoutes(mux)
	return nil
}

// StartTaxiWorkers launches background workers for dispatcher and maintenance.
func StartTaxiWorkers(ctx context.Context, deps *TaxiDeps) error {
	module, err := ensureModule(deps)
	if err != nil {
		return err
	}
	go module.dispatcher.Run(ctx)
	go module.startOfferCleanup(ctx)
	return nil
}

func (m *moduleState) startOfferCleanup(ctx context.Context) {
	ticker := time.NewTicker(m.cfgAdapter.OfferTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.offersRepo.ExpireOffers(ctx, timeutil.Now())
		}
	}
}
