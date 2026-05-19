package main

import (
	"log"
	"net/http"
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
	auction := auction.NewAuctionUsecase(store, store, hub)
	hub.BindSnapshotProvider(auction)
	app := appsvc.NewAuctionService(auction)
	httpServer := server.New(app, hub)

	log.Printf("auction backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, httpServer.Handler()))
}
