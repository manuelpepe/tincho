package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/front"
	"github.com/manuelpepe/tincho/internal/tincho"
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
	handlers := tincho.NewHandlers(&service)
	bots := bots.NewHandlers(&service)
	r.HandleFunc("/new", handlers.NewRoom)
	r.HandleFunc("/list", handlers.ListRooms)
	r.HandleFunc("/join", handlers.JoinRoom)
	r.HandleFunc("/add-bot", bots.AddBot)
	r.Handle("/{file:.*}", frontHandler)

	log.Println("Listening on port 5555")
	log.Fatal(http.ListenAndServe(":5555", r))
}
