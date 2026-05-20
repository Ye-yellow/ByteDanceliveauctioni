package server

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(addr string, auction *appsvc.AuctionService, hub *realtime.Hub, health HealthChecker) *httptransport.Server {
	srv := httptransport.NewServer(
		httptransport.Address(addr),
		httptransport.Middleware(recovery.Recovery()),
	)

	registerAuctionHTTP(srv, auction)
	registerRealtimeHTTP(srv, hub)
	registerOperationHTTP(srv, health)

	return srv
}

func registerAuctionHTTP(srv *httptransport.Server, auction *appsvc.AuctionService) {
	v1.RegisterAuctionServiceHTTPServer(srv, auction)
}
