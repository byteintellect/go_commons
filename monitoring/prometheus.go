package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var HttpTotalRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "number of http requests",
}, []string{"service", "path", "method", "code"})

var HttpResponseStatusCode = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_response_status_code",
	Help: "http calls response status code",
}, []string{"service", "path", "method"})

var HttpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_duration",
	Help: "response duration for http calls",
}, []string{"service", "path", "method"})

func InitHttp(registry *prometheus.Registry) {
	registry.MustRegister(HttpTotalRequests)
	registry.MustRegister(HttpResponseStatusCode)
	registry.MustRegister(HttpDuration)
}
