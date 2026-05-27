package http

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/monitoring"
	"io"
	"net/http"
	nUrl "net/url"
	"strings"
	"testing"
)

type requestTestSuite struct {
	jet.Suite
}

func (s *requestTestSuite) SetupSuite() {
	s.Suite.Init(logf)
}

func TestRequestSuite(t *testing.T) {
	suite.Run(t, new(requestTestSuite))
}

func (s *requestTestSuite) Test_CalculateLength() {
	reader := strings.NewReader("data")
	r, l, err := CalculateLength(s.Ctx, reader)
	s.NoError(err)
	s.Equal(int64(4), l)
	s.NotEmpty(r)

	// can't read data from reader
	data := make([]byte, l)
	_, err = reader.Read(data)
	s.Equal(io.EOF, err)
	// can read data from returned reader (r)
	_, err = r.Read(data)
	s.NoError(err)
	_, err = r.Read(data)
	s.Equal(io.EOF, err)
	s.Equal("data", string(data))
}

func (s *requestTestSuite) Test_Url() {
	tests := []struct {
		url    string
		path   string
		params map[string]string
		res    string
	}{
		{},
		{
			url: "http://test/v1",
			res: "http://test/v1",
		},
		{
			url: "http://test/v1/",
			res: "http://test/v1/",
		},
		{
			url:  "http://test",
			path: "v1/doSmt",
			res:  "http://test/v1/doSmt",
		},
		{
			url:    "http://test",
			path:   "v1/doSmt.php",
			params: map[string]string{"1": "param1", "2": "param2"},
			res:    "http://test/v1/doSmt.php?1=param1&2=param2",
		},
		{
			url:    "http://test",
			params: map[string]string{"1": "pa   ram1", "2": "param2"},
			res:    "http://test?1=pa+++ram1&2=param2",
		},
	}

	for _, test := range tests {
		s.T().Run(test.url, func(t *testing.T) {
			url := NewUrl().Url(test.url).Path(test.path).Params(test.params).String()
			s.Equal(test.res, url)
		})
	}
}

func (s *requestTestSuite) Test_NetUrl_Ok() {
	cl := &testClient{statusCode: http.StatusOK, body: io.NopCloser(strings.NewReader("data"))}

	err := NewRq().
		Cl(cl).
		Url(&nUrl.URL{Host: "host", Path: "path", Scheme: "http"}).
		Do(s.Ctx)
	s.NoError(err)
}

func (s *requestTestSuite) Test_Stream_Ok() {
	cl := &testClient{statusCode: http.StatusOK, len: 4, body: io.NopCloser(strings.NewReader("data"))}

	rs, err := NewRq().
		Cl(cl).
		Url(NewUrl().Url("http:/test")).
		Stream(s.Ctx)
	s.NoError(err)
	defer rs.Close()
	s.Equal(int64(4), rs.ContentLength)

	data, err := io.ReadAll(rs.Content)
	s.NoError(err)

	res := string(data)
	s.Equal("data", res)
}

func (s *requestTestSuite) Test_Stream_WithMetrics_Ok() {
	cl := &testClient{statusCode: http.StatusOK, len: 4, body: io.NopCloser(strings.NewReader("data"))}

	metrics := NewRequestMetrics(&monitoring.Config{Enabled: false})

	rs, err := NewRq().
		Metrics(metrics).
		Cl(cl).
		Url(NewUrl().Url("http:/test")).
		Stream(s.Ctx)
	s.NoError(err)
	defer rs.Close()
	s.Equal(int64(4), rs.ContentLength)

	data, err := io.ReadAll(rs.Content)
	s.NoError(err)

	res := string(data)
	s.Equal("data", res)
}

func (s *requestTestSuite) Test_CustomFn_Ok() {
	cl := &testClient{statusCode: http.StatusOK, body: io.NopCloser(strings.NewReader("data"))}

	var res string
	err := NewRq().
		Cl(cl).
		Url(NewUrl().Url("http:/test")).
		Fn(func(data []byte) error {
			res = string(data)
			return nil
		}).
		Do(s.Ctx)
	s.NoError(err)
	s.Equal("data", res)
}

func (s *requestTestSuite) Test_ErrIfInvalidStatusCode() {
	cl := &testClient{statusCode: http.StatusBadRequest, body: io.NopCloser(strings.NewReader("data"))}

	err := NewRq().
		Cl(cl).
		Url(NewUrl().Url("http:/test")).
		Do(s.Ctx)
	s.AssertAppErr(err, ErrCodeHttpRequestInvalidStatusCode)
}

func (s *requestTestSuite) Test_ErrIfInvalidStatusCode_CustomErr() {
	cl := &testClient{statusCode: http.StatusBadRequest, body: io.NopCloser(strings.NewReader("data"))}

	err := NewRq().
		Cl(cl).
		Url(NewUrl().Url("http:/test")).
		ErrCodeFn(func(ctx context.Context, code int, data []byte) error {
			return ErrHttpRequestInvalidUrl(ctx, errors.New("text"))
		}).
		Do(s.Ctx)
	s.AssertAppErr(err, ErrCodeHttpRequestInvalidUrl)
}

func (s *requestTestSuite) Test_EmptyFileData_RecoverPanic() {
	rq, err := CreateMultipartRequest(s.Ctx, &File{Name: "name"}, []*File{{Name: "name"}})
	s.Error(err)
	s.Empty(rq)
}

type testClient struct {
	statusCode int
	body       io.ReadCloser
	len        int64
}

func (c *testClient) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode:    c.statusCode,
		Body:          c.body,
		ContentLength: c.len,
	}, nil
}
