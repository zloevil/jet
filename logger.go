package jet

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
)

const (
	// PanicLevel level. Nothing below a panic is logged; reserved as the highest threshold.
	PanicLevel = "panic"
	// FatalLevel level. Logs and then calls os.Exit(1).
	FatalLevel = "fatal"
	// ErrorLevel level. Used for errors that should definitely be noted.
	ErrorLevel = "error"
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel = "warning"
	// InfoLevel level. General operational entries about what's going on inside the application.
	InfoLevel = "info"
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel = "debug"
	// TraceLevel level. Finer-grained informational events than Debug.
	TraceLevel = "trace"

	FormatterText = "plain"
	FormatterJson = "json"
)

// Custom slog levels. slog ships only Debug/Info/Warn/Error, so Trace/Fatal/Panic
// are defined relative to them.
const (
	levelTrace = slog.Level(-8)
	levelDebug = slog.LevelDebug // -4
	levelInfo  = slog.LevelInfo  // 0
	levelWarn  = slog.LevelWarn  // 4
	levelError = slog.LevelError // 8
	levelFatal = slog.Level(12)
	levelPanic = slog.Level(16)
)

// timestampFormat is shared by the text and JSON handlers.
const timestampFormat = "2006-01-02T15:04:05.000-0700"

// fixedFields are rendered first, in this order, without their keys (value only).
var fixedFields = []string{"call.svc", "call.pr", "call.cmp", "call.mth", "call.node"}

// parseLevel maps a string level from config to a slog.Level.
func parseLevel(level string) (slog.Level, error) {
	switch level {
	case TraceLevel:
		return levelTrace, nil
	case DebugLevel:
		return levelDebug, nil
	case InfoLevel, "":
		return levelInfo, nil
	case WarnLevel:
		return levelWarn, nil
	case ErrorLevel:
		return levelError, nil
	case FatalLevel:
		return levelFatal, nil
	case PanicLevel:
		return levelPanic, nil
	default:
		return levelInfo, fmt.Errorf("unknown log level: %q", level)
	}
}

// levelLabel returns the display name for a level, including the custom ones.
func levelLabel(l slog.Level) string {
	switch {
	case l < levelDebug:
		return "TRACE"
	case l < levelInfo:
		return "DEBUG"
	case l < levelWarn:
		return "INFO"
	case l < levelError:
		return "WARN"
	case l < levelFatal:
		return "ERROR"
	case l < levelPanic:
		return "FATAL"
	default:
		return "PANIC"
	}
}

// ErrorHook allows specifying a hook invoked for every logged error.
type ErrorHook interface {
	Error(err error)
}

// LogConfig represents logging configuration.
type LogConfig struct {
	Level   string // Level logging level
	Format  string // Format (plain, json)
	Context bool   // Context if true, request context params are part of logging
	Service bool   // Service if true, service params are part of logging
}

// Logger holds the configured slog logger and is the factory source for CLogger.
type Logger struct {
	Cfg  *LogConfig
	lvl  *slog.LevelVar
	sl   *slog.Logger
	hook ErrorHook
}

// InitLogger creates and configures a Logger from cfg.
func InitLogger(cfg *LogConfig) *Logger {
	l := &Logger{lvl: new(slog.LevelVar)}
	l.Init(cfg)
	return l
}

// Init (re)configures the logger from cfg.
func (l *Logger) Init(cfg *LogConfig) {
	l.Cfg = cfg
	if l.lvl == nil {
		l.lvl = new(slog.LevelVar)
	}
	lv, err := parseLevel(cfg.Level)
	if err != nil {
		panic(err)
	}
	l.lvl.Set(lv)

	var h slog.Handler
	if cfg.Format == FormatterJson {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       l.lvl,
			ReplaceAttr: jsonReplaceAttr,
		})
	} else {
		h = newTextHandler(os.Stdout, l.lvl)
	}
	l.sl = slog.New(h)
}

// SetErrorHook registers a hook invoked for every error logged at Error level or above.
func (l *Logger) SetErrorHook(h ErrorHook) {
	l.hook = h
}

