package middleware

import (
	"net/http"
	"time"

	"go02-neighbor-help/internal/respond"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

type LogFunc func(format string, args ...any)

func Logger(logf LogFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newStatusRecorder(w)
			next.ServeHTTP(rec, r)
			if logf != nil {
				logf("%s %s %d %dB %s", r.Method, r.URL.RequestURI(), rec.status, rec.bytes, time.Since(start))
			}
		})
	}
}

func Recover() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					respond.Error(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Chain(fns ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(fns) - 1; i >= 0; i-- {
			if fns[i] == nil {
				continue
			}
			h = fns[i](h)
		}
		return h
	}
}
