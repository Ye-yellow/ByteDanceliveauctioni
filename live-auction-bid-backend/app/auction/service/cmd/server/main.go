package main

import (
	"context"
	"log"
	"os"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	"live-auction-bid/backend/app/auction/service/internal/server"
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

	hub := realtime.NewHub(nil)
	eventPublisher := realtime.NewPublisher(hub)
	auctionUsecase := auction.NewAuctionUsecase(store, store, store, eventPublisher)
	hub.BindSnapshotProvider(auctionUsecase)
	auctionService := appsvc.NewAuctionService(auctionUsecase)
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
	httpServer := server.NewHTTPServer(addr, auctionService, hub, readiness)

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
