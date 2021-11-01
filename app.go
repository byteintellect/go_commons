package go_commons

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/byteintellect/go_commons/monitoring"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"io/ioutil"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"
)

var (
	requestIdCtxKey = "X-CO-RELATION-ID"
)

// ResponseWriter is a wrapper around http.ResponseWriter that provides extra information about
// the response. It is recommended that middleware handlers use this construct to wrap a response writer
// if the functionality calls for it.
type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	// Status returns the status code of the response or 0 if the response has
	// not been written
	Status() int
	// Written returns if the ResponseWriter has been written.
	Written() bool
	// Size returns the size of the response body.
	Size() int
	// Before allows for a function to be called before the ResponseWriter has been written to. This is
	// useful for setting headers or any other operations that must happen before a response has been written.
	Before(func(ResponseWriter))
}

type preCallback func(ResponseWriter)

// NewResponseWriter creates a ResponseWriter that wraps a http.ResponseWriter
func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	nrw := &responseWriter{
		ResponseWriter: rw,
	}

	if _, ok := rw.(http.CloseNotifier); ok {
		return &responseWriterCloseNotifier{nrw}
	}

	return nrw
}

type responseWriter struct {
	http.ResponseWriter
	status       int
	size         int
	preCallbacks []preCallback
}

func (rw *responseWriter) WriteHeader(s int) {
	rw.status = s
	rw.callBefore()
	rw.ResponseWriter.WriteHeader(s)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.Written() {
		// The status will be StatusOK if WriteHeader has not been called yet
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) Size() int {
	return rw.size
}

func (rw *responseWriter) Written() bool {
	return rw.status != 0
}

func (rw *responseWriter) Before(before func(ResponseWriter)) {
	rw.preCallbacks = append(rw.preCallbacks, before)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

func (rw *responseWriter) callBefore() {
	for i := len(rw.preCallbacks) - 1; i >= 0; i-- {
		rw.preCallbacks[i](rw)
	}
}

func (rw *responseWriter) Flush() {
	flusher, ok := rw.ResponseWriter.(http.Flusher)
	if ok {
		if !rw.Written() {
			// The status will be StatusOK if WriteHeader has not been called yet
			rw.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

// Deprecated: the CloseNotifier interface predates Go's context package.
// New code should use Request.Context instead.
//
// We still implement it for backwards compatibility with older versions of Go
type responseWriterCloseNotifier struct {
	*responseWriter
}

func (rw *responseWriterCloseNotifier) CloseNotify() <-chan bool {
	return rw.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

type BaseApp struct {
	Logger    *zap.Logger
	appTokens []string
}

func (a *BaseApp) validAppToken(given string) bool {
	for _, token := range a.appTokens {
		if token == given {
			return true
		}
	}
	return false
}

func (a *BaseApp) BasicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		apiKey := request.Header.Get("X-API-TOKEN")
		if apiKey != "" && a.validAppToken(apiKey) {
			next.ServeHTTP(writer, request)
		} else {
			a.WriteResp(request.Context(), errors.New("missing internal api token"), http.StatusForbidden, writer)
			return
		}
	})
}

func (a *BaseApp) LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				a.Logger.Error("application error",
					zap.Any("err", err),
					zap.String("trace", string(debug.Stack())))
			}
		}()
		start := time.Now()
		cfResponseWriter := NewResponseWriter(writer)

		// set correlation id for distributed tracing
		// example SERVICE1 makes http calls to SERVICE2, while making the call SERVICE1 should set the X-CORRELATION-ID
		// using the value of the X-CORRELATION-ID we can deduce that the parent invoker for the api in SERVICE2
		var requestId string
		if correlationId := request.Header.Get(requestIdCtxKey); correlationId != "" {
			requestId = correlationId
		} else {
			requestId = uuid.New().String()
		}
		newCtx := context.WithValue(request.Context(), requestIdCtxKey, requestId)
		a.Logger.Info(requestId,
			zap.String("method", request.Method),
			zap.Int("status", cfResponseWriter.Status()),
			zap.String("uri", request.URL.EscapedPath()),
			zap.Int64("time", int64(time.Since(start))))
		next.ServeHTTP(cfResponseWriter, request.WithContext(newCtx))
	})
}

func (a *BaseApp) CommonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (a *BaseApp) WriteResp(ctx context.Context, resp interface{}, status int, writer http.ResponseWriter) {
	defaultResp := "{\"error\":\"internal server error\"}"
	respBytes := new(bytes.Buffer)
	err := json.NewEncoder(respBytes).Encode(resp)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(defaultResp))
		if err != nil {
			a.Logger.Error(ctx.Value(requestIdCtxKey).(string), zap.String("response_to_json_serialization_err", err.Error()), zap.Error(err))
		}
	}
	writer.WriteHeader(status)
	_, err = writer.Write(respBytes.Bytes())
	if err != nil {
		a.Logger.Error(ctx.Value(requestIdCtxKey).(string), zap.String("response_writer_err", err.Error()), zap.Error(err))
	}
}

func (a *BaseApp) RequestWithBody(request *http.Request) bool {
	return request.Method == http.MethodPost || request.Method == http.MethodPatch || request.Method == http.MethodPut
}

func (a *BaseApp) RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Enable this carefully, might cause a lot of problems if you log everything
		if a.RequestWithBody(request) {
			bodyBytes, _ := ioutil.ReadAll(request.Body)
			if len(bodyBytes) > 0 {
				a.Logger.Info(request.Context().Value(requestIdCtxKey).(string), zap.String("payload", string(bodyBytes)))
				request.Body.Close() //  must close
				request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func (a *BaseApp) RegisterHttpPrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		route := mux.CurrentRoute(request)
		path, _ := route.GetPathTemplate()

		timer := prometheus.NewTimer(monitoring.HttpDuration.WithLabelValues(path))
		rw := NewResponseWriter(writer)
		next.ServeHTTP(rw, request)
		statusCode := rw.Status()
		monitoring.HttpResponseStatusCode.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		monitoring.HttpTotalRequests.WithLabelValues(path).Inc()
		timer.ObserveDuration()
	})
}

func NewBaseApp(logger *zap.Logger, appTokens []string) *BaseApp {
	return &BaseApp{
		Logger:    logger,
		appTokens: appTokens,
	}
}
