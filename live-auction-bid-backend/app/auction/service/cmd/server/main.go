package main

import (
	"context"
	"log"
	"os"

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

	store := data.NewMemoryStore()
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
