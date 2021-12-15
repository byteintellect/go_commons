package go_commons

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/byteintellect/go_commons/config"
	"github.com/byteintellect/go_commons/db"
	"github.com/byteintellect/go_commons/logger"
	"github.com/byteintellect/go_commons/monitoring"
	"github.com/byteintellect/go_commons/tracing"
	"github.com/google/uuid"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/infobloxopen/atlas-app-toolkit/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

var (
	requestIdCtxKey   = "X-CO-RELATION-ID"
	httpPatternCtxKey = "X-HTTP-PATH"
	gRPCMethodCtxKey  = "X-GRPC-HANDLER-METHOD"
	serviceName       = fmt.Sprintf("%v_%v", os.Getenv("APP_NAME"), os.Getenv("APP_ENV"))
	metricsPath       = "/metrics"
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

func GatewayOpts(cfg *config.BaseConfig, endPointFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)) ([]gateway.Option, error) {
	return []gateway.Option{
		gateway.WithGatewayOptions(
			runtime.WithMetadata(func(ctx context.Context, request *http.Request) metadata.MD {
				md := make(map[string]string)
				if method, ok := runtime.RPCMethod(ctx); ok {
					md["method"] = method
					request.Header.Add("x-grpc-handler-method", method)
				}
				if pattern, ok := runtime.HTTPPathPattern(ctx); ok {
					md["pattern"] = pattern
					request.Header.Add("x-http-path", pattern)
				}
				return metadata.New(md)
			}),
			runtime.WithForwardResponseOption(forwardResponseOption),
			runtime.WithIncomingHeaderMatcher(gateway.AtlasDefaultHeaderMatcher()),
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					AllowPartial:   false,
					DiscardUnknown: true,
				},
			}),
		),
		gateway.WithServerAddress(fmt.Sprintf("%s:%s", cfg.ServerConfig.Address, cfg.ServerConfig.Port)),
		gateway.WithEndpointRegistration(cfg.GatewayConfig.Endpoint, endPointFunc),
	}, nil
}

type BaseApp struct {
	logger      *zap.Logger
	registry    *prometheus.Registry
	tracer      *traceSdk.TracerProvider
	db          *gorm.DB
	ctx         context.Context
	grpcMetrics *grpcPrometheus.ServerMetrics
	appTokens   []string
}

func (a *BaseApp) GrpcMetrics() *grpcPrometheus.ServerMetrics {
	return a.grpcMetrics
}

func (a *BaseApp) Logger() *zap.Logger {
	return a.logger
}

func (a *BaseApp) Registry() *prometheus.Registry {
	return a.registry
}

func (a *BaseApp) Tracer() *traceSdk.TracerProvider {
	return a.tracer
}

func (a *BaseApp) Db() *gorm.DB {
	return a.db
}

func (a *BaseApp) Ctx() context.Context {
	return a.ctx
}

func (a *BaseApp) AppTokens() []string {
	return a.appTokens
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
				a.logger.Error("application error",
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
		a.logger.Info(requestId,
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
			a.logger.Error(ctx.Value(requestIdCtxKey).(string), zap.String("response_to_json_serialization_err", err.Error()), zap.Error(err))
		}
	}
	writer.WriteHeader(status)
	_, err = writer.Write(respBytes.Bytes())
	if err != nil {
		a.logger.Error(ctx.Value(requestIdCtxKey).(string), zap.String("response_writer_err", err.Error()), zap.Error(err))
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
				a.logger.Info(request.Context().Value(requestIdCtxKey).(string), zap.String("payload", string(bodyBytes)))
				request.Body.Close() //  must close
				request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func (a *BaseApp) HandlerWithMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rw := NewResponseWriter(writer)
		defer func() {
			reqPath := request.Header.Get(httpPatternCtxKey)
			if reqPath != metricsPath && reqPath != "" {
				timer := prometheus.NewTimer(monitoring.HttpDuration.WithLabelValues(serviceName, reqPath, request.Method))
				statusCode := rw.Status()
				monitoring.HttpTotalRequests.WithLabelValues(serviceName, reqPath, request.Method, strconv.Itoa(statusCode)).Inc()
				monitoring.HttpResponseStatusCode.WithLabelValues(serviceName, reqPath, request.Method).Inc()
				timer.ObserveDuration()
			}
		}()
		next.ServeHTTP(rw, request)
	})
}

