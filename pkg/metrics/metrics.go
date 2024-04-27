package metrics

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "tincho_http_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})

	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tincho_http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code"},
	)

	requestSize = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "tincho_http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		[]string{"method", "code"},
	)
	responseSize = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "tincho_http_response_size_bytes",
			Help: "Tracks the size of HTTP responses.",
		},
		[]string{"method", "code"},
	)
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(requestDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
		requestsTotal.WithLabelValues(r.Method, http.StatusText(http.StatusOK)).Inc()
		requestSize.WithLabelValues(r.Method, http.StatusText(http.StatusOK)).Observe(float64(r.ContentLength))
		reslen, err := strconv.Atoi(w.Header().Get("Content-Length"))
		if err != nil {
			reslen = -1
		}
		responseSize.WithLabelValues(r.Method, http.StatusText(http.StatusOK)).Observe(float64(reslen))
	})
}
