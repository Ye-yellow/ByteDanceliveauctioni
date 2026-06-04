package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/aiassistant"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	"live-auction-bid/backend/app/auction/service/internal/searchindex"
	"live-auction-bid/backend/app/auction/service/internal/server"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
	"live-auction-bid/backend/app/auction/service/internal/storage"
)

func main() {
	addr := os.Getenv("AUCTION_HTTP_ADDR")
	if addr == "" {
		addr = ":18080"
	}

	store, err := data.NewStore(context.Background(), data.Config{
		MySQLDSN:                getenv("AUCTION_MYSQL_DSN", "auction:auction_dev@tcp(127.0.0.1:13306)/live_auction?parseTime=true&charset=utf8mb4&loc=Local"),
		RedisAddr:               getenv("AUCTION_REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword:           getenv("AUCTION_REDIS_PASSWORD", "auction_redis"),
		RuntimeProjectionShards: getenvInt("AUCTION_RUNTIME_PROJECTION_SHARDS", 16),
		DBMaxOpenConns:          getenvInt("AUCTION_DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:          getenvInt("AUCTION_DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:       getenvDuration("AUCTION_DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime:       getenvDuration("AUCTION_DB_CONN_MAX_IDLE_TIME", 2*time.Minute),
		RedisPoolSize:           getenvInt("AUCTION_REDIS_POOL_SIZE", 0),
		RedisMinIdleConns:       getenvInt("AUCTION_REDIS_MIN_IDLE_CONNS", 0),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buyerSearch, buyerEmbedder, buyerSearchWorker := newBuyerSearchFromEnv(ctx, store)
	if buyerSearch != nil {
		defer buyerSearch.Close()
	}
	if buyerSearchWorker != nil {
		buyerSearchWorker.Start(ctx)
	}
	instanceID := getenv("AUCTION_INSTANCE_ID", defaultInstanceID())
	leaseProvider := data.NewRedisLeaseProvider(store)
	leaseTTL := getenvDuration("AUCTION_WORKER_LEASE_TTL", 15*time.Second)
	leaseRenewInterval := getenvDuration("AUCTION_WORKER_LEASE_RENEW_INTERVAL", 5*time.Second)
	outboxWorker := data.NewEventOutboxWorker(store, 10*time.Second, 100)
	outboxWorker.BindLease(leaseProvider, "auction:lease:event-outbox-worker", instanceID, leaseTTL, leaseRenewInterval)
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
	assetCleanupWorker := data.NewAssetCleanupWorker(store, imageStorage, getenvDuration("AUCTION_TEMP_ASSET_CLEANUP_INTERVAL", time.Hour), 100)
	assetCleanupWorker.Start(ctx)

	userUsecase := userbiz.NewUsecase(store, authManager)
	if err := bootstrapMainAccount(ctx, userUsecase); err != nil {
		log.Fatal(err)
	}

	realtimeConfig, err := realtime.ConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	hub := realtime.NewHub(nil, realtimeConfig)
	hub.BindAuthManager(authManager)
	eventPublisher := realtime.NewPublisher(hub)
	auctionUsecase := auction.NewAuctionUsecase(store, store, store, eventPublisher).
		SetSyncRuntimeProjection(getenvBool("AUCTION_BID_SYNC_PROJECTION", false)).
		SetBidDBGuardMode(getenv("AUCTION_BID_DB_GUARD_MODE", "runtime-first"))
	runtimeProjectionWorker := data.NewRuntimeProjectionWorker(
		store,
		eventPublisher,
		getenvDuration("AUCTION_RUNTIME_PROJECTION_INTERVAL", 2*time.Second),
		getenvInt("AUCTION_RUNTIME_PROJECTION_BATCH_LIMIT", 100),
	)
	runtimeProjectionWorker.BindLease(leaseProvider, instanceID, leaseTTL, leaseRenewInterval)
	runtimeProjectionWorker.Start(ctx)
	hub.BindRoomAccessValidator(auctionUsecase)
	auctionCloseWorker := auction.NewAuctionCloseWorker(auctionUsecase, getenvDuration("AUCTION_CLOSE_WORKER_INTERVAL", 2*time.Second), 100)
	auctionCloseWorker.BindLease(leaseProvider, "auction:lease:auction-close-worker", instanceID, leaseTTL, leaseRenewInterval)
	auctionCloseWorker.Start(ctx)
	hub.BindSnapshotProvider(auctionUsecase)
	auctionService := appsvc.NewAuctionService(auctionUsecase, hub).
		SetAIAssistant(aiassistant.NewFromEnv(os.Getenv)).
		SetBuyerSearch(buyerSearch, buyerEmbedder).
		SetVerboseBidLog(getenvBool("AUCTION_VERBOSE_BID_LOG", false))
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

	readiness := server.Readiness{
		Store:             store,
		Outbox:            outboxWorker,
		AuctionClose:      auctionCloseWorker,
		RuntimeProjection: runtimeProjectionWorker,
		ProjectionMetrics: runtimeProjectionWorker,
		WorkerStatuses:    []auction.WorkerStatusProvider{outboxWorker, auctionCloseWorker, runtimeProjectionWorker},
		Consul:            consulRegistration,
	}
	httpServer := server.NewHTTPServer(addr, auctionService, userService, hub, readiness, authManager, authManager.Middleware(), imageStorage, store)

	log.Printf("auction backend listening on %s via kratos http", addr)
	if err := httpServer.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func newBuyerSearchFromEnv(ctx context.Context, source searchindex.DocumentSource) (*searchindex.PGVectorIndex, *searchindex.EmbeddingClient, *searchindex.SyncWorker) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AUCTION_SEARCH_PROVIDER")))
	dsn := strings.TrimSpace(os.Getenv("AUCTION_SEARCH_PG_DSN"))
	if provider == "" && dsn == "" {
		log.Printf("buyer search index disabled: pgvector dsn is not configured")
		return nil, nil, nil
	}
	if provider != "" && provider != "pgvector" {
		log.Printf("buyer search index disabled: unsupported provider %s", provider)
		return nil, nil, nil
	}
	embedder := searchindex.NewEmbeddingClientFromEnv(os.Getenv)
	if !embedder.Configured() {
		log.Printf("buyer search index disabled: embedding client is not configured")
		return nil, nil, nil
	}
	indexCfg := searchindex.PGVectorConfig{
		DSN:                 dsn,
		EmbeddingModel:      embedder.Model(),
		EmbeddingDimensions: embedder.Dimensions(),
		MaxOpenConns:        getenvInt("AUCTION_SEARCH_PG_MAX_OPEN_CONNS", 5),
		MaxIdleConns:        getenvInt("AUCTION_SEARCH_PG_MAX_IDLE_CONNS", 2),
		ConnMaxLifetime:     getenvDuration("AUCTION_SEARCH_PG_CONN_MAX_LIFETIME", 30*time.Minute),
		ConnMaxIdleTime:     getenvDuration("AUCTION_SEARCH_PG_CONN_MAX_IDLE_TIME", 2*time.Minute),
	}
	index, err := newPGVectorIndexWithStartupWait(ctx, indexCfg)
	if err != nil {
		log.Printf("buyer search index disabled: %v", err)
		return nil, nil, nil
	}
	worker := searchindex.NewSyncWorker(source, index, embedder, searchindex.SyncWorkerConfig{
		Interval:        getenvDuration("AUCTION_SEARCH_SYNC_INTERVAL", 5*time.Minute),
		BatchLimit:      getenvInt("AUCTION_SEARCH_SYNC_BATCH_LIMIT", 500),
		HiddenRetention: getenvDuration("AUCTION_SEARCH_HIDDEN_RETENTION", searchindex.DefaultHiddenRetention),
	})
	log.Printf("buyer search index enabled: model=%s dimensions=%d", embedder.Model(), embedder.Dimensions())
	return index, embedder, worker
}

func newPGVectorIndexWithStartupWait(ctx context.Context, cfg searchindex.PGVectorConfig) (*searchindex.PGVectorIndex, error) {
	wait := getenvDuration("AUCTION_SEARCH_STARTUP_WAIT", 30*time.Second)
	retryEvery := 2 * time.Second
	deadline := time.Now().Add(wait)
	var lastErr error
	for attempt := 1; ; attempt++ {
		index, err := searchindex.NewPGVectorIndex(ctx, cfg)
		if err == nil {
			return index, nil
		}
		lastErr = err
		if wait <= 0 || time.Now().After(deadline) {
			return nil, lastErr
		}
		log.Printf("buyer search index waiting for pgvector: attempt=%d error=%v", attempt, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryEvery):
		}
	}
}

func defaultInstanceID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
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

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("invalid %s integer: %v", key, err)
	}
	return parsed
}

func bootstrapMainAccount(ctx context.Context, users *userbiz.Usecase) error {
	username := os.Getenv("AUCTION_BOOTSTRAP_MAIN_ACCOUNT_USERNAME")
	password := os.Getenv("AUCTION_BOOTSTRAP_MAIN_ACCOUNT_PASSWORD")
	nickname := os.Getenv("AUCTION_BOOTSTRAP_MAIN_ACCOUNT_NICKNAME")
	if username == "" && password == "" && nickname == "" {
		return nil
	}
	if username == "" || password == "" || nickname == "" {
		return errors.New("bootstrap main account username, password and nickname must be configured together")
	}
	return users.BootstrapMainAccount(ctx, username, password, nickname)
}

func newImageStorageFromEnv() (storage.StorageProvider, error) {
	provider := strings.TrimSpace(os.Getenv("AUCTION_STORAGE_PROVIDER"))
	if provider == "" {
		provider = "tos"
	}
	if provider != "tos" {
		return nil, errors.New("unsupported auction storage provider: " + provider)
	}
	return storage.NewTOSStorage(storage.TOSConfig{
		Endpoint:      strings.TrimSpace(os.Getenv("AUCTION_TOS_ENDPOINT")),
		Region:        strings.TrimSpace(os.Getenv("AUCTION_TOS_REGION")),
		Bucket:        strings.TrimSpace(os.Getenv("AUCTION_TOS_BUCKET")),
		AccessKey:     strings.TrimSpace(os.Getenv("AUCTION_TOS_ACCESS_KEY")),
		SecretKey:     strings.TrimSpace(os.Getenv("AUCTION_TOS_SECRET_KEY")),
		PublicBaseURL: strings.TrimSpace(os.Getenv("AUCTION_TOS_PUBLIC_BASE_URL")),
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
