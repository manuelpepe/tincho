package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/manuelpepe/tincho/internal/tincho"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	game := tincho.NewGame()
	r := mux.NewRouter()
	handlers := tincho.NewHandlers(&game)
	r.HandleFunc("/new", handlers.NewRoom)
	r.HandleFunc("/list", handlers.ListRooms)
	r.HandleFunc("/join", handlers.JoinRoom)
	// r.HandleFunc("/echo", handlers.Echo)
	log.Println("Listening on port 5555")
	log.Fatal(http.ListenAndServe(":5555", r))
}
