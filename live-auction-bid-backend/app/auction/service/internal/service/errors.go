package service

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const (
	ResultCodeOK                 int32 = 0
	ResultCodeInvalidArgument    int32 = 400001
	ResultCodeUnauthenticated    int32 = 401001
	ResultCodePermissionDenied   int32 = 403001
	ResultCodeLotVersionConflict int32 = 409001
	ResultCodeUsernameTaken      int32 = 409002
	ResultCodeInvalidCredentials int32 = 401002
	ResultCodeInvalidToken       int32 = 401003
	ResultCodeUserNotFound       int32 = 404001
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
	if apperr.IsInvalidArgument(err) {
		return &v1.ReplyResult{Code: ResultCodeInvalidArgument, Message: err.Error()}
	}
	if apperr.IsUnauthenticated(err) {
		return &v1.ReplyResult{Code: ResultCodeUnauthenticated, Message: "login required"}
	}
	if apperr.IsPermissionDenied(err) {
		return &v1.ReplyResult{Code: ResultCodePermissionDenied, Message: "permission denied"}
	}
	if apperr.IsUsernameTaken(err) {
		return &v1.ReplyResult{Code: ResultCodeUsernameTaken, Message: "username already exists"}
	}
	if apperr.IsInvalidCredentials(err) {
		return &v1.ReplyResult{Code: ResultCodeInvalidCredentials, Message: "invalid username or password"}
	}
	if apperr.IsInvalidToken(err) {
		return &v1.ReplyResult{Code: ResultCodeInvalidToken, Message: "invalid token"}
	}
	if apperr.IsUserNotFound(err) {
		return &v1.ReplyResult{Code: ResultCodeUserNotFound, Message: "user not found"}
	}
	return &v1.ReplyResult{Code: ResultCodeInternalError, Message: err.Error()}
}
