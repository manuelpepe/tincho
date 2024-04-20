package main

import (
	"context"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/manuelpepe/tincho/pkg/bots"
	"github.com/manuelpepe/tincho/pkg/front"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := tincho.NewService(ctx, tincho.ServiceConfig{
		MaxRooms:    10,
		RoomTimeout: 60 * time.Minute,
	})
	r := mux.NewRouter()
	frontHandler, err := front.FrontendHandler()
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handlers := tincho.NewHandlers(logger, &service)
	bots := bots.NewHandlers(logger, &service)
	r.HandleFunc("/new", handlers.NewRoom)
	r.HandleFunc("/list", handlers.ListRooms)
	r.HandleFunc("/join", handlers.JoinRoom)
	r.HandleFunc("/add-bot", bots.AddBot)
	r.Handle("/{file:.*}", frontHandler)

	log.Println("Listening on port 5555")
	log.Fatal(http.ListenAndServe(":5555", r))
}
