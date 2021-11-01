package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var HttpTotalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "number of http requests",
}, []string{"path"})

var GRPCTotalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "grpc_requests_total",
	Help: "number of gRPC requests",
}, []string{"path"})

var HttpResponseStatusCode = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_response_status_code",
	Help: "http calls response status code",
}, []string{"status"})

var HttpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_duration",
	Help: "response duration for http calls",
}, []string{"path"})

var GRPCDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "grpc_response_duration",
	Help: "response duration for gRPC calls",
}, []string{"path"})

func InitHttp(registry *prometheus.Registry) {
	registry.MustRegister(HttpTotalRequests)
	registry.MustRegister(HttpResponseStatusCode)
	registry.MustRegister(HttpDuration)
}

func InitGrpc(registry *prometheus.Registry) {
	registry.MustRegister(GRPCTotalRequests)
	registry.MustRegister(GRPCDuration)
}
