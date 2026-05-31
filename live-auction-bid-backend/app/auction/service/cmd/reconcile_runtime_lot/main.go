package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"live-auction-bid/backend/app/auction/service/internal/data"
)

func main() {
	lotID := flag.String("lot-id", "", "auction lot id to reconcile into Redis runtime")
	flag.Parse()
	if *lotID == "" {
		log.Fatal("--lot-id is required")
	}

	ctx := context.Background()
	store, err := data.NewStore(ctx, data.Config{
		MySQLDSN:      getenv("AUCTION_MYSQL_DSN", "auction:auction_dev@tcp(127.0.0.1:13306)/live_auction?parseTime=true&charset=utf8mb4&loc=Local"),
		RedisAddr:     getenv("AUCTION_REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword: getenv("AUCTION_REDIS_PASSWORD", "auction_redis"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	lot, err := store.FindByID(ctx, *lotID)
	if err != nil {
		log.Fatal(err)
	}
	if err := store.SyncLotRuntime(ctx, lot); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("reconciled lot %s into Redis runtime at version %d\n", lot.Id, lot.Version)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
