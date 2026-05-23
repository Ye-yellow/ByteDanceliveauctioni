package test

import (
	"context"
	"testing"

	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	service "live-auction-bid/backend/app/auction/service/internal/service"
)

func TestServiceErrorIsWrappedIntoReplyResult(t *testing.T) {
	result := service.ErrorResult(context.Background(), apperr.ErrLotVersionConflict)
	if result.GetCode() != service.ResultCodeLotVersionConflict || result.GetMessage() != service.MessageLotVersionConflict {
		t.Fatalf("expected wrapped lot version conflict result, got code=%d message=%q", result.GetCode(), result.GetMessage())
	}
}
