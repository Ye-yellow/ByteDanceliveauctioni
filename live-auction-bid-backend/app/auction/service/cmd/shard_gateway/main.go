package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"live-auction-bid/backend/app/auction/service/internal/cluster"
)

func main() {
	addr := getenv("AUCTION_GATEWAY_ADDR", ":18081")
	registry, err := cluster.ParseStaticRegistryJSON(os.Getenv("AUCTION_CLUSTER_REGISTRY_JSON"))
	if err != nil {
		log.Fatalf("parse AUCTION_CLUSTER_REGISTRY_JSON failed: %v", err)
	}
	routes, closeRoutes, err := newRouteTable(registry)
	if err != nil {
		log.Fatalf("create route table failed: %v", err)
	}
	defer closeRoutes()
	gateway, err := cluster.NewGateway(registry, routes)
	if err != nil {
		log.Fatalf("create shard gateway failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"service":"auction-shard-gateway"}`))
	}))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/clusterz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(gateway.ClusterSnapshot(r.Context()))
	}))
	registerAdminHTTP(mux, gateway, os.Getenv("AUCTION_GATEWAY_ADMIN_TOKEN"))
	mux.Handle("/", gateway)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("auction shard gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func newRouteTable(registry *cluster.StaticRegistry) (cluster.RouteTable, func(), error) {
	redisAddr := getenv("AUCTION_GATEWAY_ROUTE_REDIS_ADDR", os.Getenv("AUCTION_REDIS_ADDR"))
	if redisAddr == "" {
		routes, err := cluster.NewMemoryRouteTable(registry)
		return routes, func() {}, err
	}
	routes, err := cluster.NewRedisRouteTable(registry, cluster.RedisRouteConfig{
		Addr:      redisAddr,
		Password:  getenv("AUCTION_GATEWAY_ROUTE_REDIS_PASSWORD", os.Getenv("AUCTION_REDIS_PASSWORD")),
		KeyPrefix: getenv("AUCTION_GATEWAY_ROUTE_PREFIX", "auction:route"),
	})
	if err != nil {
		return nil, func() {}, err
	}
	return routes, func() { _ = routes.Close() }, nil
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func shutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
