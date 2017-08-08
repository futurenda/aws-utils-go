package proxyIntegration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Request represents an HTTP request received by an API Gateway proxy integrations.
type Request struct {
	Body                  string            `json:"body"`
	Headers               map[string]string `json:"headers"`
	HTTPMethod            string            `json:"httpMethod"`
	Path                  string            `json:"path"`
	PathParameters        map[string]string `json:"pathParameters"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	Resource              string            `json:"resource"`
	StageVariables        map[string]string `json:"stageVariables"`
	RequestContext        RequestContext    `json:"requestContext"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
}

// NewRequest creates *net/http.Request from a Request.
func NewRequest(event json.RawMessage) (*http.Request, error) {
	var r Request
	if err := json.Unmarshal(event, &r); err != nil {
		return nil, err
	}
	return r.httpRequest()
}

func (r Request) httpRequest() (*http.Request, error) {
	header := make(http.Header)
	for key, value := range r.Headers {
		header.Add(key, value)
	}
	host := header.Get("Host")
	header.Del("Host")
	v := make(url.Values)
	for key, value := range r.QueryStringParameters {
		v.Add(key, value)
	}
	uri := r.Path
	if len(r.QueryStringParameters) > 0 {
		uri = uri + "?" + v.Encode()
	}
	u, _ := url.Parse(uri)
	var contentLength int64
	var b io.Reader
	if r.IsBase64Encoded {
		raw := make([]byte, len(r.Body))
		n, err := base64.StdEncoding.Decode(raw, []byte(r.Body))
		if err != nil {
			return nil, err
		}
		contentLength = int64(n)
		b = bytes.NewReader(raw[0:n])
	} else {
		contentLength = int64(len(r.Body))
		b = strings.NewReader(r.Body)
	}
	req := http.Request{
		Method:        r.HTTPMethod,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		ContentLength: contentLength,
		Body:          ioutil.NopCloser(b),
		RemoteAddr:    r.RequestContext.Identity["sourceIp"],
		Host:          host,
		RequestURI:    uri,
		URL:           u,
	}
	return &req, nil
}

// RequestContext represents request contest object.
type RequestContext struct {
	AccountID    string            `json:"accountId"`
	ApiID        string            `json:"apiId"`
	HTTPMethod   string            `json:"httpMethod"`
	Identity     map[string]string `json:"identity"`
	RequestID    string            `json:"requestId"`
	ResourceID   string            `json:"resourceId"`
	ResourcePath string            `json:"resourcePath"`
	Stage        string            `json:"stage"`
}

// Response represents a response for API Gateway proxy integration.
type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// NewResponseWriter creates ResponseWriter
func NewResponseWriter() *ResponseWriter {
	var b bytes.Buffer
	w := &ResponseWriter{
		Buffer:     &b,
		statusCode: http.StatusOK,
		header:     make(http.Header),
	}
	return w
}

// ResponeWriter represents a response writer implements http.ResponseWriter.
type ResponseWriter struct {
	*bytes.Buffer
	header     http.Header
	statusCode int
}

func (w *ResponseWriter) Header() http.Header {
	return w.header
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.statusCode = code
}

func (w *ResponseWriter) Response() Response {
	h := make(map[string]string, len(w.header))
	for key := range w.header {
		h[key] = w.header.Get(key)
	}
	return Response{
		StatusCode: w.statusCode,
		Headers:    h,
		Body:       w.String(),
	}
}
