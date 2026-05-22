package server

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
	"live-auction-bid/backend/app/auction/service/internal/storage"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(addr string, auction *appsvc.AuctionService, users *appsvc.UserService, hub *realtime.Hub, health HealthChecker, authManager *auth.Manager, authMiddleware middleware.Middleware, imageStorage storage.StorageProvider, assetStore assetStore) *httptransport.Server {
	middlewares := []middleware.Middleware{recovery.Recovery()}
	if authMiddleware != nil {
		middlewares = append(middlewares, authMiddleware)
	}
	srv := httptransport.NewServer(
		httptransport.Address(addr),
		httptransport.Middleware(middlewares...),
	)

	registerAuctionHTTP(srv, auction)
	registerUserHTTP(srv, users)
	registerRealtimeHTTP(srv, hub)
	registerUploadHTTP(srv, authManager, assetStore, imageStorage)
	registerOperationHTTP(srv, health)

	return srv
}

func registerAuctionHTTP(srv *httptransport.Server, auction *appsvc.AuctionService) {
	v1.RegisterAuctionServiceHTTPServer(srv, auction)
}

func registerUserHTTP(srv *httptransport.Server, users *appsvc.UserService) {
	v1.RegisterUserServiceHTTPServer(srv, users)
}
