package test

import (
	"context"
	"testing"

	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	service "live-auction-bid/backend/app/auction/service/internal/service"
)

func TestServiceErrorIsWrappedIntoReplyResult(t *testing.T) {
	result := service.ErrorResult(context.Background(), apperr.ErrLotVersionConflict)
	if result.GetCode() != service.ResultCodeBidVersionStale || result.GetMessage() != string(apperr.CodeBidVersionStale) {
		t.Fatalf("expected wrapped lot version conflict result, got code=%d message=%q", result.GetCode(), result.GetMessage())
	}

	active := service.ErrorResult(context.Background(), apperr.ErrRoomActiveLotExists)
	if active.GetCode() != service.ResultCodeRoomActiveLotExists || active.GetMessage() != string(apperr.CodeRoomActiveLotExists) {
		t.Fatalf("expected wrapped room active lot result, got code=%d message=%q", active.GetCode(), active.GetMessage())
	}

	queue := service.ErrorResult(context.Background(), apperr.ErrQueuePositionConflict)
	if queue.GetCode() != service.ResultCodeQueuePositionConflict || queue.GetMessage() != service.MessageQueuePositionConflict {
		t.Fatalf("expected wrapped queue position conflict result, got code=%d message=%q", queue.GetCode(), queue.GetMessage())
	}
}

func TestServiceBidErrorsReturnStableBusinessCodes(t *testing.T) {
	cases := []struct {
		err     error
		code    int32
		message string
	}{
		{apperr.ErrBidTooLow, service.ResultCodeBidTooLow, string(apperr.CodeBidTooLow)},
		{apperr.ErrBidNotLive, service.ResultCodeBidNotLive, string(apperr.CodeBidNotLive)},
		{apperr.ErrBidEnded, service.ResultCodeBidEnded, string(apperr.CodeBidEnded)},
		{apperr.ErrBidAlreadyLeading, service.ResultCodeBidAlreadyLeading, string(apperr.CodeBidAlreadyLeading)},
		{apperr.ErrBidCurrencyMismatch, service.ResultCodeBidCurrencyMismatch, string(apperr.CodeBidCurrencyMismatch)},
		{apperr.ErrLotCancelled, service.ResultCodeLotCancelled, string(apperr.CodeLotCancelled)},
		{apperr.ErrRuntimeProjectionGap, service.ResultCodeProjectionPending, string(apperr.CodeProjectionPending)},
	}
	for _, tc := range cases {
		result := service.ErrorResult(context.Background(), tc.err)
		if result.GetCode() != tc.code || result.GetMessage() != tc.message {
			t.Fatalf("expected code=%d message=%q for %v, got code=%d message=%q", tc.code, tc.message, tc.err, result.GetCode(), result.GetMessage())
		}
	}
}
