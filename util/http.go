package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Http struct {
	client         *http.Client
	logger         *logrus.Logger
	defaultHeaders map[string]string
	tracer         *traceSdk.TracerProvider
}

func NewHttp(client *http.Client, logger *logrus.Logger, defaultHeaders map[string]string, tracer *traceSdk.TracerProvider) *Http {
	return &Http{
		client:         client,
		logger:         logger,
		defaultHeaders: defaultHeaders,
		tracer:         tracer,
	}
}

func GetQueryParamsString(queryParams map[string]string) string {
	var qpSlice []string
	for key, value := range queryParams {
		qpSlice = append(qpSlice, fmt.Sprintf("%v=%v", key, value))
	}
	return strings.Join(qpSlice, "&")
}

type HttpResponseMapper func(resBytes []byte) (interface{}, error)

type Factory func() interface{}

type ReqOption func(r *http.Request)

func NewGetReq(baseUri string) (*http.Request, error) {
	return http.NewRequest(http.MethodGet, baseUri, nil)
}

func NewPostReq(baseUri string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(http.MethodPost, baseUri, body)
}

func NewPutReq(baseUri string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(http.MethodPut, baseUri, body)
}

func NewPatchReq(baseUri string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(http.MethodPatch, baseUri, body)
}

func WithQueryParams(queryParams map[string]string) ReqOption {
	return func(r *http.Request) {
		qStr := GetQueryParamsString(queryParams)
		if len(qStr) > 0 {
			r.RequestURI += "?" + qStr
		}
	}
}

func WithHeaders(headers map[string][]string) ReqOption {
	return func(r *http.Request) {
		for key, value := range headers {
			r.Header[key] = value
		}
	}
}

func WithBody(bodyFactory func() []byte) ReqOption {
	return func(r *http.Request) {
		reqBytes := bodyFactory()
		r.GetBody = func() (io.ReadCloser, error) {
			rBytes := bytes.NewReader(reqBytes)
			return ioutil.NopCloser(rBytes), nil
		}
	}
}

func WithCtx(context context.Context) ReqOption {
	return func(r *http.Request) {
		r.WithContext(context)
	}
}

func (hul *Http) DoOperation(req *http.Request, factoryFunc Factory, reqOptions ...ReqOption) error {
	for _, option := range reqOptions {
		option(req)
	}
	defer hul.recordSpan(req)
	resp, err := hul.client.Do(req)
	if err != nil {
		return err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	entity := factoryFunc()
	err = json.Unmarshal(respBytes, entity)
	if err != nil {
		return err
	}
	return nil
}

func (hul *Http) unmarshal(factoryFunc Factory, response *http.Response) error {
	defer response.Body.Close()
	respBytes, err := ioutil.ReadAll(response.Body)
	entity := factoryFunc()
	err = json.Unmarshal(respBytes, entity)
	if err != nil {
		return err
	}
	return nil
}

func (hul *Http) DoGet(uri string, factoryFunc Factory, reqOptions ...ReqOption) error {
	req, err := NewGetReq(uri)
	for _, option := range reqOptions {
		option(req)
	}
	defer hul.recordSpan(req)
	resp, err := hul.client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return hul.unmarshal(factoryFunc, resp)
}

func (hul *Http) DoPost(uri string, factoryFunc Factory, bodyFunc func() []byte, reqOptions ...ReqOption) error {
	req, err := NewPostReq(uri, bytes.NewReader(bodyFunc()))
	for _, option := range reqOptions {
		option(req)
	}
	defer hul.recordSpan(req)
	res, err := hul.client.Do(req)
	if err != nil {
		return err
	}
	return hul.unmarshal(factoryFunc, res)
}

func (hul *Http) recordSpan(req *http.Request) {
	tr := hul.tracer.Tracer(os.Getenv("APP_NAME"))
	_, span := tr.Start(req.Context(), req.RequestURI)
	defer span.End()
}
