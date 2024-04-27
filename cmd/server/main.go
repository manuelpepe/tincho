package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/manuelpepe/tincho/pkg/bots"
	"github.com/manuelpepe/tincho/pkg/front"
	"github.com/manuelpepe/tincho/pkg/metrics"
	"github.com/manuelpepe/tincho/pkg/middleware"
	"github.com/manuelpepe/tincho/pkg/tincho"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With(slog.String("app", "tincho"))

	service := tincho.NewService(ctx, tincho.ServiceConfig{
		MaxRooms:    10,
		RoomTimeout: 60 * time.Minute,
	})

	frontHandler, err := front.FrontendHandler()
	if err != nil {
		log.Fatal(err)
	}
	handlers := tincho.NewHandlers(logger, &service)
	bots := bots.NewHandlers(logger, &service)

	r := mux.NewRouter()
	r.Use(middleware.LogRequestMiddleweare(logger))
	r.Use(metrics.MetricsMiddleware)

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/new", handlers.NewRoom)
	r.HandleFunc("/list", handlers.ListRooms)
	r.HandleFunc("/join", handlers.JoinRoom)
	r.HandleFunc("/add-bot", bots.AddBot)
	r.Handle("/{file:.*}", frontHandler)

	slog.Info("Listening on port 5555")
	log.Fatal(http.ListenAndServe(":5555", r))
}
