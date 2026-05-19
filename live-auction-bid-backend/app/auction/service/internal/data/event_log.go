package data

import (
	"context"
	"errors"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

var ErrEventLogNotConfigured = errors.New("auction event log is not configured")

type EventLog struct{}

func NewEventLog() *EventLog { return &EventLog{} }

func (l *EventLog) Append(ctx context.Context, event biz.AuctionEvent) error {
	return ErrEventLogNotConfigured
}
