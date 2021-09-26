package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var httpTotalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "number of http requests",
}, []string{"path"})

var grpcTotalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "grpc_requests_total",
	Help: "number of gRPC requests",
}, []string{"path"})

var httpResponseStatusCode = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_response_status_code",
	Help: "http calls response status code",
}, []string{"status"})

var httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_duration",
	Help: "response duration for http calls",
}, []string{"path"})

var gRPCDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "grpc_response_duration",
	Help: "response duration for gRPC calls",
}, []string{"path"})

func InitHttp(registry *prometheus.Registry) {
	registry.MustRegister(httpTotalRequests)
	registry.MustRegister(httpResponseStatusCode)
	registry.MustRegister(httpDuration)
}

func InitGrpc(registry *prometheus.Registry) {
	registry.MustRegister(grpcTotalRequests)
	registry.MustRegister(gRPCDuration)
}