// SetLevel changes the active logging level at runtime.
func (l *Logger) SetLevel(level string) {
	lv, err := parseLevel(level)
	if err != nil {
		panic(err)
	}
	l.Cfg.Level = level
	l.lvl.Set(lv)
}

// jsonReplaceAttr normalizes the time format and renders custom level labels for the JSON handler.
func jsonReplaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			a.Value = slog.StringValue(t.Format(timestampFormat))
		}
	case slog.LevelKey:
		if lv, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(levelLabel(lv))
		}
	}
	return a
}

type CLoggerFunc func() CLogger

// CLogger provides structured logging abilities.
// !!!! Not thread safe. Don't share one CLogger instance through multiple goroutines.
type CLogger interface {
	// C - adds request context to log
	//
	// don't put context when logging error, as it makes sense a context of where error happens rather than a context of where error log is invoked
	// otherwise, context will be logged twice
	C(ctx context.Context) CLogger
	// F - adds fields to log
	F(fields KV) CLogger
	// E - adds error to log
	E(err error) CLogger
	// St - adds stack to log (if err is already set)
	St() CLogger
	// Cmp - adds component
	Cmp(c string) CLogger
	// Mth - adds method
	Mth(m string) CLogger
	// Pr - adds protocol
	Pr(m string) CLogger
	// Srv - adds unique service code
	Srv(s string) CLogger
	// Node - adds service instance code
	Node(n string) CLogger
	Inf(args ...interface{}) CLogger
	InfF(format string, args ...interface{}) CLogger
	Err(args ...interface{}) CLogger
	ErrF(format string, args ...interface{}) CLogger
	Dbg(args ...interface{}) CLogger
	DbgF(format string, args ...interface{}) CLogger
	Trc(args ...interface{}) CLogger
	TrcF(format string, args ...interface{}) CLogger
	// TrcObj marshals all args only if loglevel = Trace, otherwise bypass
	// Note that only Exported fields of objects are logged (due to nature of json.Marshal)
	TrcObj(format string, args ...interface{}) CLogger
	Warn(args ...interface{}) CLogger
	WarnF(format string, args ...interface{}) CLogger
	Fatal(args ...interface{}) CLogger
	FatalF(format string, args ...interface{}) CLogger
	Clone() CLogger
	Printf(string, ...interface{})
	PrintfErr(f string, args ...interface{})
	// Write writer implementation
	Write(p []byte) (n int, err error)
}

// L creates a CLogger bound to logger.
func L(logger *Logger) CLogger {
	return &clogger{logger: logger}
}

type clogger struct {
	logger *Logger
	attrs  []slog.Attr
	err    error
}

// Clone always use Clone when passing CLogger between goroutines.
func (cl *clogger) Clone() CLogger {
	attrs := make([]slog.Attr, len(cl.attrs))
	copy(attrs, cl.attrs)
	return &clogger{logger: cl.logger, attrs: attrs, err: cl.err}
}

func (cl *clogger) C(ctx context.Context) CLogger {
	if !cl.logger.Cfg.Context {
		return cl
	}
	if r, ok := Request(ctx); ok && r != nil {
		if rid := r.GetRequestId(); rid != "" {
			cl.attrs = append(cl.attrs, slog.String("ctx.rid", rid))
		}
		if un := r.GetUsername(); un != "" {
			cl.attrs = append(cl.attrs, slog.String("ctx.un", un))
		}
		if sid := r.GetSessionId(); sid != "" {
			cl.attrs = append(cl.attrs, slog.String("ctx.sid", sid))
		}
	}
	return cl
}

func (cl *clogger) F(fields KV) CLogger {
	for k, v := range fields {
		cl.attrs = append(cl.attrs, slog.Any(k, v))
	}
	return cl
}

