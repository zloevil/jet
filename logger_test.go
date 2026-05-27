package jet

import (
	"context"
	"fmt"
	"testing"
)

func Test_Clogger_WithCtx(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	ctx := NewRequestCtx().WithNewRequestId().WithUser("1", "john").ToContext(context.Background())
	l := L(logger).C(ctx)
	l.Inf("I'm logger")
}

func Test_Clogger_WithComponentAndMethod(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).Cmp("service").Mth("do")
	l.Inf("I'm logger")
}

func Test_Clogger_WithComponentMethodAndCtx(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	ctx := NewRequestCtx().WithNewRequestId().WithUser("1", "john").ToContext(context.Background())
	l := L(logger).Cmp("service").Mth("do").C(ctx)
	l.Inf("I'm logger")
}

func Test_Clogger_All(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	ctx := NewRequestCtx().WithNewRequestId().WithUser("1", "john").ToContext(context.Background())
	l := L(logger).Cmp("service").Mth("do").C(ctx).F(KV{"field": "value"})
	l.Inf("I'm logger")
}

func Test_Clogger_WithFields(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).F(KV{"field": "value"})
	l.Inf("I'm logger")
}

func Test_Clogger_WithErr(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).E(fmt.Errorf("error"))
	l.Err("my bad")
}

func Test_Clogger_WithErrStack(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).E(fmt.Errorf("error")).St()
	l.Err("my bad")
}

func Test_Clogger_WithAppErr(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).E(NewAppError("ERR-123", "%s happened", "shit"))
	l.Err("my bad")
}

func Test_Clogger_WithAppErrAndStack(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	l := L(logger).E(NewAppError("ERR-123", "%s happened", "shit")).St()
	l.Err("my bad")
}

func Test_Clogger_WithTraceObj(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})

	obj1 := struct {
		A string
		B int
	}{
		A: "test",
		B: 5,
	}

	type n struct {
		A string
	}
	type s struct {
		Nested *n
	}

	obj2 := &s{
		Nested: &n{
			A: "str",
		},
	}

	L(logger).TrcObj("objects: %v, %v", obj1, obj2)
}

func Test_Clogger_WithAppErrAndFields(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	e := NewAppErrBuilder("ERR-123", "%s happened", "shit").F(KV{"f": "v"}).Err()
	l := L(logger).E(e)
	l.Err("my bad")
}

func Test_Clogger_WithAppErrAndAppContext(t *testing.T) {
	logger := InitLogger(&LogConfig{Level: TraceLevel})
	ctx := NewRequestCtx().WithRequestId("123").ToContext(context.Background())
	e := NewAppErrBuilder("ERR-123", "%s happened", "shit").C(ctx).Err()
	l := L(logger).E(e)
	l.Err("my bad")
}
