package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/zloevil/jet"
	"io"
	"mime/multipart"
	"net/http"
	nUrl "net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	Authorization = "Authorization"
	ContentType   = "Content-Type"
)

type Request struct {
	headers         map[string]string
	contentLength   *int64
	method          string
	url             string
	payload         io.Reader
	cl              HttpClient
	err             error
	extendedLog     bool
	metricsProvider MetricsProvider
	respFn          func(data []byte) error
	errCodeFn       func(ctx context.Context, code int, data []byte) error
	logger          jet.CLogger
	name            string
}

func NewRq() *Request {
	return &Request{
		headers: map[string]string{},
		method:  http.MethodGet,
		respFn: func(data []byte) error {
			return nil
		},
		errCodeFn: func(ctx context.Context, code int, data []byte) error {
			return ErrHttpRequestInvalidStatusCode(ctx, code, string(data))
		},
	}
}

func (r *Request) ExtendedLog() *Request {
	r.extendedLog = true
	return r
}

func (r *Request) Metrics(provider MetricsProvider) *Request {
	r.metricsProvider = provider
	return r
}

func (r *Request) WithLogger(logger jet.CLogger) *Request {
	r.logger = logger
	return r
}

func (r *Request) Name(name string) *Request {
	r.name = name
	return r
}

func (r *Request) Mth(method string) *Request {
	r.method = method
	return r
}

func (r *Request) Cl(cl HttpClient) *Request {
	r.cl = cl
	return r
}

// Fn empty fn is default
func (r *Request) Fn(fn func(data []byte) error) *Request {
	r.respFn = fn
	return r
}

func (r *Request) ErrCodeFn(fn func(ctx context.Context, code int, data []byte) error) *Request {
	r.errCodeFn = fn
	return r
}

// FnJson sets json function as out parses
func (r *Request) FnJson(out any) *Request {
	r.respFn = func(data []byte) error {
		return json.Unmarshal(data, &out)
	}
	return r
}

func (r *Request) AuthBasic(user, pass string) *Request {
	r.headers[Authorization] = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
	return r
}

func (r *Request) ContentLength(length int64) *Request {
	r.contentLength = &length
	return r
}

func (r *Request) AuthBearer(token string) *Request {
	r.headers[Authorization] = "Bearer " + token
	return r
}

func (r *Request) ContType(content string) *Request {
	r.headers[ContentType] = content
	return r
}

// Get this is default by preset
func (r *Request) Get() *Request {
	return r.Mth(http.MethodGet)
}
func (r *Request) Post() *Request {
	return r.Mth(http.MethodPost)
}
func (r *Request) Delete() *Request {
	return r.Mth(http.MethodDelete)
}
func (r *Request) Connect() *Request {
	return r.Mth(http.MethodConnect)
}
func (r *Request) Head() *Request {
	return r.Mth(http.MethodHead)
}
func (r *Request) Put() *Request {
	return r.Mth(http.MethodPut)
}
func (r *Request) Patch() *Request {
	return r.Mth(http.MethodPatch)
}
func (r *Request) Options() *Request {
	return r.Mth(http.MethodOptions)
}
func (r *Request) Trace() *Request {
	return r.Mth(http.MethodTrace)
}

func (r *Request) H(name, value string) *Request {
	r.headers[name] = value
	return r
}

func (r *Request) Url(url fmt.Stringer) *Request {
	r.url = url.String()
	return r
}

func (r *Request) Payload(payload io.Reader) *Request {
	r.payload = payload
	return r
}

func (r *Request) Stream(ctx context.Context) (*StreamReadCloser, error) {
	if err := r.validate(ctx); err != nil {
		return nil, err
	}
	// create request
	req, err := http.NewRequest(r.method, r.url, r.payload)
	if err != nil {
		return nil, ErrHttpRequestNew(ctx, err)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	if r.contentLength != nil {
		req.ContentLength = *r.contentLength
	}
	// call http integration
	start := jet.Now()
	resp, err := r.cl.Do(req)
	if err != nil {
		return nil, ErrHttpRequestDo(ctx, err)
	}

	r.log(ctx, start, req, resp, nil)
	r.metrics(ctx, start, req, resp)
	// error processing
	if resp.StatusCode >= http.StatusMultipleChoices {
		// close response here
		defer resp.Body.Close()
		// read all data
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, ErrHttpRequestReadAll(ctx, err)
		}
		// return failed response
		return nil, r.errCodeFn(ctx, resp.StatusCode, data)
	}
	return &StreamReadCloser{
		Content:       resp.Body, // should be closed by the client's method externally
		ContentLength: resp.ContentLength,
	}, nil
}

func (r *Request) Do(ctx context.Context) error {
	if err := r.validate(ctx); err != nil {
		return err
	}
	// create request
	req, err := http.NewRequest(r.method, r.url, r.payload)
	if err != nil {
		return ErrHttpRequestNew(ctx, err)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	if r.contentLength != nil {
		req.ContentLength = *r.contentLength
	}
	// call http integration
	start := jet.Now()
	resp, err := r.cl.Do(req)
	if err != nil {
		return ErrHttpRequestDo(ctx, err)
	}
	defer resp.Body.Close()
	// read all data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrHttpRequestReadAll(ctx, err)
	}
	r.log(ctx, start, req, resp, data)
	r.metrics(ctx, start, req, resp)
	// error processing
	if resp.StatusCode >= http.StatusMultipleChoices {
		// return failed response
		return r.errCodeFn(ctx, resp.StatusCode, data)
	}
	// response processing
	if err = r.respFn(data); err != nil {
		// return failed response
		return ErrHttpRequestResponseFuncFailed(ctx, err)
	}
	return nil
}

