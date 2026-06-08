package auction

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

const runtimeProjectionBidAccepted = "BID_ACCEPTED"

func BuildRuntimeBidProjectionArtifacts(projection RuntimeProjectionEvent) ([]v1.AuctionEvent, *Order, error) {
	if projection.Lot == nil {
		return nil, nil, errors.New("projection lot is required")
	}
	if projection.Bid.Id == "" {
		return nil, nil, errors.New("projection bid is required")
	}
	nowMs := projection.OccurredAtUnixMs
	eventID := func(suffix string) string {
		return deterministicProjectionID("evt", projection.RuntimeEventID, suffix)
	}

	acceptedEvent := newAuctionEventWithID(eventID("accepted"), v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED, projection.Lot, nowMs)
	acceptedEvent.Bid = cloneBid(projection.Bid)
	acceptedEvent.Ranking = projection.Ranking
	rankingEvent := newAuctionEventWithID(eventID("ranking"), v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED, projection.Lot, nowMs)
	rankingEvent.Ranking = projection.Ranking
	events := []v1.AuctionEvent{acceptedEvent, rankingEvent}

	if projection.PreviousLeaderID != "" && projection.PreviousLeaderID != projection.Bid.UserId {
		outbidEvent := newAuctionEventWithID(eventID("outbid"), v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_OUTBID, projection.Lot, nowMs)
		outbidEvent.Bid = cloneBid(projection.Bid)
		outbidEvent.Ranking = projection.Ranking
		outbidEvent.Reason = projection.PreviousLeaderID
		events = append(events, outbidEvent)
	}
	if projection.Lot.EndsAtUnixMs != projection.EndsBeforeBid || projection.Lot.GetDuelState().GetExtendCount() != projection.ExtendCountBefore {
		updatedEvent := newAuctionEventWithID(eventID("lot_updated"), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED, projection.Lot, nowMs)
		updatedEvent.Bid = cloneBid(projection.Bid)
		updatedEvent.Ranking = projection.Ranking
		updatedEvent.DuelState = projection.Lot.DuelState
		events = append(events, updatedEvent)
		extendedEvent := newAuctionEventWithID(eventID("extended"), v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_EXTENDED, projection.Lot, nowMs)
		extendedEvent.Bid = cloneBid(projection.Bid)
		extendedEvent.Ranking = projection.Ranking
		extendedEvent.DuelState = projection.Lot.DuelState
		events = append(events, extendedEvent)
	}
	if projection.Lot.GetDuelState().GetActive() {
		duelEvent := newAuctionEventWithID(eventID("duel_started"), v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, projection.Lot, nowMs)
		duelEvent.Ranking = projection.Ranking
		duelEvent.DuelState = projection.Lot.DuelState
		events = append(events, duelEvent)
	}
	if AuctionStateOf(projection.Lot) != AuctionStateSettled {
		return events, nil, nil
	}

	orderID := strings.TrimSpace(projection.OrderID)
	if orderID == "" {
		orderID = deterministicProjectionID("order", projection.RuntimeEventID, "settled")
	}
	order, err := NewOrderFromSettledLot(orderID, projection.Lot, nowMs)
	if err != nil {
		return nil, nil, err
	}
	settledEvent := newAuctionEventWithID(eventID("settled"), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, projection.Lot, nowMs)
	settledEvent.Bid = cloneBid(projection.Bid)
	settledEvent.Ranking = projection.Ranking
	closedEvent := newAuctionEventWithID(eventID("closed"), v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, projection.Lot, nowMs)
	closedEvent.Bid = cloneBid(projection.Bid)
	closedEvent.Ranking = projection.Ranking
	orderEvent := newAuctionEventWithID(eventID("order_created"), v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED, projection.Lot, nowMs)
	orderEvent.Ranking = projection.Ranking
	orderEvent.Reason = orderCreatedPublicReason
	events = append(events, settledEvent, closedEvent, orderEvent)
	return events, order, nil
}

func RuntimeProjectionFromBidResult(result RuntimeBidResult, idempotencyKey string, occurredAtUnixMs int64) RuntimeProjectionEvent {
	event := RuntimeProjectionEvent{
		RuntimeEventID:     result.RuntimeEventID,
		RuntimeStreamID:    result.RuntimeStreamID,
		EventType:          runtimeProjectionBidAccepted,
		IdempotencyKey:     idempotencyKey,
		Lot:                result.Lot,
		Ranking:            result.Ranking,
		PreviousLeaderID:   result.PreviousLeaderID,
		EndsBeforeBid:      result.EndsBeforeBid,
		ExtendCountBefore:  result.ExtendCountBefore,
		PreviousLotVersion: result.PreviousLotVersion,
		LotVersion:         result.LotVersion,
		OccurredAtUnixMs:   occurredAtUnixMs,
		OrderID:            result.OrderID,
	}
	if result.Bid != nil {
		event.Bid = *result.Bid
	}
	if result.Lot != nil {
		event.RoomID = result.Lot.RoomId
		event.LotID = result.Lot.Id
		if event.LotVersion == 0 {
			event.LotVersion = result.Lot.Version
		}
		if event.PreviousLotVersion == 0 && result.Lot.Version > 0 {
			event.PreviousLotVersion = result.Lot.Version - 1
		}
	}
	return event
}

func deterministicProjectionID(prefix, runtimeEventID, suffix string) string {
	runtimeEventID = strings.TrimSpace(runtimeEventID)
	if runtimeEventID == "" {
		return idgen.New(prefix)
	}
	sum := sha256.Sum256([]byte(runtimeEventID + ":" + suffix))
	return prefix + "_" + hex.EncodeToString(sum[:8])
}

func cloneBid(bid v1.Bid) *v1.Bid {
	next := bid
	return &next
}
