package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/biz"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
)

type Server struct { app *biz.Service; hub *realtime.Hub }
func New(app *biz.Service, hub *realtime.Hub) *Server { return &Server{app: app, hub: hub} }
func (s *Server) Handler() http.Handler { mux := http.NewServeMux(); mux.HandleFunc("/healthz", s.health); mux.HandleFunc("/api/lots", s.lots); mux.HandleFunc("/api/lots/", s.lotAction); mux.HandleFunc("/api/rooms/", s.roomAction); mux.HandleFunc("/ws/rooms/", s.wsRoom); mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){ writeJSON(w, map[string]string{"service":"auction-backend","status":"ok"}) }); return cors(mux) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]bool{"ok": true}) }
func (s *Server) lots(w http.ResponseWriter, r *http.Request) { switch r.Method { case http.MethodGet: status := biz.LotStatus(r.URL.Query().Get("status")); lots, err := s.app.ListLots(r.Context(), r.URL.Query().Get("roomId"), status); write(w,lots,err,200); case http.MethodPost: var cmd biz.CreateLotCommand; if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil { http.Error(w,err.Error(),400); return }; lot, err := s.app.CreateLot(r.Context(), cmd); write(w, map[string]any{"lot":lot}, err, 201); default: http.NotFound(w,r) } }
func (s *Server) lotAction(w http.ResponseWriter, r *http.Request) { parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/lots/"),"/"),"/"); if len(parts)<1 || parts[0]=="" { http.NotFound(w,r); return }; lotID := parts[0]; action := ""; if len(parts)>1 { action = parts[1] }; switch { case r.Method==http.MethodGet && action=="": lot,err:=s.app.GetLot(r.Context(),lotID); write(w,map[string]any{"lot":lot},err,200); case r.Method==http.MethodPost && action=="start": lot,err:=s.app.StartLot(r.Context(),lotID); write(w,map[string]any{"lot":lot},err,200); case r.Method==http.MethodPost && action=="bid": var cmd biz.PlaceBidCommand; if err:=json.NewDecoder(r.Body).Decode(&cmd); err!=nil { http.Error(w,err.Error(),400); return }; cmd.LotID=lotID; lot,bid,ranking,err:=s.app.PlaceBid(r.Context(),cmd); write(w,map[string]any{"accepted":err==nil,"lot":lot,"bid":bid,"ranking":ranking,"rejectReason":errText(err)},err,200); case r.Method==http.MethodPost && len(parts)>=4 && parts[1]=="trust-cards" && parts[3]=="reveal": lot,card,err:=s.app.RevealTrustCard(r.Context(),lotID,parts[2],""); write(w,map[string]any{"lot":lot,"trustCard":card},err,200); case r.Method==http.MethodPost && action=="duel": lot,duel,err:=s.app.StartDuel(r.Context(),lotID,"","",""); write(w,map[string]any{"lot":lot,"duelState":duel},err,200); case r.Method==http.MethodPost && action=="settle": lot,err:=s.app.SettleLot(r.Context(),lotID,""); write(w,map[string]any{"lot":lot},err,200); default: http.NotFound(w,r) } }
func (s *Server) roomAction(w http.ResponseWriter, r *http.Request) { parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/rooms/"),"/"),"/"); if len(parts)==2 && parts[1]=="snapshot" { snap,err:=s.app.Snapshot(r.Context(),parts[0]); write(w,map[string]any{"snapshot":snap},err,200); return }; http.NotFound(w,r) }
func (s *Server) wsRoom(w http.ResponseWriter, r *http.Request) { roomID := strings.Trim(strings.TrimPrefix(r.URL.Path,"/ws/rooms/"),"/"); if roomID=="" { roomID="demo" }; s.hub.ServeRoom(w,r,roomID) }
func write(w http.ResponseWriter, v any, err error, ok int) { if err != nil { http.Error(w,err.Error(),409); return }; writeJSON(w,v) }
func writeJSON(w http.ResponseWriter, v any) { w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(v) }
func errText(err error) string { if err==nil { return "" }; return err.Error() }
func cors(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ w.Header().Set("Access-Control-Allow-Origin","*"); w.Header().Set("Access-Control-Allow-Headers","Content-Type"); w.Header().Set("Access-Control-Allow-Methods","GET,POST,OPTIONS"); if r.Method==http.MethodOptions { return }; next.ServeHTTP(w,r) }) }
