package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

type testSnapshotProvider struct {
	snapshot *v1.RoomSnapshot
}

func (p testSnapshotProvider) Snapshot(context.Context, string) (*v1.RoomSnapshot, error) {
	if p.snapshot == nil {
		return &v1.RoomSnapshot{RoomId: "room1"}, nil
	}
	return proto.Clone(p.snapshot).(*v1.RoomSnapshot), nil
}

type testRoomAccess struct {
	mainByRoom map[string]string
}

func (a testRoomAccess) ValidateRoomInMainAccount(_ context.Context, roomID, mainAccountID string) error {
	if a.mainByRoom[roomID] != mainAccountID {
		return errors.New("room access denied")
	}
	return nil
}

func TestHubRejectsNonWhitelistedOrigin(t *testing.T) {
	hub, _, server := newRealtimeTestServer(t)
	defer server.Close()

	_, response, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=public"), http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		t.Fatal("expected websocket dial to fail for non-whitelisted origin")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got response=%v err=%v hub=%v", response, err, hub)
	}
}

func TestHubOriginPolicyDevLocalhostAndProdMissingOrigin(t *testing.T) {
	devConfig := DefaultConfig()
	devConfig.TicketSecret = "secret"
	devHub := NewHub(nil, devConfig)
	if !devHub.checkOrigin(httptest.NewRequest(http.MethodGet, "/ws/rooms/room1", nil)) {
		t.Fatal("dev should allow missing Origin for non-browser clients")
	}
	devReq := httptest.NewRequest(http.MethodGet, "/ws/rooms/room1", nil)
	devReq.Header.Set("Origin", "http://localhost:5174")
	if !devHub.checkOrigin(devReq) {
		t.Fatal("dev should allow localhost Origin")
	}

	prodHub := NewHub(nil, Config{Environment: "prod", AllowedOrigins: []string{"https://admin.example.test"}, TicketSecret: "secret"})
	if prodHub.checkOrigin(httptest.NewRequest(http.MethodGet, "/ws/rooms/room1", nil)) {
		t.Fatal("prod should reject missing Origin by default")
	}
}

func TestHubRejectsAdminWithoutTicketBeforeUpgrade(t *testing.T) {
	_, _, server := newRealtimeTestServer(t)
	defer server.Close()

	_, response, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=admin"), allowedOriginHeader())
	if err == nil {
		t.Fatal("expected admin websocket dial without ticket to fail")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got response=%v err=%v", response, err)
	}
}

func TestHubRejectsAdminInvalidTicketBeforeUpgrade(t *testing.T) {
	_, _, server := newRealtimeTestServer(t)
	defer server.Close()

	_, response, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=admin&ticket=invalid"), allowedOriginHeader())
	if err == nil {
		t.Fatal("expected admin websocket dial with invalid ticket to fail")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got response=%v err=%v", response, err)
	}
}

func TestWSTicketExpires(t *testing.T) {
	codec := newWSTicketCodec(Config{TicketSecret: "secret", TicketTTL: time.Minute})
	now := time.Unix(1000, 0)
	codec.now = func() time.Time { return now }
	ticket, _, err := codec.issue(wsTicketClaims{RoomID: "room1", Scope: ScopeAdmin, UserID: "main1"})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	codec.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := codec.parse(ticket, "room1", ScopeAdmin); !errors.Is(err, errTicketExpired) {
		t.Fatalf("expected expired ticket, got %v", err)
	}
}

func TestHubPublicAnonymousReceivesRedactedSnapshot(t *testing.T) {
	_, _, server := newRealtimeTestServer(t)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=public"), allowedOriginHeader())
	if err != nil {
		t.Fatalf("dial public websocket: %v", err)
	}
	defer conn.Close()

	event := readAuctionEvent(t, conn)
	if event.GetType() != v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT {
		t.Fatalf("expected snapshot, got %s", event.GetType())
	}
	lot := event.GetSnapshot().GetCurrentLot()
	if lot.GetMainAccountId() != "" || lot.GetLeadingUserId() != "" || lot.GetLeadingNickname() != "张***" {
		t.Fatalf("public snapshot should be redacted, lot=%+v", lot)
	}
	if got := event.GetSnapshot().GetRanking()[0]; got.GetUserId() != "" || got.GetNickname() != "张***" {
		t.Fatalf("public ranking should be redacted, ranking=%+v", got)
	}
}

