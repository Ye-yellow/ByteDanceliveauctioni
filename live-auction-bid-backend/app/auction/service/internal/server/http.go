package server

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(addr string, auction *appsvc.AuctionService, hub *realtime.Hub) *httptransport.Server {
	srv := httptransport.NewServer(httptransport.Address(addr))

	registerAuctionHTTP(srv, auction)
	registerRealtimeHTTP(srv, hub)
	registerOperationHTTP(srv)

	return srv
}

func registerAuctionHTTP(srv *httptransport.Server, auction *appsvc.AuctionService) {
	v1.RegisterAuctionServiceHTTPServer(srv, auction)
}
