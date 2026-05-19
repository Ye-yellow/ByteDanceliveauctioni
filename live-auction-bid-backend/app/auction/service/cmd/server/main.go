package main

import (
	"context"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
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

	hub := realtime.NewHub(nil)
	auctionUsecase := auction.NewAuctionUsecase(store, store, hub)
	hub.BindSnapshotProvider(auctionUsecase)
	auctionService := appsvc.NewAuctionService(auctionUsecase)
	httpServer := server.NewHTTPServer(addr, auctionService, hub)

	log.Printf("auction backend listening on %s via kratos http", addr)
	if err := httpServer.Start(context.Background()); err != nil {
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
