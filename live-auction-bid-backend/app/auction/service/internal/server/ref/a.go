//go:build ignore
// +build ignore

package server

import (
	v2 "odin/api/loli/service/v2"
	"odin/app/loli/service/internal/conf"
	"odin/app/loli/service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	kmetrics "github.com/go-kratos/prometheus/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// NewHTTPServer new a HTTP server.
func NewHTTPServer(c *conf.Server, logger log.Logger, tp *tracesdk.TracerProvider, s *service.UserService) *http.Server {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "loli_counter"}, []string{"method", "server", "code", "label"})
	prometheus.MustRegister(
		counter,
	)
	var opts = []http.ServerOption{
		http.Middleware(
			middleware.Chain(
				RecoveryWithArgs(),
				metrics.Server(metrics.WithRequests(kmetrics.NewCounter(counter))),
				tracing.Server(
					tracing.WithTracerProvider(tp),
					tracing.WithPropagator(
						propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}),
					),
				),
				server(logger),
			),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v2.RegisterUserServiceHTTPServer(srv, s)
	v2.RegisterFriendServiceHTTPServer(srv, s)
	v2.RegisterBuildingServiceHTTPServer(srv, s)
	v2.RegisterCableCarServiceHTTPServer(srv, s)
	v2.RegisterTrainServiceHTTPServer(srv, s)
	v2.RegisterMailServiceHTTPServer(srv, s)
	v2.RegisterExploreServiceHTTPServer(srv, s)
	v2.RegisterPhoneServiceHTTPServer(srv, s)
	v2.RegisterDressUpServiceHTTPServer(srv, s)
	v2.RegisterStoreServiceHTTPServer(srv, s)
	v2.RegisterGuildServiceHTTPServer(srv, s)
	v2.RegisterTaskServiceHTTPServer(srv, s)
	v2.RegisterTestUserServiceHTTPServer(srv, s)
	v2.RegisterMinigameServiceHTTPServer(srv, s)
	v2.RegisterBatchServiceHTTPServer(srv, s)
	v2.RegisterNewsTickerServiceHTTPServer(srv, s)
	v2.RegisterBuildingFunctionServiceHTTPServer(srv, s)
	v2.RegisterAchieveServiceHTTPServer(srv, s)
	v2.RegisterActivityServiceHTTPServer(srv, s)
	v2.RegisterVillagerServiceHTTPServer(srv, s)
	v2.RegisterCardServiceHTTPServer(srv, s)
	v2.RegisterAirplaneServiceHTTPServer(srv, s)
	v2.RegisterMilestoneServiceHTTPServer(srv, s)
	v2.RegisterDispatchServiceHTTPServer(srv, s)

	v2.RegisterWorkerServiceHTTPServer(srv, s)
	v2.RegisterFishingServiceHTTPServer(srv, s)
	v2.RegisterAdvertisementServiceHTTPServer(srv, s)
	return srv
}

// wire放providerSet时，必须定义一个新的类型，不能和原有类型重复
type HttpMetrics http.Server

func NewMetricsHTTPServer(c *conf.Server, s *service.UserService) *HttpMetrics {
	var opts = []http.ServerOption{}
	if c.Metrics.Network != "" {
		opts = append(opts, http.Network(c.Metrics.Network))
	}
	if c.Metrics.Addr != "" {
		opts = append(opts, http.Address(c.Metrics.Addr))
	}
	if c.Metrics.Timeout != nil {
		opts = append(opts, http.Timeout(c.Metrics.Timeout.AsDuration()))
	}

	srv := http.NewServer(opts...)

	srv.HandlePrefix("/metrics", promhttp.Handler())

	return (*HttpMetrics)(srv)
}
