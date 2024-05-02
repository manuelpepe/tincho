package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

	cfg, err := parseEnv()
	if err != nil {
		log.Fatal(fmt.Errorf("error parsing env: %w", err))
	}

	service := tincho.NewService(ctx, cfg)
	handlers, err := newHandlers(logger, &service)
	if err != nil {
		log.Fatal(fmt.Errorf("error creating handlers: %w", err))
	}

	r := mux.NewRouter()
	r.Use(middleware.LogRequestMiddleweare(logger))
	r.Use(metrics.MetricsMiddleware)

	r.Handle("/metrics", handlers.prom)
	r.HandleFunc("/new", handlers.tincho.NewRoom)
	r.HandleFunc("/list", handlers.tincho.ListRooms)
	r.HandleFunc("/join", handlers.tincho.JoinRoom)
	r.HandleFunc("/add-bot", handlers.bots.AddBot)
	r.Handle("/{file:.*}", handlers.front)

	slog.Info("Listening on port 5555")
	log.Fatal(http.ListenAndServe(":5555", r))
}

type handlers struct {
	tincho *tincho.Handlers
	bots   *bots.Handlers
	front  http.Handler
	prom   http.Handler
}

func newHandlers(logger *slog.Logger, service *tincho.Service) (handlers, error) {
	frontHandler, err := front.FrontendHandler()
	if err != nil {
		return handlers{}, fmt.Errorf("error creating frontend handler: %w", err)
	}
	return handlers{
		tincho: tincho.NewHandlers(logger, service),
		bots:   bots.NewHandlers(logger, service),
		front:  frontHandler,
		prom:   promhttp.Handler(),
	}, nil
}

func parseEnv() (tincho.ServiceConfig, error) {
	maxRooms, err := strconv.Atoi(os.Getenv("TINCHO_MAX_ROOMS"))
	if err != nil {
		return tincho.ServiceConfig{}, fmt.Errorf("error parsing TINCHO_MAX_ROOMS: %w", err)
	}

	roomTimeout, err := strconv.Atoi(os.Getenv("TINCHO_ROOM_TIMEOUT"))
	if err != nil {
		return tincho.ServiceConfig{}, fmt.Errorf("error parsing TINCHO_ROOM_TIMEOUT: %w", err)
	}

	return tincho.ServiceConfig{
		MaxRooms:    maxRooms,
		RoomTimeout: time.Duration(roomTimeout) * time.Minute,
	}, nil

}
