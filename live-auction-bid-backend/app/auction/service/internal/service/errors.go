package service

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"
)

const (
	ResultCodeOK                 int32 = 0
	ResultCodeInvalidArgument    int32 = 400001
	ResultCodeLoginRequired      int32 = 401001
	ResultCodeTokenExpired       int32 = 401002
	ResultCodeTokenInvalid       int32 = 401003
	ResultCodeSessionExpired     int32 = 401004
	ResultCodeInvalidCredentials int32 = 401005
	ResultCodeForbidden          int32 = 403001
	ResultCodeAccountDisabled    int32 = 403002
	ResultCodeLotVersionConflict int32 = 409001
	ResultCodeUsernameTaken      int32 = 409002
	ResultCodeUserNotFound       int32 = 404001
	ResultCodeInternalError      int32 = 500000

	ResultCodeUnauthenticated  = ResultCodeLoginRequired
	ResultCodeInvalidToken     = ResultCodeTokenInvalid
	ResultCodePermissionDenied = ResultCodeForbidden
)

const (
	MessageOK                 = "ok"
	MessageLotVersionConflict = "lot state changed, please refresh and retry"
	MessageInternalError      = "internal error, please try again later"
	MessageTokenExpired       = "access token expired, please refresh"
	MessageSessionExpired     = "session expired, please login again"
)

func OKResult(ctx context.Context) *v1.ReplyResult {
	return okResult(ctx)
}

func okResult(ctx context.Context) *v1.ReplyResult {
	return &v1.ReplyResult{Code: ResultCodeOK, Message: MessageOK, TraceId: requestctx.TraceID(ctx)}
}

func ErrorResult(ctx context.Context, err error) *v1.ReplyResult {
	if err == nil {
		return okResult(ctx)
	}
	traceID := requestctx.TraceID(ctx)
	if apperr.IsLotVersionConflict(err) {
		return &v1.ReplyResult{Code: ResultCodeLotVersionConflict, Message: MessageLotVersionConflict, TraceId: traceID}
	}
	if apperr.IsInvalidArgument(err) {
		return &v1.ReplyResult{Code: ResultCodeInvalidArgument, Message: err.Error(), TraceId: traceID}
	}
	if apperr.IsUnauthenticated(err) {
		return &v1.ReplyResult{Code: ResultCodeLoginRequired, Message: "login required", TraceId: traceID}
	}
	if apperr.IsPermissionDenied(err) {
		return &v1.ReplyResult{Code: ResultCodeForbidden, Message: "permission denied", TraceId: traceID}
	}
	if apperr.IsAccountDisabled(err) {
		return &v1.ReplyResult{Code: ResultCodeAccountDisabled, Message: "account disabled", TraceId: traceID}
	}
	if apperr.IsUsernameTaken(err) {
		return &v1.ReplyResult{Code: ResultCodeUsernameTaken, Message: "username already exists", TraceId: traceID}
	}
	if apperr.IsInvalidCredentials(err) {
		return &v1.ReplyResult{Code: ResultCodeInvalidCredentials, Message: "invalid username or password", TraceId: traceID}
	}
	if apperr.IsSessionExpired(err) {
		return &v1.ReplyResult{Code: ResultCodeSessionExpired, Message: MessageSessionExpired, TraceId: traceID}
	}
	if apperr.IsTokenExpired(err) {
		return &v1.ReplyResult{Code: ResultCodeTokenExpired, Message: MessageTokenExpired, TraceId: traceID}
	}
	if apperr.IsInvalidToken(err) {
		return &v1.ReplyResult{Code: ResultCodeTokenInvalid, Message: "invalid token", TraceId: traceID}
	}
	if apperr.IsUserNotFound(err) {
		return &v1.ReplyResult{Code: ResultCodeUserNotFound, Message: "user not found", TraceId: traceID}
	}
	if apperr.IsNotFound(err) {
		return &v1.ReplyResult{Code: ResultCodeUserNotFound, Message: "not found", TraceId: traceID}
	}
	return &v1.ReplyResult{Code: ResultCodeInternalError, Message: MessageInternalError, TraceId: traceID}
}
