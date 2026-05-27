package http

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gotest.tools/v3/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequest simplifies testing of controller methods
// it provides fluent API to prepare and make mock request
type TestRequest struct {
	method           string
	url              string
	vars             map[string]string
	rqBodyReader     io.Reader
	rsBody           interface{}
	ts               *testing.T
	assertCode       int
	ctx              context.Context
	assertAppErrCode string
	headers          map[string]string
}

// NewTestRequest creates a new test HTTP request
func NewTestRequest(ts *testing.T, ctx context.Context) *TestRequest {
	return &TestRequest{
		vars:       map[string]string{},
		ts:         ts,
		assertCode: http.StatusOK,
		ctx:        ctx,
		headers:    map[string]string{},
	}
}

func (t *TestRequest) POST() *TestRequest {
	t.method = "POST"
	return t
}

func (t *TestRequest) GET() *TestRequest {
	t.method = "GET"
	return t
}

func (t *TestRequest) PUT() *TestRequest {
	t.method = "PUT"
	return t
}

func (t *TestRequest) DELETE() *TestRequest {
	t.method = "DELETE"
	return t
}

func (t *TestRequest) Url(url string) *TestRequest {
	t.url = url
	return t
}

// Var allows define URL parameters
func (t *TestRequest) Var(key, value string) *TestRequest {
	t.vars[key] = value
	return t
}

// Header allows define Header parameter
func (t *TestRequest) Header(key, value string) *TestRequest {
	t.headers[key] = value
	return t
}

// RqBody allows defining request body if any
func (t *TestRequest) RqBody(rq interface{}) *TestRequest {
	if rq != nil {
		rqJ, _ := json.Marshal(rq)
		t.rqBodyReader = strings.NewReader(string(rqJ))
	}
	return t
}

// RsBody allows pass a variable of expected response type, so that response body is unmarshalled to this variable
func (t *TestRequest) RsBody(rs interface{}) *TestRequest {
	if rs != nil {
		t.rsBody = rs
	}
	return t
}

// AssertOk asserts HTTP code is OK
func (t *TestRequest) AssertOk() *TestRequest {
	t.assertCode = http.StatusOK
	return t
}

// AssertCode asserts given http code
func (t *TestRequest) AssertCode(code int) *TestRequest {
	t.assertCode = code
	return t
}

// AssertAppError assert application error with given code
func (t *TestRequest) AssertAppError(code string) *TestRequest {
	t.assertAppErrCode = code
	return t
}

// Make makes a request with all specified params
// It returns
// 1: http code
// 2: response body
func (t *TestRequest) Make(controllerFn func(w http.ResponseWriter, r *http.Request)) {

	if t.method == "" {
		t.ts.Fatal(fmt.Errorf("method empty"))
	}
	if t.url == "" {
		t.ts.Fatal(fmt.Errorf("url empty"))
	}
	if t.ctx == nil {
		t.ts.Fatal(fmt.Errorf("context empty"))
	}

	r, err := http.NewRequest(t.method, t.url, t.rqBodyReader)
	if err != nil {
		t.ts.Fatal(err)
	}
	r = r.WithContext(t.ctx)

	for k, v := range t.headers {
		r.Header.Set(k, v)
	}

	if len(t.vars) > 0 {
		r = mux.SetURLVars(r, t.vars)
	}
	w := httptest.NewRecorder()

	controllerFn(w, r)

	if t.rsBody != nil && t.assertAppErrCode == "" {
		err = json.Unmarshal(w.Body.Bytes(), &t.rsBody)
		if err != nil {
			t.ts.Fatal(err)
		}
	}

	assert.Equal(t.ts, t.assertCode, w.Code)

	if t.assertAppErrCode != "" {
		appErr := &Error{}
		err = json.Unmarshal(w.Body.Bytes(), appErr)
		if err != nil {
			t.ts.Fatal(err)
		}
		assert.Equal(t.ts, t.assertAppErrCode, appErr.Code)
	}
}
