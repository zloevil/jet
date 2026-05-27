package jet

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-viper/mapstructure/v2"
	"golang.org/x/text/language"
	"google.golang.org/grpc/metadata"
)

const (
	AppTest = "test"
)

type requestContextKey struct{}

type RequestContext struct {
	Rid   string       `json:"_ctx.rid,omitempty" mapstructure:"_ctx.rid"`   // Rid request ID
	Sid   string       `json:"_ctx.sid,omitempty" mapstructure:"_ctx.sid"`   // Sid session ID
	Uid   string       `json:"_ctx.uid,omitempty" mapstructure:"_ctx.uid"`   // Uid user ID
	Un    string       `json:"_ctx.un,omitempty" mapstructure:"_ctx.un"`     // Un username
	App   string       `json:"_ctx.app,omitempty" mapstructure:"_ctx.app"`   // App application
	ClIp  string       `json:"_ctx.clIp,omitempty" mapstructure:"_ctx.clIp"` // ClIp client IP
	Roles []string     `json:"_ctx.rl,omitempty" mapstructure:"_ctx.rl"`     // Roles list of roles
	Lang  language.Tag `json:"_ctx.lang,omitempty" mapstructure:"_ctx.lang"` // Lang client language
	Kv    KV           `json:"_ctx.kv,omitempty" mapstructure:"_ctx.kv"`     // Kv arbitrary key-value
}

func NewRequestCtx() *RequestContext {
	return &RequestContext{}
}

func (r *RequestContext) GetRequestId() string {
	return r.Rid
}

func (r *RequestContext) GetSessionId() string {
	return r.Sid
}

func (r *RequestContext) GetUserId() string {
	return r.Uid
}

func (r *RequestContext) GetRoles() []string {
	return r.Roles
}

func (r *RequestContext) GetUsername() string {
	return r.Un
}

func (r *RequestContext) GetApp() string {
	return r.App
}

func (r *RequestContext) GetClientIp() string {
	return r.ClIp
}

func (r *RequestContext) GetLang() language.Tag {
	return r.Lang
}

func (r *RequestContext) GetKv() KV {
	return r.Kv
}

func (r *RequestContext) Empty() *RequestContext {
	return &RequestContext{}
}

func (r *RequestContext) WithRequestId(requestId string) *RequestContext {
	r.Rid = requestId
	return r
}

func (r *RequestContext) WithNewRequestId() *RequestContext {
	r.Rid = NewId()
	return r
}

func (r *RequestContext) WithSessionId(sessionId string) *RequestContext {
	r.Sid = sessionId
	return r
}

func (r *RequestContext) WithApp(app string) *RequestContext {
	r.App = app
	return r
}

func (r *RequestContext) WithClientIp(ip string) *RequestContext {
	r.ClIp = ip
	return r
}

func (r *RequestContext) WithLang(lang language.Tag) *RequestContext {
	r.Lang = lang
	return r
}

func (r *RequestContext) WithKv(key string, val interface{}) *RequestContext {
	if r.Kv == nil {
		r.Kv = KV{}
	}
	r.Kv[key] = val
	return r
}

func (r *RequestContext) TestApp() *RequestContext {
	return r.WithApp(AppTest)
}

func (r *RequestContext) EN() *RequestContext {
	return r.WithLang(language.English)
}

func (r *RequestContext) WithUser(userId, username string) *RequestContext {
	r.Uid = userId
	r.Un = username
	return r
}

func (r *RequestContext) WithRoles(roles ...string) *RequestContext {
	r.Roles = roles
	return r
}

func (r *RequestContext) ToContext(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, requestContextKey{}, r)
}

func (r *RequestContext) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"_ctx.rid":  r.Rid,
		"_ctx.sid":  r.Sid,
		"_ctx.uid":  r.Uid,
		"_ctx.un":   r.Un,
		"_ctx.app":  r.App,
		"_ctx.clIp": r.ClIp,
		"_ctx.rl":   r.Roles,
		"_ctx.lang": r.Lang,
		"_ctx.kv":   r.Kv,
	}
}

func Request(context context.Context) (*RequestContext, bool) {
	if r, ok := context.Value(requestContextKey{}).(*RequestContext); ok {
		return r, true
	}
	return &RequestContext{}, false
}

func MustRequest(context context.Context) (*RequestContext, error) {
	if r, ok := context.Value(requestContextKey{}).(*RequestContext); ok {
		return r, nil
	}
	return &RequestContext{}, errors.New("context is invalid")
}

func ContextToGrpcMD(ctx context.Context) (metadata.MD, bool) {
	if r, ok := Request(ctx); ok {
		rm, _ := json.Marshal(*r)
		return metadata.Pairs("rq-bin", string(rm)), true
	}
	return metadata.Pairs(), false
}

func FromGrpcMD(ctx context.Context, md metadata.MD) context.Context {
	if rqb, ok := md["rq-bin"]; ok {
		if len(rqb) > 0 {
			rm := []byte(rqb[0])
			rq := &RequestContext{}
			_ = json.Unmarshal(rm, rq)
			return context.WithValue(ctx, requestContextKey{}, rq)
		}
	}
	return ctx
}

func FromMap(ctx context.Context, mp map[string]interface{}) (context.Context, error) {
	var r *RequestContext
	err := mapstructure.Decode(mp, &r)
	if err != nil {
		return nil, err
	}
	return r.ToContext(ctx), nil
}

func Copy(ctx context.Context) context.Context {
	if r, ok := Request(ctx); ok {
		ct, err := FromMap(context.TODO(), r.ToMap())
		if err != nil {
			return ctx
		}
		return ct
	}
	return ctx
}
