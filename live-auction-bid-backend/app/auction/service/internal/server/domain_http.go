package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type lotResultHTTPResponse struct {
	Result       *v1.ReplyResult       `json:"result"`
	Lot          json.RawMessage       `json:"lot,omitempty"`
	AuctionState auction.AuctionState  `json:"auctionState,omitempty"`
	Order        *auction.OrderSummary `json:"order,omitempty"`
}

type listOrdersHTTPResponse struct {
	Result *v1.ReplyResult        `json:"result"`
	Orders []auction.OrderSummary `json:"orders"`
	Total  int64                  `json:"total"`
	Page   int                    `json:"page"`
	Size   int                    `json:"pageSize"`
}

type listBidRecordsHTTPResponse struct {
	Result *v1.ReplyResult     `json:"result"`
	Bids   []auction.BidRecord `json:"bids"`
	Total  int64               `json:"total"`
	Page   int                 `json:"page"`
	Size   int                 `json:"pageSize"`
}

type listLotsHTTPResponse struct {
	Result *v1.ReplyResult   `json:"result"`
	Lots   []json.RawMessage `json:"lots"`
	Total  int64             `json:"total"`
	Page   int               `json:"page"`
	Size   int               `json:"pageSize"`
}

type listRoomsHTTPResponse struct {
	Result *v1.ReplyResult `json:"result"`
	Rooms  []auction.Room  `json:"rooms"`
}

type publicRoomHTTPResponse struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Platform        string             `json:"platform,omitempty"`
	PlatformRoomID  string             `json:"platformRoomId,omitempty"`
	Status          auction.RoomStatus `json:"status,omitempty"`
	CreatedAtUnixMs int64              `json:"createdAtUnixMs,omitempty"`
	UpdatedAtUnixMs int64              `json:"updatedAtUnixMs,omitempty"`
}

type listPublicRoomsHTTPResponse struct {
	Result *v1.ReplyResult          `json:"result"`
	Rooms  []publicRoomHTTPResponse `json:"rooms"`
}

type listUsersHTTPResponse struct {
	Result *v1.ReplyResult   `json:"result"`
	Users  []json.RawMessage `json:"users"`
	Total  int64             `json:"total"`
	Page   int               `json:"page"`
	Size   int               `json:"pageSize"`
}

type mockPayHTTPResponse struct {
	Result  *v1.ReplyResult         `json:"result"`
	Order   *auction.OrderSummary   `json:"order,omitempty"`
	Payment *auction.PaymentSummary `json:"payment,omitempty"`
	Paid    bool                    `json:"paid"`
}

var domainProtoJSONMarshal = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseEnumNumbers:  false,
	UseProtoNames:   false,
}

func protoJSONRaw(message proto.Message) json.RawMessage {
	if message == nil {
		return nil
	}
	payload, err := domainProtoJSONMarshal.Marshal(message)
	if err != nil {
		return nil
	}
	return json.RawMessage(payload)
}

func protoJSONRawSlice[T proto.Message](messages []T) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, protoJSONRaw(message))
	}
	return out
}

func registerDomainHTTP(srv *httptransport.Server, service *appsvc.AuctionService, users *appsvc.UserService) {
	r := srv.Route("/")
	r.GET("/api/lots/{lot_id}/result", func(ctx httptransport.Context) error {
		lotID := ctx.Vars().Get("lot_id")
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			result, err := service.GetLotResult(ctx, lotID)
			if err != nil {
				return lotResultHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return lotResultHTTPResponse{Result: appsvc.OKResult(ctx), Lot: protoJSONRaw(result.Lot), AuctionState: result.AuctionState, Order: result.Order}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, lotResultHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/me/orders", func(ctx httptransport.Context) error {
		query := orderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			orders, err := service.ListMyOrdersPage(ctx, query)
			if err != nil {
				return listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}}, nil
			}
			return listOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: orders.Orders, Total: orders.Total, Page: orders.Page, Size: orders.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/me/bids", func(ctx httptransport.Context) error {
		query := bidRecordQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			bids, err := service.ListMyBids(ctx, query)
			if err != nil {
				return listBidRecordsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Bids: []auction.BidRecord{}}, nil
			}
			return listBidRecordsHTTPResponse{Result: appsvc.OKResult(ctx), Bids: bids.Bids, Total: bids.Total, Page: bids.Page, Size: bids.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listBidRecordsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Bids: []auction.BidRecord{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/orders", func(ctx httptransport.Context) error {
		query := orderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			orders, err := service.ListOrders(ctx, query)
			if err != nil {
				return listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}}, nil
			}
			return listOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: orders.Orders, Total: orders.Total, Page: orders.Page, Size: orders.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/lots", func(ctx httptransport.Context) error {
		query := lotQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			lots, err := service.ListAdminLots(ctx, query)
			if err != nil {
				return listLotsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Lots: []json.RawMessage{}}, nil
			}
			return listLotsHTTPResponse{Result: appsvc.OKResult(ctx), Lots: protoJSONRawSlice(lots.Lots), Total: lots.Total, Page: lots.Page, Size: lots.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listLotsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Lots: []json.RawMessage{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/rooms", func(ctx httptransport.Context) error {
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			rooms, err := service.ListAdminRooms(ctx)
			if err != nil {
				return listRoomsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Rooms: []auction.Room{}}, nil
			}
			return listRoomsHTTPResponse{Result: appsvc.OKResult(ctx), Rooms: rooms}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listRoomsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Rooms: []auction.Room{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/rooms", func(ctx httptransport.Context) error {
		rooms, err := service.ListPublicRooms(ctx)
		if err != nil {
			return ctx.Result(200, listPublicRoomsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Rooms: []publicRoomHTTPResponse{}})
		}
		return ctx.Result(200, listPublicRoomsHTTPResponse{Result: appsvc.OKResult(ctx), Rooms: publicRoomsHTTP(rooms)})
	})
	r.GET("/api/admin/users", func(ctx httptransport.Context) error {
		query := userQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			list, err := users.ListUsers(ctx, query)
			if err != nil {
				return listUsersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Users: []json.RawMessage{}}, nil
			}
			return listUsersHTTPResponse{Result: appsvc.OKResult(ctx), Users: protoJSONRawSlice(list.Users), Total: list.Total, Page: list.Page, Size: list.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listUsersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Users: []json.RawMessage{}})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/orders/{order_id}/mock-pay", func(ctx httptransport.Context) error {
		orderID := ctx.Vars().Get("order_id")
		var req auction.MockPayRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			result, err := service.MockPayOrder(ctx, orderID, req)
			if err != nil {
				return mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return mockPayHTTPResponse{Result: appsvc.OKResult(ctx), Order: &result.Order, Payment: &result.Payment, Paid: result.Paid}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
}

