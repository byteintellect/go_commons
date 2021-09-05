package monitoring

import (
	"github.com/byteintellect/go_commons"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
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
	Name: "grpc_response_duaration",
	Help: "response duration for gRPC calls",
}, []string{"path"})

func RegisterHttpPrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		route := mux.CurrentRoute(request)
		path, _ := route.GetPathTemplate()

		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		rw := &go_commons.ResponseWriter{
			ResponseWriter: writer,
		}
		next.ServeHTTP(rw, request)
		statusCode := rw.Status()
		httpResponseStatusCode.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		httpTotalRequests.WithLabelValues(path).Inc()
		timer.ObserveDuration()
	})
}

func InitHttp() {
	prometheus.Register(httpTotalRequests)
	prometheus.Register(httpResponseStatusCode)
	prometheus.Register(httpDuration)
}

func InitGrpc() {
	prometheus.Register(grpcTotalRequests)
	prometheus.Register(gRPCDuration)
}
