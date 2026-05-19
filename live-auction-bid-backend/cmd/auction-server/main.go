package main

import (
	"context"
	"log"
	"os"
	"time"

	kratos "github.com/go-kratos/kratos/v2"
	"gopkg.in/yaml.v3"

	auctionapp "live-auction-bid/backend/internal/application/auction"
	domain "live-auction-bid/backend/internal/domain/auction"
	"live-auction-bid/backend/internal/conf"
	"live-auction-bid/backend/internal/infrastructure/memory"
	httpiface "live-auction-bid/backend/internal/interfaces/http"
	"live-auction-bid/backend/internal/interfaces/ws"
)

func main() {
	cfg := conf.Bootstrap{}
	data, err := os.ReadFile("configs/config.yaml")
	if err != nil { log.Fatal(err) }
	if err := yaml.Unmarshal(data, &cfg); err != nil { log.Fatal(err) }

	repo := memory.NewLotRepository()
	hub := ws.NewHub()
	ai := memory.StubAI{}
	ds := domain.NewDomainService(repo, hub, ai, time.Duration(cfg.Auction.AntiSnipeExtendSeconds)*time.Second)
	appSvc := auctionapp.NewService(repo, ds)
	seedDemo(context.Background(), appSvc, cfg.Auction.DefaultRoomID)

	httpSrv := httpiface.NewServer(cfg.Server.HTTP.Addr, appSvc, hub)
	app := kratos.New(kratos.Name("live-auction-bid"), kratos.Server(httpSrv))
	if err := app.Run(); err != nil { log.Fatal(err) }
}

func seedDemo(ctx context.Context, svc *auctionapp.Service, roomID string) {
	_, _ = svc.CreateLot(ctx, auctionapp.CreateLotCommand{RoomID: roomID, Title: "18K 金镶翡翠吊坠", Description: "直播竞拍样品，支持实时出价和延时落锤。", ImageURL: "https://images.unsplash.com/photo-1601121141461-9d6647bca1ed?w=900", StartPrice: 188800, MinIncrement: 5000, DurationSec: 1800})
}
