package data

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func TestIntegrationRuntimeProjectionWorkerRecoversAcceptedBid(t *testing.T) {
	if os.Getenv("AUCTION_INTEGRATION_TEST") != "1" {
		t.Skip("set AUCTION_INTEGRATION_TEST=1 to run against real MySQL/Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := NewStore(ctx, Config{
		MySQLDSN:      getenvForTest("AUCTION_MYSQL_DSN", "auction:auction_dev@tcp(127.0.0.1:13306)/live_auction?parseTime=true&charset=utf8mb4&loc=Local"),
		RedisAddr:     getenvForTest("AUCTION_REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword: getenvForTest("AUCTION_REDIS_PASSWORD", "auction_redis"),
	})
	if err != nil {
		t.Fatalf("new integration store failed: %v", err)
	}
	defer store.Close()

	mainAccountID := "it_main_" + idgen.New("acct")
	room, err := store.EnsureDefaultRoom(ctx, mainAccountID, "integration-test", time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("ensure room failed: %v", err)
	}
	defer cleanupProjectionIntegrationRows(ctx, store, mainAccountID, room.ID)

	lot, err := auction.NewLotFromRequest(idgen.New("lot"), &v1.CreateLotRequest{
		RoomId:      room.ID,
		Title:       "runtime projection integration",
		Description: "verifies Redis accepted event can recover MySQL projection",
		ImageUrl:    "https://example.com/runtime-projection.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			CapPrice:               &v1.Money{Amount: 11000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("new lot failed: %v", err)
	}
	lot.MainAccountId = mainAccountID
	if err := store.Create(ctx, lot, "integration-test", nil); err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := auction.StartLot(lot, time.Now().UnixMilli()); err != nil {
		t.Fatalf("start lot domain failed: %v", err)
	}
	if err := store.StartLotAsOnlyActive(ctx, lot, 1, nil); err != nil {
		t.Fatalf("persist start failed: %v", err)
	}

	result, err := store.PlaceBidRuntime(ctx, lot, &v1.PlaceBidRequest{
		LotId:          lot.Id,
		Amount:         &v1.Money{Amount: 11000, Currency: "CNY"},
		IdempotencyKey: "it-runtime-projection",
	}, "it_buyer", "集成测试买家", idgen.New("bid"), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("runtime place bid failed: %v", err)
	}
	if result.RuntimeEventID == "" || result.RuntimeStreamID == "" {
		t.Fatalf("runtime bid must include durable event ids: %+v", result)
	}
	if result.Lot.GetStatus() != v1.LotStatus_LOT_STATUS_SETTLED {
		t.Fatalf("cap bid should settle runtime lot: %+v", result.Lot)
	}
	assertIntegrationCounts(t, ctx, store, lot.Id, 0, 0)

	worker := NewRuntimeProjectionWorker(store, nil, time.Second, 100)
	shard := store.runtimeProjectionShard(lot.Id)
	if _, err := worker.projectShardOnce(ctx, shard); err != nil {
		t.Fatalf("project runtime event failed: %v", err)
	}
	assertIntegrationCounts(t, ctx, store, lot.Id, 1, 1)

	projectedLot, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find projected lot failed: %v", err)
	}
	if projectedLot.GetStatus() != v1.LotStatus_LOT_STATUS_SETTLED || projectedLot.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("projected lot mismatch: %+v", projectedLot)
	}
	var events int64
	if err := store.db.WithContext(ctx).Model(&AuctionEventModel{}).Where("lot_id = ?", lot.Id).Count(&events).Error; err != nil {
		t.Fatalf("count projected events failed: %v", err)
	}
	if events == 0 {
		t.Fatalf("projector should persist auction events")
	}

	if _, err := worker.projectShardOnce(ctx, shard); err != nil {
		t.Fatalf("second projection pass failed: %v", err)
	}
	assertIntegrationCounts(t, ctx, store, lot.Id, 1, 1)
}

func assertIntegrationCounts(t *testing.T, ctx context.Context, store *Store, lotID string, wantBids, wantOrders int64) {
	t.Helper()
	var bids int64
	if err := store.db.WithContext(ctx).Model(&AuctionBidModel{}).Where("lot_id = ?", lotID).Count(&bids).Error; err != nil {
		t.Fatalf("count bids failed: %v", err)
	}
	if bids != wantBids {
		t.Fatalf("bid count mismatch: want=%d got=%d", wantBids, bids)
	}
	var orders int64
	if err := store.db.WithContext(ctx).Model(&AuctionOrderModel{}).Where("lot_id = ?", lotID).Count(&orders).Error; err != nil {
		t.Fatalf("count orders failed: %v", err)
	}
	if orders != wantOrders {
		t.Fatalf("order count mismatch: want=%d got=%d", wantOrders, orders)
	}
}

func cleanupProjectionIntegrationRows(ctx context.Context, store *Store, mainAccountID, roomID string) {
	_ = store.redis.Del(ctx, runtimeEventStreamKey(roomID)).Err()
	_ = store.redis.SRem(ctx, runtimeEventRoomsKey(), roomID).Err()
	for shard := 0; shard < store.runtimeProjectionShards; shard++ {
		_ = store.redis.Del(ctx, runtimeEventShardStreamKey(shard)).Err()
		_ = store.redis.SRem(ctx, runtimeEventShardsKey(), strconv.Itoa(shard)).Err()
	}
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionEventModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionOrderModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionBidModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionLotStatsModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionLotParticipantModel{}).Error
	_ = store.db.WithContext(ctx).Where("room_id = ?", roomID).Delete(&AuctionRuntimeProjectionOffsetModel{}).Error
	_ = store.db.WithContext(ctx).Where("1 = 1").Delete(&AuctionRuntimeProjectionShardOffsetModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionLotModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionRoomStateModel{}).Error
	_ = store.db.WithContext(ctx).Where("main_account_id = ?", mainAccountID).Delete(&AuctionRoomModel{}).Error
}

func getenvForTest(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
