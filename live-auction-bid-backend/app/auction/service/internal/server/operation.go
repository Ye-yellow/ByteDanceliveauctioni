package server

import (
	"encoding/json"
	"net/http"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func registerOperationHTTP(srv *httptransport.Server) {
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "auction-backend", "transport": "kratos-http", "status": "ok"})
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