func TestHubAdminTicketReceivesFullSnapshotAndCrossMainEventRedacted(t *testing.T) {
	hub, manager, server := newRealtimeTestServer(t)
	defer server.Close()

	token := issueAccessToken(t, manager, &v1.User{
		Id:              "main1",
		Username:        "owner",
		Nickname:        "主账号",
		RoleCodes:       []string{userbiz.RoleMerchantOwner},
		PermissionCodes: userbiz.PermissionsForRole(userbiz.RoleMerchantOwner),
		MainAccountId:   "main1",
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	})
	ticket := requestWSTicket(t, server.URL, token, "room1", ScopeAdmin, http.StatusOK)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=admin&ticket="+ticket), allowedOriginHeader())
	if err != nil {
		t.Fatalf("dial admin websocket: %v", err)
	}
	defer conn.Close()

	snapshot := readAuctionEvent(t, conn)
	if snapshot.GetSnapshot().GetCurrentLot().GetMainAccountId() != "main1" || snapshot.GetSnapshot().GetRanking()[0].GetUserId() != "buyer1" {
		t.Fatalf("same-main admin should receive full snapshot: %+v", snapshot.GetSnapshot())
	}

	if err := hub.Publish(context.Background(), v1.AuctionEvent{
		Id:       "evt_snapshot_without_main",
		Type:     v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT,
		RoomId:   "room1",
		Snapshot: testSnapshot(),
	}); err != nil {
		t.Fatalf("publish snapshot event without main account id: %v", err)
	}
	publishedSnapshot := readAuctionEvent(t, conn)
	if publishedSnapshot.GetMainAccountId() != "main1" || publishedSnapshot.GetSnapshot().GetRanking()[0].GetUserId() != "buyer1" {
		t.Fatalf("admin should receive full snapshot even when event mainAccountId is omitted: %+v", publishedSnapshot)
	}

	if err := hub.Publish(context.Background(), v1.AuctionEvent{
		Id:            "evt_order",
		Type:          v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED,
		RoomId:        "room1",
		MainAccountId: "main1",
		Reason:        "order_id=order_private",
		Lot: &v1.Lot{
			Id:             "lot1",
			RoomId:         "room1",
			MainAccountId:  "main1",
			WinnerUserId:   "buyer1",
			WinnerNickname: "张三",
			FinalPrice:     &v1.Money{Amount: 12000, Currency: "CNY"},
		},
	}); err != nil {
		t.Fatalf("publish same-main order event: %v", err)
	}
	orderEvent := readAuctionEvent(t, conn)
	if orderEvent.GetReason() != "order_id=order_private" || orderEvent.GetLot().GetWinnerUserId() != "buyer1" {
		t.Fatalf("same-main admin should receive private order event fields: %+v", orderEvent)
	}

	if err := hub.Publish(context.Background(), v1.AuctionEvent{
		Id:            "evt_cross",
		Type:          v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED,
		RoomId:        "room1",
		MainAccountId: "main2",
		Bid:           &v1.Bid{UserId: "buyer2", Nickname: "李四", Amount: &v1.Money{Amount: 20000, Currency: "CNY"}},
	}); err != nil {
		t.Fatalf("publish cross-main event: %v", err)
	}
	cross := readAuctionEvent(t, conn)
	if cross.GetMainAccountId() != "" || cross.GetBid().GetUserId() != "" || cross.GetBid().GetNickname() != "李***" {
		t.Fatalf("cross-main event should be redacted for admin: %+v", cross)
	}
}

func TestHubRejectsCrossMainAdminTicket(t *testing.T) {
	_, manager, server := newRealtimeTestServer(t)
	defer server.Close()

	token := issueAccessToken(t, manager, &v1.User{
		Id:              "main2",
		Username:        "other",
		Nickname:        "其他主账号",
		RoleCodes:       []string{userbiz.RoleMerchantOwner},
		PermissionCodes: userbiz.PermissionsForRole(userbiz.RoleMerchantOwner),
		MainAccountId:   "main2",
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	})
	_ = requestWSTicket(t, server.URL, token, "room1", ScopeAdmin, http.StatusForbidden)
}

