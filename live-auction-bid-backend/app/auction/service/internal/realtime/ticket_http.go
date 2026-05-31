package realtime

import (
	"encoding/json"
	"net/http"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

type wsTicketRequest struct {
	RoomID string `json:"roomId"`
	Scope  string `json:"scope"`
}

type wsTicketReply struct {
	Result          wsTicketResult `json:"result"`
	Ticket          string         `json:"ticket,omitempty"`
	Scope           string         `json:"scope,omitempty"`
	ExpiresAtUnixMs int64          `json:"expiresAtUnixMs,omitempty"`
}

type wsTicketResult struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func (h *Hub) ServeTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeWSTicketError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeWSTicketError(w, http.StatusServiceUnavailable, "auth manager is not configured")
		return
	}
	var req wsTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeWSTicketError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	roomID := strings.TrimSpace(req.RoomID)
	if roomID == "" {
		writeWSTicketError(w, http.StatusBadRequest, "roomId is required")
		return
	}
	scope, ok := normalizeScope(req.Scope)
	if !ok {
		writeWSTicketError(w, http.StatusBadRequest, "invalid websocket scope")
		return
	}
	ctx, authCtx := h.authContextFromAuthorization(r.Context(), websocketAuthorization(r))
	if authCtx.TokenStatus != auth.TokenStatusValid || authCtx.Claims == nil {
		writeWSTicketError(w, http.StatusUnauthorized, "login is required")
		return
	}
	if scope == ScopeAdmin {
		if !canOpenAdminScope(authCtx.Claims) {
			writeWSTicketError(w, http.StatusForbidden, "admin websocket scope is forbidden")
			return
		}
		mainAccountID := auth.EffectiveMainAccountID(authCtx.Claims)
		if mainAccountID == "" {
			writeWSTicketError(w, http.StatusForbidden, "main account id is required")
			return
		}
		if h.roomAccess == nil {
			writeWSTicketError(w, http.StatusServiceUnavailable, "room access validator is not configured")
			return
		}
		if err := h.roomAccess.ValidateRoomInMainAccount(ctx, roomID, mainAccountID); err != nil {
			writeWSTicketError(w, http.StatusForbidden, "room access denied")
			return
		}
	}
	ticket, expiresAt, err := h.tickets.issue(wsTicketClaims{
		RoomID:          roomID,
		Scope:           scope,
		UserID:          authCtx.Claims.UserID,
		MainAccountID:   auth.EffectiveMainAccountID(authCtx.Claims),
		RoleCodes:       append([]string(nil), authCtx.Claims.RoleCodes...),
		PermissionCodes: append([]string(nil), authCtx.Claims.PermissionCodes...),
	})
	if err != nil {
		writeWSTicketError(w, http.StatusServiceUnavailable, "failed to issue websocket ticket")
		return
	}
	writeWSTicketJSON(w, http.StatusOK, wsTicketReply{
		Result:          wsTicketResult{Code: 0, Message: "OK"},
		Ticket:          ticket,
		Scope:           scope,
		ExpiresAtUnixMs: expiresAt,
	})
}

func writeWSTicketError(w http.ResponseWriter, status int, message string) {
	writeWSTicketJSON(w, status, wsTicketReply{Result: wsTicketResult{Code: int32(status), Message: message}})
}

func writeWSTicketJSON(w http.ResponseWriter, status int, payload wsTicketReply) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
