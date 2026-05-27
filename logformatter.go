package jet

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sort"
	"sync"
)

// textHandler is a slog.Handler that renders records in the toolkit's compact
// bracketed format:
//
//	<time> [<LEVEL>][<fixed values>][<key:value>...] <message>
//
// Fixed fields (see fixedFields) are printed first, in order, as their value only.
// Remaining fields follow, sorted alphabetically, as key:value.
type textHandler struct {
	mu    *sync.Mutex
	out   io.Writer
	level slog.Leveler
	attrs []slog.Attr
}

func newTextHandler(out io.Writer, level slog.Leveler) *textHandler {
	return &textHandler{mu: &sync.Mutex{}, out: out, level: level}
}

func (h *textHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	nh.attrs = append(nh.attrs, h.attrs...)
	nh.attrs = append(nh.attrs, attrs...)
	return &nh
}

// WithGroup is a no-op: this flat format does not support attribute groups.
func (h *textHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	b := &bytes.Buffer{}

	b.WriteString(r.Time.Format(timestampFormat))
	b.WriteString(" [")
	b.WriteString(levelLabel(r.Level))
	b.WriteString("]")

	// collect fields preserving first-seen order
	values := make(map[string]string)
	order := make([]string, 0, len(h.attrs)+r.NumAttrs())
	add := func(a slog.Attr) {
		if _, exists := values[a.Key]; !exists {
			order = append(order, a.Key)
		}
		values[a.Key] = a.Value.Resolve().String()
	}
	for _, a := range h.attrs {
		add(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		add(a)
		return true
	})

	// fixed fields first, value only
	used := make(map[string]bool, len(fixedFields))
	for _, f := range fixedFields {
		if v, ok := values[f]; ok {
			b.WriteString("[")
			b.WriteString(v)
			b.WriteString("]")
			used[f] = true
		}
	}

	// remaining fields, sorted, as key:value
	rest := make([]string, 0, len(order))
	for _, k := range order {
		if !used[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		b.WriteString("[")
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(values[k])
		b.WriteString("]")
	}

	b.WriteString(" ")
	b.WriteString(r.Message)
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(b.Bytes())
	return err
}
