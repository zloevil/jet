package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/zloevil/jet"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// ProxyConfig is proxy http configuration
type ProxyConfig struct {
	Url string
}

// Proxy represents HTTP proxy
type Proxy struct {
	buf    bytes.Buffer
	writer *multipart.Writer
	cfg    *ProxyConfig
	err    error
}

func NewProxy(cfg *ProxyConfig) *Proxy {
	p := &Proxy{
		cfg: cfg,
		buf: bytes.Buffer{},
	}
	p.writer = multipart.NewWriter(&p.buf)
	return p
}

func (p *Proxy) NewRequest() *Proxy {
	p.buf = bytes.Buffer{}
	p.writer = multipart.NewWriter(&p.buf)
	p.err = nil
	return p
}

func (p *Proxy) Error() error {
	return p.err
}

func (p *Proxy) AddFile(name, fileName string, file io.Reader) *Proxy {
	if p.err != nil {
		return p
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, name, fileName))
	h.Set("Content-Type", "multipart/form-data")

	var fw io.Writer
	fw, err := p.writer.CreatePart(h)
	if err != nil {
		p.err = ErrHttpProxyFileCreatePart(err)
		return p
	}
	if _, err = io.Copy(fw, file); err != nil {
		p.err = ErrHttpProxyFileCopyFile(err)
		return p
	}
	return p
}

func (p *Proxy) AddField(key, value string) *Proxy {
	if p.err != nil || value == "" {
		return p
	}
	if err := p.writer.WriteField(key, value); err != nil {
		p.err = ErrHttpProxyFileWriteField(err)
		return p
	}
	return p
}

func (p *Proxy) AddMetadataField(key string, value map[string]string) *Proxy {
	if len(value) == 0 {
		return p
	}
	bts, _ := json.Marshal(value)
	return p.AddField(key, string(bts))
}

func (p *Proxy) PUT(ctx context.Context, path string, res any) error {
	return p.do(ctx, http.MethodPut, path, &res)
}

func (p *Proxy) POST(ctx context.Context, path string, res any) error {
	return p.do(ctx, http.MethodPost, path, &res)
}

func (p *Proxy) do(ctx context.Context, method, path string, res any) error {
	if p.err != nil {
		return p.err
	}

	p.writer.Close()
	req, err := http.NewRequest(method, p.cfg.Url+path, &p.buf)
	if err != nil {
		return ErrHttpProxyFileNewRequest(ctx, err)
	}
	// set up headers
	req.Header.Set("Content-Type", p.writer.FormDataContentType())
	rCtx, err := jet.MustRequest(ctx)
	if err != nil {
		return ErrHttpProxyFileInvalidContext(ctx, err)
	}
	req.Header.Add("RequestId", rCtx.GetRequestId())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ErrHttpProxyFileClientDo(ctx, err)
	}
	defer resp.Body.Close()

	return p.readResponse(ctx, resp, &res)
}

func (p *Proxy) readResponse(ctx context.Context, r *http.Response, res any) error {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ErrHttpProxyFileReadResponse(ctx, err)
	}
	// check app error
	httpErr := &Error{}
	if err = json.Unmarshal(data, &httpErr); err != nil {
		return ErrHttpProxyFileJsonUnmarshal(ctx, err)
	}
	if httpErr != nil && httpErr.Code != "" && httpErr.Message != "" {
		return jet.NewAppErrBuilder(httpErr.Code, httpErr.Message).Err()
	}
	// set response
	if err = json.Unmarshal(data, &res); err != nil {
		return ErrHttpProxyFileJsonUnmarshal(ctx, err)
	}
	return nil
}
