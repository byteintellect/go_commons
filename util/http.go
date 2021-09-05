package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Http struct {
	Client         *http.Client
	Logger         *logrus.Logger
	DefaultHeaders map[string]string
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

func (hul *Http) DoOperation(req *http.Request, factoryFunc Factory, reqOptions ...ReqOption) error {
	for _, option := range reqOptions {
		option(req)
	}
	resp, err := hul.Client.Do(req)
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
	resp, err := hul.Client.Do(req)
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
	res, err := hul.Client.Do(req)
	if err != nil {
		return err
	}

	return hul.unmarshal(factoryFunc, res)
}
