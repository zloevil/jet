package http

import (
	"bytes"
	"context"
	"github.com/zloevil/jet"
	"io"
	"net/http"
)

func (s *Server) createTraceMiddleware(traceDetails *TraceDetails) func(next http.Handler) http.Handler {

	// set default if not passed
	if traceDetails == nil {
		traceDetails = &TraceDetails{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			// logging request before execution
			if err := s.traceRequest(ctx, r, traceDetails); err != nil {
				s.logger().C(ctx).E(err).St().Err()
				next.ServeHTTP(w, r)
				return
			}

			lrw := &traceResponseWriter{ResponseWriter: w, traceDetails: traceDetails}

			next.ServeHTTP(lrw, r)

			s.traceResponse(ctx, lrw, traceDetails)

		})
	}
}

func (s *Server) traceRequest(ctx context.Context, r *http.Request, traceDetails *TraceDetails) error {

	kv := jet.KV{"verb": r.Method, "url": r.URL.Path, "headers": r.Header}

	if traceDetails.RequestBody {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		kv["body"] = string(body)
	}

	s.logger().C(ctx).F(kv).Trc("rq")

	return nil
}

func (s *Server) traceResponse(ctx context.Context, w *traceResponseWriter, traceDetails *TraceDetails) {

	if !traceDetails.Response {
		return
	}

	kv := jet.KV{"status": w.StatusCode, "headers": w.Header()}

	if traceDetails.ResponseBody {
		kv["body"] = string(w.Body)
	}

	s.logger().C(ctx).F(kv).Trc("rs")

}

type traceResponseWriter struct {
	http.ResponseWriter
	Body         []byte
	StatusCode   int
	traceDetails *TraceDetails
}

func (rw *traceResponseWriter) Write(data []byte) (int, error) {
	if rw.traceDetails.Response {
		rw.Body = append(rw.Body, data...)
	}
	return rw.ResponseWriter.Write(data)
}

func (rw *traceResponseWriter) WriteHeader(code int) {

	if rw.traceDetails.Response {
		rw.StatusCode = code
	}

	rw.ResponseWriter.WriteHeader(code)
}