func (cl *clogger) E(err error) CLogger {
	// if err is AppErr, log error code / type and its fields separately
	if appErr, ok := IsAppErr(err); ok {
		cl.attrs = append(cl.attrs,
			slog.String("err-code", appErr.Code()),
			slog.Any("error-type", appErr.Type()),
		)
		for k, v := range appErr.Fields() {
			cl.attrs = append(cl.attrs, slog.Any(k, v))
		}
	}
	cl.err = err
	cl.attrs = append(cl.attrs, slog.String("error", err.Error()))
	return cl
}

func (cl *clogger) St() CLogger {
	if cl.err != nil {
		// if err is AppErr take stack from error itself, otherwise build stack right here
		if appErr, ok := IsAppErr(cl.err); ok {
			cl.attrs = append(cl.attrs, slog.String("err-stack", appErr.WithStack()))
		} else {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			cl.attrs = append(cl.attrs, slog.String("err-stack", string(buf[:n])))
		}
	}
	return cl
}

func (cl *clogger) Srv(s string) CLogger {
	if !cl.logger.Cfg.Service {
		return cl
	}
	cl.attrs = append(cl.attrs, slog.String("call.svc", s))
	return cl
}

func (cl *clogger) Node(n string) CLogger {
	if !cl.logger.Cfg.Service {
		return cl
	}
	cl.attrs = append(cl.attrs, slog.String("call.node", n))
	return cl
}

func (cl *clogger) Cmp(c string) CLogger {
	cl.attrs = append(cl.attrs, slog.String("call.cmp", c))
	return cl
}

func (cl *clogger) Pr(p string) CLogger {
	cl.attrs = append(cl.attrs, slog.String("call.pr", p))
	return cl
}

func (cl *clogger) Mth(m string) CLogger {
	cl.attrs = append(cl.attrs, slog.String("call.mth", m))
	return cl
}

// log emits the accumulated record at the given level and applies error-hook /
// fatal semantics.
func (cl *clogger) log(level slog.Level, msg string) {
	cl.logger.sl.LogAttrs(context.Background(), level, msg, cl.attrs...)

	if level >= levelError && cl.err != nil && cl.logger.hook != nil {
		cl.logger.hook.Error(cl.err)
	}
	if level >= levelFatal {
		os.Exit(1)
	}
}

func (cl *clogger) Err(args ...interface{}) CLogger {
	cl.log(levelError, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) ErrF(format string, args ...interface{}) CLogger {
	cl.log(levelError, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) Inf(args ...interface{}) CLogger {
	cl.log(levelInfo, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) InfF(format string, args ...interface{}) CLogger {
	cl.log(levelInfo, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) Warn(args ...interface{}) CLogger {
	cl.log(levelWarn, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) WarnF(format string, args ...interface{}) CLogger {
	cl.log(levelWarn, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) Dbg(args ...interface{}) CLogger {
	cl.log(levelDebug, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) DbgF(format string, args ...interface{}) CLogger {
	cl.log(levelDebug, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) Trc(args ...interface{}) CLogger {
	cl.log(levelTrace, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) TrcF(format string, args ...interface{}) CLogger {
	cl.log(levelTrace, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) TrcObj(format string, args ...interface{}) CLogger {
	if cl.logger.Cfg.Level == TraceLevel {
		argsJs := make([]interface{}, 0, len(args))
		for _, a := range args {
			if a != nil {
				js, _ := json.Marshal(a)
				argsJs = append(argsJs, string(js))
			}
		}
		return cl.TrcF(format, argsJs...)
	}
	return cl
}

func (cl *clogger) Fatal(args ...interface{}) CLogger {
	cl.log(levelFatal, fmt.Sprint(args...))
	return cl
}

func (cl *clogger) FatalF(format string, args ...interface{}) CLogger {
	cl.log(levelFatal, fmt.Sprintf(format, args...))
	return cl
}

func (cl *clogger) Printf(f string, args ...interface{}) {
	cl.DbgF(f, args...)
}

func (cl *clogger) PrintfErr(f string, args ...interface{}) {
	cl.ErrF(f, args...)
}

func (cl *clogger) Write(p []byte) (n int, err error) {
	cl.Trc(string(p))
	return len(p), nil
}
