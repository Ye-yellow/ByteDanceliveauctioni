package server

import (
	"net/http"
	"os"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
	"live-auction-bid/backend/app/auction/service/internal/storage"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(addr string, auction *appsvc.AuctionService, users *appsvc.UserService, shop *appsvc.ShopService, orders *appsvc.OrderService, hub *realtime.Hub, health HealthChecker, authManager *auth.Manager, authMiddleware middleware.Middleware, imageStorage storage.StorageProvider, assetStore assetStore) *httptransport.Server {
	middlewares := []middleware.Middleware{recovery.Recovery()}
	if authMiddleware != nil {
		middlewares = append(middlewares, authMiddleware)
	}
	srv := httptransport.NewServer(
		httptransport.Address(addr),
		httptransport.Middleware(middlewares...),
		httptransport.Filter(localDevCORSFilter, requestctx.HTTPMiddleware),
	)

	registerAuctionHTTP(srv, auction)
	registerOrderHTTP(srv, orders)
	registerDomainHTTP(srv, auction, users)
	registerShopHTTP(srv, shop)
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

func localDevCORSFilter(next http.Handler) http.Handler {
	allowed := allowedCORSOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && allowed[origin] {
			header := w.Header()
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Vary", "Origin")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-Id,X-Trace-Id,X-Client-App,X-Client-Version,X-Client-Time,Idempotency-Key")
			header.Set("Access-Control-Expose-Headers", "X-Request-Id,X-Trace-Id,X-Server-Time")
			header.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedCORSOrigins() map[string]bool {
	origins := map[string]bool{
		"http://localhost:5173": true,
		"http://localhost:5174": true,
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:5174": true,
	}
	for _, origin := range strings.Split(os.Getenv("AUCTION_HTTP_CORS_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}
