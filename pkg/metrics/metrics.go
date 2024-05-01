package metrics

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/manuelpepe/tincho/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "tincho_http_duration_seconds",
			Help: "Duration of HTTP requests.",
		},
		[]string{"path"},
	)

	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tincho_http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		},
		[]string{"method", "code"},
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

	gamesStarted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tincho_games_started",
			Help: "Tracks the number of games started.",
		},
	)

	gamesEnded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tincho_games_ended",
			Help: "Tracks the number of games ended.",
		},
	)

	connectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tincho_connections_total",
			Help: "Tracks the number of connections.",
		},
		[]string{"reconnection"},
	)
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(requestDuration.WithLabelValues(path))

		next.ServeHTTP(w, r)

		reslen, err := strconv.Atoi(w.Header().Get("Content-Length"))
		if err != nil {
			reslen = -1
		}

		rec, ok := w.(*middleware.StatusRecorder)
		if !ok {
			return // TODO: Log error
		}

		timer.ObserveDuration()
		requestsTotal.WithLabelValues(r.Method, strconv.Itoa(rec.Status)).Inc()
		requestSize.WithLabelValues(r.Method, strconv.Itoa(rec.Status)).Observe(float64(r.ContentLength))
		responseSize.WithLabelValues(r.Method, strconv.Itoa(rec.Status)).Observe(float64(reslen))
	})
}

func IncGamesTotal() {
	gamesStarted.Inc()
}

func IncGamesEnded() {
	gamesEnded.Inc()
}

func IncConnectionsTotal(reconnection bool) {
	connectionsTotal.WithLabelValues(strconv.FormatBool(reconnection)).Inc()
}
