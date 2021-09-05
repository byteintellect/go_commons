package go_commons

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
	requestSize  int64
}

func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}

type BaseApp struct {
	AppTokens []string
}

func (a *BaseApp) validAppToken(given string) bool {
	for _, token := range a.AppTokens {
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
			a.WriteResp(errors.New("missing internal api token"), http.StatusForbidden, writer)
			return
		}
	})
}

func (a *BaseApp) LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				zap.S().Error("application error",
					zap.Any("err", err),
					zap.String("trace", string(debug.Stack())))
			}
		}()
		start := time.Now()
		cfResponse := &ResponseWriter{
			ResponseWriter: writer,
		}
		next.ServeHTTP(writer, request)
		requestId := uuid.New()
		zap.S().Info(requestId.String(),
			zap.String("method", request.Method),
			zap.Int("status", cfResponse.statusCode),
			zap.String("uri", request.URL.EscapedPath()),
			zap.Int64("time", int64(time.Since(start))))
	})
}

func (a *BaseApp) WriteResp(resp interface{}, status int, writer http.ResponseWriter) {
	defaultResp := "{\"error\":\"internal server error\"}"
	respBytes := new(bytes.Buffer)
	err := json.NewEncoder(respBytes).Encode(resp)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(defaultResp))
		if err != nil {
			log.Printf("error %v while writing default response \n", err)
		}
	}
	writer.WriteHeader(status)
	_, err = writer.Write(respBytes.Bytes())
	if err != nil {
		log.Printf("error %v while writing response \n", err)
	}
}
