package server

import (
	"net/http"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func RegisterLocalAssetsHTTP(srv *httptransport.Server, rootDir string) {
	if srv == nil {
		return
	}
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = "/tmp/live-auction-assets"
	}
	fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir(rootDir)))
	srv.HandlePrefix("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(w, r)
	}))
}