func (r *Request) validate(ctx context.Context) error {
	if r.url == "" {
		return ErrHttpRequestEmptyUrl(ctx)
	}
	if r.method == "" {
		return ErrHttpRequestEmptyMethod(ctx)
	}
	if r.cl == nil {
		return ErrHttpRequestEmptyClient(ctx)
	}
	if r.respFn == nil {
		return ErrHttpRequestEmptyResponseFunc(ctx)
	}
	if _, err := nUrl.Parse(r.url); err != nil {
		return ErrHttpRequestInvalidUrl(ctx, err)
	}
	return nil
}

type StreamReadCloser struct {
	Content       io.ReadCloser // Content should be closed by the client's method externally
	ContentLength int64         // ContentLength records the length of the associated content.
}

func (s *StreamReadCloser) Close() error {
	if s.Content != nil {
		return s.Content.Close()
	}
	return nil
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Url struct {
	url    string
	path   string
	values string
}

func NewUrl() *Url {
	return new(Url)
}

func (u *Url) Path(path string) *Url {
	if path == "" {
		u.path = ""
	} else {
		u.path = "/" + path
	}
	return u
}

func (u *Url) String() string {
	return fmt.Sprintf("%s%s%s", u.url, u.path, u.values)
}

func (u *Url) Url(url string) *Url {
	u.url = url
	return u
}

func (u *Url) Pathf(path string, a ...any) *Url {
	return u.Path(fmt.Sprintf(path, a...))
}

func (u *Url) Params(params map[string]string) *Url {
	if len(params) == 0 {
		u.values = ""
	} else {
		values := make(nUrl.Values)
		for k, v := range params {
			values.Add(k, v)
		}
		u.values = "?" + values.Encode()
	}
	return u
}

// helpers block

type File struct {
	Name string    // Name file name
	Data io.Reader // Data file data
}

type MultipartRequest struct {
	Body        *bytes.Buffer
	ContentType string
}

func CreateJsonReader[T any](ctx context.Context, metadata T) (io.Reader, error) {
	jStr, err := json.Marshal(metadata)
	if err != nil {
		return nil, ErrHttpRequestJsonMarshal(ctx, err)
	}
	return strings.NewReader(string(jStr)), nil
}

func CalculateLength(ctx context.Context, reader io.Reader) (io.Reader, int64, error) {
	if reader == nil {
		return reader, 0, nil
	}
	var buf bytes.Buffer
	length, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, 0, ErrHttpCalculateLengthCopy(ctx, err)
	}
	return &buf, length, nil
}

func CreateMultipartRequestOnlyFiles(ctx context.Context, files []*File) (rs *MultipartRequest, err error) {
	return CreateMultipartRequest(ctx, struct{}{}, files)
}

func CreateMultipartRequest[T comparable](ctx context.Context, metadata T, files []*File) (rs *MultipartRequest, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = jet.ErrPanic(ctx, r)
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if metadata != *new(T) {
		payload, err := mapRequestMetadata(ctx, metadata)
		if err != nil {
			return nil, err
		}
		for key, value := range payload {
			if err = writer.WriteField(key, value); err != nil {
				return nil, ErrHttpRequestWriterWriteField(ctx, err)
			}
		}
	}

	for _, file := range files {
		var dst io.Writer
		dst, err = writer.CreateFormFile("files", filepath.Base(file.Name))
		if err != nil {
			return nil, ErrHttpRequestWriterCreate(ctx, err)
		}
		if _, err = io.Copy(dst, file.Data); err != nil {
			return nil, ErrHttpRequestWriterCopy(ctx, err)
		}
	}
	if err = writer.Close(); err != nil {
		return nil, ErrHttpRequestWriterClose(ctx, err)
	}
	return &MultipartRequest{
		Body:        body,
		ContentType: writer.FormDataContentType(),
	}, nil
}

func mapRequestMetadata[T any](ctx context.Context, metadata T) (map[string]string, error) {
	jStr, err := json.Marshal(metadata)
	if err != nil {
		return nil, ErrHttpRequestJsonMarshal(ctx, err)
	}
	m := map[string]string{}
	if err = json.Unmarshal(jStr, &m); err != nil {
		return nil, ErrHttpRequestJsonUnmarshal(ctx, err)
	}
	return m, nil
}

func (r *Request) metrics(ctx context.Context, start time.Time, req *http.Request, resp *http.Response) {
	if r.metricsProvider != nil {
		elapsed := jet.Now().Sub(start)
		r.metricsProvider.RequestLatencySet(ctx, &RequestLatency{
			Url:             req.URL.String(),
			IntegrationName: r.name,
			LatencyMs:       elapsed.Milliseconds(),
		})

		if resp.StatusCode >= http.StatusMultipleChoices {
			r.metricsProvider.RequestErrorInc(ctx, &RequestError{
				Url:             req.URL.String(),
				IntegrationName: r.name,
				ErrorCode:       resp.StatusCode,
			})
		}
	}
}

func (r *Request) log(ctx context.Context, start time.Time, req *http.Request, resp *http.Response, data []byte) {
	if r.logger == nil {
		return
	}
	elapsed := jet.Now().Sub(start)
	l := r.logger.C(ctx).Mth("http-call").F(jet.KV{
		"method":  req.Method,
		"url":     req.URL.String(),
		"elapsed": elapsed.Seconds(),
		"status":  resp.Status,
		"size":    resp.ContentLength,
	})

	if r.extendedLog && len(data) != 0 {
		l.Trc(string(data))
	} else {
		l.Dbg()
	}
}
