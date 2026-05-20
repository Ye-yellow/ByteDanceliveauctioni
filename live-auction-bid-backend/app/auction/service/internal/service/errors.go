package service

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const (
	ResultCodeOK                 int32 = 0
	ResultCodeLotVersionConflict int32 = 409001
	ResultCodeInternalError      int32 = 500000
)

const MessageLotVersionConflict = "lot state changed, please refresh and retry"

func okResult() *v1.ReplyResult {
	return &v1.ReplyResult{Code: ResultCodeOK, Message: "ok"}
}

func ErrorResult(err error) *v1.ReplyResult {
	if err == nil {
		return okResult()
	}
	if apperr.IsLotVersionConflict(err) {
		return &v1.ReplyResult{Code: ResultCodeLotVersionConflict, Message: MessageLotVersionConflict}
	}
	return &v1.ReplyResult{Code: ResultCodeInternalError, Message: err.Error()}
}