func TestHubPublicAuthCannotEscalateToAdminPrivateData(t *testing.T) {
	_, manager, server := newRealtimeTestServer(t)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL, "/ws/rooms/room1?scope=public"), allowedOriginHeader())
	if err != nil {
		t.Fatalf("dial public websocket: %v", err)
	}
	defer conn.Close()
	_ = readAuctionEvent(t, conn)

	token := issueAccessToken(t, manager, &v1.User{
		Id:              "main1",
		Username:        "owner",
		Nickname:        "主账号",
		RoleCodes:       []string{userbiz.RoleMerchantOwner},
		PermissionCodes: userbiz.PermissionsForRole(userbiz.RoleMerchantOwner),
		MainAccountId:   "main1",
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	})
	if err := conn.WriteJSON(map[string]string{"type": "AUTH", "accessToken": token}); err != nil {
		t.Fatalf("send public AUTH: %v", err)
	}
	event := readAuctionEvent(t, conn)
	if event.GetSnapshot().GetCurrentLot().GetMainAccountId() != "" || event.GetSnapshot().GetRanking()[0].GetUserId() != "" {
		t.Fatalf("public AUTH with admin token must not expose private data: %+v", event.GetSnapshot())
	}
}

func TestRealtimeConfigProdRequiresAllowedOrigins(t *testing.T) {
	if _, err := NormalizeConfig(Config{Environment: "prod", TicketSecret: "secret"}); err == nil {
		t.Fatal("prod websocket config should require allowed origins")
	}
}

func newRealtimeTestServer(t *testing.T) (*Hub, *auth.Manager, *httptest.Server) {
	t.Helper()
	manager, err := auth.NewManager(auth.Config{Secret: "unit-test-secret", Issuer: "test", AccessTTL: time.Minute})
	if err != nil {
		t.Fatalf("new auth manager: %v", err)
	}
	hub := NewHub(testSnapshotProvider{snapshot: testSnapshot()}, Config{
		Environment:        "prod",
		AllowedOrigins:     []string{"https://admin.example.test"},
		AllowMissingOrigin: false,
		TicketTTL:          time.Minute,
		TicketSecret:       "unit-test-secret",
	})
	hub.BindAuthManager(manager)
	hub.BindRoomAccessValidator(testRoomAccess{mainByRoom: map[string]string{"room1": "main1"}})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/realtime/ws-ticket", hub.ServeTicket)
	mux.HandleFunc("/ws/rooms/", func(w http.ResponseWriter, r *http.Request) {
		roomID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/ws/rooms/"), "/")
		hub.ServeRoom(w, r, roomID)
	})
	return hub, manager, httptest.NewServer(mux)
}

func testSnapshot() *v1.RoomSnapshot {
	return &v1.RoomSnapshot{
		RoomId: "room1",
		CurrentLot: &v1.Lot{
			Id:              "lot1",
			RoomId:          "room1",
			MainAccountId:   "main1",
			Status:          v1.LotStatus_LOT_STATUS_LIVE,
			LeadingUserId:   "buyer1",
			LeadingNickname: "张三",
			CurrentPrice:    &v1.Money{Amount: 12000, Currency: "CNY"},
		},
		Ranking: []*v1.RankingItem{{
			Rank:     1,
			UserId:   "buyer1",
			Nickname: "张三",
			Amount:   &v1.Money{Amount: 12000, Currency: "CNY"},
		}},
		RecentBids: []*v1.Bid{{
			UserId:   "buyer1",
			Nickname: "张三",
			Amount:   &v1.Money{Amount: 12000, Currency: "CNY"},
		}},
	}
}

func issueAccessToken(t *testing.T, manager *auth.Manager, user *v1.User) string {
	t.Helper()
	pair, err := manager.IssueTokenPair(user)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return pair.AccessToken
}

func requestWSTicket(t *testing.T, baseURL, token, roomID, scope string, expectedStatus int) string {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"roomId": roomID, "scope": scope})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/realtime/ws-ticket", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new ticket request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request ticket: %v", err)
	}
	defer response.Body.Close()
	var reply wsTicketReply
	_ = json.NewDecoder(response.Body).Decode(&reply)
	if response.StatusCode != expectedStatus {
		t.Fatalf("expected ticket status %d, got %d reply=%+v", expectedStatus, response.StatusCode, reply)
	}
	return reply.Ticket
}

func readAuctionEvent(t *testing.T, conn *websocket.Conn) v1.AuctionEvent {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	var event v1.AuctionEvent
	if err := protojson.Unmarshal(payload, &event); err != nil {
		t.Fatalf("decode websocket event: %v payload=%s", err, string(payload))
	}
	return event
}

func allowedOriginHeader() http.Header {
	return http.Header{"Origin": []string{"https://admin.example.test"}}
}

func wsURL(baseURL, path string) string {
	return "ws" + strings.TrimPrefix(baseURL, "http") + path
}