func getDbDSN(cfg *config.BaseConfig) string {
	dCfg := cfg.DatabaseConfig
	return fmt.Sprintf("%v:%v@(%v:%v)/%v?parseTime=true",
		dCfg.UserName, dCfg.Password, dCfg.HostName, dCfg.Port, dCfg.DatabaseName)
}

func NewBaseApp(cfg *config.BaseConfig) (*BaseApp, error) {

	// Initialize Logger
	zapLogger, err := logger.InitLogger()
	if err != nil {
		log.Println("failed to initialize logger")
		return nil, err
	}
	promRegistry := prometheus.NewRegistry()
	grpcMetrics := grpcPrometheus.NewServerMetrics()
	promRegistry.MustRegister(grpcMetrics)

	// Initialize Trace Provider connection
	traceProvider, err := tracing.NewTracer(cfg.TraceProviderUrl)
	if err != nil {
		zapLogger.Error("failed to initialize app due to trace provider", zap.Error(err))
		return nil, err
	}

	// Initialize context
	ctx := context.Background()

	database, err := db.NewGormDbConn(getDbDSN(cfg), traceProvider)
	if err != nil {
		zapLogger.Error("failed to initialize app due to db connection", zap.Error(err))
		return nil, err
	}

	return &BaseApp{
		logger:      zapLogger,
		appTokens:   cfg.AppTokens,
		ctx:         ctx,
		db:          database,
		tracer:      traceProvider,
		registry:    promRegistry,
		grpcMetrics: grpcMetrics,
	}, nil
}

func forwardResponseOption(ctx context.Context, w http.ResponseWriter, resp protoreflect.ProtoMessage) error {
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	// set http status code
	if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
		code, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		// delete the headers to not expose any grpc-metadata in http response
		delete(md.HeaderMD, "x-http-code")
		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
		w.WriteHeader(code)
	}
	return nil
}

func ServeExternal(cfg *config.BaseConfig, app *BaseApp, grpcServer *grpc.Server, endPointFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)) error {
	gatewayOpts, err := GatewayOpts(cfg, endPointFunc)
	if err != nil {
		return err
	}
	s, err := server.NewServer(
		server.WithGrpcServer(grpcServer),
		server.WithGateway(gatewayOpts...),
		// this endpoint will be used for our health checks
		server.WithHandler(fmt.Sprintf("/%v/ping", os.Getenv("APP_NAME")), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("pong"))
		})),
		server.WithHandler(fmt.Sprintf("/%v/ready", os.Getenv("APP_NAME")), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := app.db.Raw("SELECT 1").Error; err != nil {
				w.WriteHeader(http.StatusBadGateway)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{\"status\": \"ok\"}"))
		})),
		// register middlewares
		server.WithMiddlewares(app.LogMiddleware, app.RequestLoggerMiddleware, app.CommonMiddleware),
		// register metrics
		server.WithHandler(fmt.Sprintf("/%v/metrics", os.Getenv("APP_NAME")), promhttp.HandlerFor(app.registry, promhttp.HandlerOpts{Registry: app.Registry()})),
	)
	if err != nil {
		return err
	}

	// wrap handler
	monitoring.InitHttp(app.registry)
	s.HTTPServer.Handler = app.HandlerWithMetrics(s.HTTPServer.Handler)

	grpcL, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.ServerConfig.Address, cfg.ServerConfig.Port))
	if err != nil {
		app.logger.Fatal("Error starting gRPC listener for clients", zap.Error(err))
	}

	httpL, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.GatewayConfig.Address, cfg.GatewayConfig.Port))
	if err != nil {
		app.logger.Fatal("Error starting http listener for client", zap.Error(err))
	}
	app.logger.Info("serving gRPC ", zap.String("address", cfg.ServerConfig.Address), zap.String("port", cfg.ServerConfig.Port))
	app.logger.Info("serving http", zap.String("address", cfg.GatewayConfig.Address), zap.String("port", cfg.GatewayConfig.Port))
	return s.Serve(grpcL, httpL)
}
