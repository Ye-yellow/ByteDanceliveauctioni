package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	"live-auction-bid/backend/app/auction/service/internal/server"
	"live-auction-bid/backend/app/auction/service/internal/storage"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

func main() {
	addr := os.Getenv("AUCTION_HTTP_ADDR")
	if addr == "" {
		addr = ":18080"
	}

	store, err := data.NewStore(context.Background(), data.Config{
		MySQLDSN:      getenv("AUCTION_MYSQL_DSN", "auction:auction_dev@tcp(127.0.0.1:13306)/live_auction?parseTime=true&charset=utf8mb4&loc=Local"),
		RedisAddr:     getenv("AUCTION_REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword: getenv("AUCTION_REDIS_PASSWORD", "auction_redis"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outboxWorker := data.NewEventOutboxWorker(store, 10*time.Second, 100)
	outboxWorker.Start(ctx)

	authManager, err := auth.NewManager(auth.Config{
		Secret:     os.Getenv("AUCTION_JWT_SECRET"),
		Issuer:     getenv("AUCTION_JWT_ISSUER", "auction-backend"),
		AccessTTL:  getenvDuration("AUCTION_ACCESS_TOKEN_TTL", auth.DefaultAccessTTL),
		RefreshTTL: getenvDuration("AUCTION_REFRESH_TOKEN_TTL", auth.DefaultRefreshTTL),
	})
	if err != nil {
		log.Fatal(err)
	}
	imageStorage, err := newImageStorageFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	userUsecase := userbiz.NewUsecase(store, authManager)
	if err := bootstrapAdmin(ctx, userUsecase); err != nil {
		log.Fatal(err)
	}

	hub := realtime.NewHub(nil)
	eventPublisher := realtime.NewPublisher(hub)
	auctionUsecase := auction.NewAuctionUsecase(store, store, store, eventPublisher)
	hub.BindSnapshotProvider(auctionUsecase)
	auctionService := appsvc.NewAuctionService(auctionUsecase)
	userService := appsvc.NewUserService(userUsecase)
	consulRegistration, err := server.RegisterConsulService(context.Background(), server.ConsulConfig{
		Addr:           getenv("AUCTION_CONSUL_ADDR", "127.0.0.1:18500"),
		ServiceName:    getenv("AUCTION_SERVICE_NAME", "auction-backend"),
		ServiceAddress: getenv("AUCTION_SERVICE_ADDR", "127.0.0.1"),
		HTTPAddr:       addr,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := consulRegistration.Deregister(ctx); err != nil {
			log.Printf("consul deregister failed: %v", err)
		}
	}()

	readiness := server.Readiness{Store: store, Outbox: outboxWorker, Consul: consulRegistration}
	httpServer := server.NewHTTPServer(addr, auctionService, userService, hub, readiness, authManager, authManager.Middleware(), imageStorage, store)

	log.Printf("auction backend listening on %s via kratos http", addr)
	if err := httpServer.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("invalid %s duration: %v", key, err)
	}
	return duration
}

func bootstrapAdmin(ctx context.Context, users *userbiz.Usecase) error {
	username := os.Getenv("AUCTION_BOOTSTRAP_ADMIN_USERNAME")
	password := os.Getenv("AUCTION_BOOTSTRAP_ADMIN_PASSWORD")
	nickname := os.Getenv("AUCTION_BOOTSTRAP_ADMIN_NICKNAME")
	if username == "" && password == "" && nickname == "" {
		return nil
	}
	if username == "" || password == "" || nickname == "" {
		return errors.New("bootstrap admin username, password and nickname must be configured together")
	}
	return users.BootstrapAdmin(ctx, username, password, nickname)
}


func newImageStorageFromEnv() (storage.StorageProvider, error) {
	provider := getenv("AUCTION_STORAGE_PROVIDER", "")
	if provider == "" {
		return nil, nil
	}
	if provider != "tos" {
		return nil, errors.New("unsupported auction storage provider: " + provider)
	}
	return storage.NewTOSStorage(storage.TOSConfig{
		Endpoint:      getenv("AUCTION_TOS_ENDPOINT", "tos-cn-beijing.volces.com"),
		Region:        getenv("AUCTION_TOS_REGION", "cn-beijing"),
		Bucket:        getenv("AUCTION_TOS_BUCKET", "live-auction-assets"),
		AccessKey:     os.Getenv("AUCTION_TOS_ACCESS_KEY"),
		SecretKey:     os.Getenv("AUCTION_TOS_SECRET_KEY"),
		PublicBaseURL: os.Getenv("AUCTION_TOS_PUBLIC_BASE_URL"),
		UseSSL:        getenvBool("AUCTION_TOS_USE_SSL", true),
	})
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		log.Fatalf("invalid %s bool: %s", key, value)
		return fallback
	}
}
