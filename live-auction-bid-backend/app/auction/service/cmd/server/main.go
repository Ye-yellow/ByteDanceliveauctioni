package main

import (
	"log"
	"net/http"
	"os"

	"live-auction-bid/backend/app/auction/service/internal/biz"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	"live-auction-bid/backend/app/auction/service/internal/server"
)

func main() {
	addr := os.Getenv("AUCTION_HTTP_ADDR")
	if addr == "" { addr = ":18080" }
	repo := data.NewMemoryLotRepo()
	var app *biz.Service
	hub := realtime.NewHub(nil)
	app = biz.NewService(repo, hub)
	hub = realtime.NewHub(app)
	app = biz.NewService(repo, hub)
	srv := server.New(app, hub)
	log.Printf("auction backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
