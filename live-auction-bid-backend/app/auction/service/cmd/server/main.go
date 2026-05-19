package main

import (
	"context"
	"log"
	"os"
	"time"

	kratos "github.com/go-kratos/kratos/v2"
	"gopkg.in/yaml.v3"

	auctionsvc "live-auction-bid/backend/app/auction/service/internal/service"
	biz "live-auction-bid/backend/app/auction/service/internal/biz"
	"live-auction-bid/backend/app/auction/service/internal/conf"
	data "live-auction-bid/backend/app/auction/service/internal/data"
	serveriface "live-auction-bid/backend/app/auction/service/internal/server"
	ws "live-auction-bid/backend/app/auction/service/internal/server/ws"
)

func main() {
	cfg := conf.Bootstrap{}
	raw, err := os.ReadFile("configs/config.yaml")
	if err != nil { log.Fatal(err) }
	if err := yaml.Unmarshal(raw, &cfg); err != nil { log.Fatal(err) }

	repo := data.NewLotRepository()
	hub := ws.NewHub()
	ai := data.StubAI{}
	ds := biz.NewDomainService(repo, hub, ai, time.Duration(cfg.Auction.AntiSnipeExtendSeconds)*time.Second)
	appSvc := auctionsvc.NewService(repo, ds)
	seedDemo(context.Background(), appSvc, cfg.Auction.DefaultRoomID)

	httpSrv := serveriface.NewServer(cfg.Server.HTTP.Addr, appSvc, hub)
	app := kratos.New(kratos.Name("live-auction-bid"), kratos.Server(httpSrv))
	if err := app.Run(); err != nil { log.Fatal(err) }
}

func seedDemo(ctx context.Context, svc *auctionsvc.Service, roomID string) {
	_, _ = svc.CreateLot(ctx, auctionsvc.CreateLotCommand{RoomID: roomID, Title: "18K 金镶翡翠吊坠", Description: "直播竞拍样品，支持实时出价和延时落锤。", ImageURL: "https://images.unsplash.com/photo-1601121141461-9d6647bca1ed?w=900", StartPrice: 188800, MinIncrement: 5000, DurationSec: 1800})
}