func publicRoomsHTTP(rooms []auction.Room) []publicRoomHTTPResponse {
	out := make([]publicRoomHTTPResponse, 0, len(rooms))
	for _, room := range rooms {
		out = append(out, publicRoomHTTPResponse{
			ID:              room.ID,
			Name:            room.Name,
			Platform:        room.Platform,
			PlatformRoomID:  room.PlatformRoomID,
			Status:          room.Status,
			CreatedAtUnixMs: room.CreatedAtUnixMs,
			UpdatedAtUnixMs: room.UpdatedAtUnixMs,
		})
	}
	return out
}

func orderQueryFromHTTP(ctx httptransport.Context) auction.OrderQuery {
	query := ctx.Query()
	return auction.OrderQuery{
		Page:          intQuery(query.Get("page")),
		PageSize:      intQuery(query.Get("pageSize")),
		Status:        auction.OrderStatus(strings.TrimSpace(query.Get("status"))),
		PaymentStatus: auction.PaymentStatus(strings.TrimSpace(query.Get("paymentStatus"))),
		LotID:         strings.TrimSpace(query.Get("lotId")),
		Buyer:         strings.TrimSpace(query.Get("buyer")),
	}
}

func bidRecordQueryFromHTTP(ctx httptransport.Context) auction.BidRecordQuery {
	query := ctx.Query()
	return auction.BidRecordQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		LotID:    strings.TrimSpace(query.Get("lotId")),
	}
}

func lotQueryFromHTTP(ctx httptransport.Context) auction.LotQuery {
	query := ctx.Query()
	return auction.LotQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		Status:   lotStatusFromString(query.Get("status")),
		View:     strings.TrimSpace(query.Get("view")),
		Keyword:  strings.TrimSpace(query.Get("keyword")),
		RoomID:   strings.TrimSpace(query.Get("roomId")),
	}
}

func userQueryFromHTTP(ctx httptransport.Context) userbiz.ListUsersQuery {
	query := ctx.Query()
	return userbiz.ListUsersQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		RoleCode: strings.TrimSpace(query.Get("roleCode")),
		Status:   userStatusFromString(query.Get("status")),
		Keyword:  strings.TrimSpace(query.Get("keyword")),
	}
}

func intQuery(value string) int {
	next, _ := strconv.Atoi(strings.TrimSpace(value))
	return next
}

func lotStatusFromString(value string) v1.LotStatus {
	value = strings.TrimSpace(value)
	if value == "" {
		return v1.LotStatus_LOT_STATUS_UNSPECIFIED
	}
	next, ok := v1.LotStatus_value[value]
	if !ok {
		return v1.LotStatus_LOT_STATUS_UNSPECIFIED
	}
	return v1.LotStatus(next)
}

func userStatusFromString(value string) v1.UserStatus {
	value = strings.TrimSpace(value)
	if value == "" {
		return v1.UserStatus_USER_STATUS_UNSPECIFIED
	}
	next, ok := v1.UserStatus_value[value]
	if !ok {
		return v1.UserStatus_USER_STATUS_UNSPECIFIED
	}
	return v1.UserStatus(next)
}
