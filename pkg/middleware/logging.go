package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// StatusRecorder is a wrapper for an http.ResponseWriter that tracks the status code written to the response.
type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

// WriteHeader implements the http.ResponseWriter interface.
func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// Hijack implements the http.Hijacker interface.
func (r *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not implement Hijacker")
	}
	return hj.Hijack()
}

func LogRequestMiddleweare(logger *slog.Logger) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			recorder := &StatusRecorder{
				ResponseWriter: w,
				Status:         200,
			}

			h.ServeHTTP(recorder, r)

			elapsed := time.Since(now)

			logger.Info(
				fmt.Sprintf(
					"[%s] %s %s - %d %s - %dns",
					now.Format("2006-01-02 15:04:05"),
					r.Method,
					r.URL.Path,
					recorder.Status,
					http.StatusText(recorder.Status),
					elapsed.Nanoseconds(),
				),
				slog.String("method", r.Method),
				slog.String("url", r.URL.Path),
				slog.Int("status_code", recorder.Status),
				slog.String("status_name", http.StatusText(recorder.Status)),
				slog.Duration("duration_ns", elapsed),
				slog.String("component", "logging-middleware"),
				slog.String("remote_addr", r.RemoteAddr),
			)

		})
	}
}
