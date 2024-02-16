package bots

import (
	"log"
	"net/http"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type Handlers struct {
	game *tincho.Game
}

func NewHandlers(game *tincho.Game) Handlers {
	return Handlers{game: game}
}

func (h *Handlers) AddBot(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing room attribute"))
		return
	}
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing difficulty attribute"))
		return
	}
	room, ok := h.game.GetRoom(roomID)
	if !ok {
		log.Printf("Error getting room index")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error getting room index"))
		return
	}
	player := tincho.NewPlayer(RandomBotName())
	bot, err := NewBot(room.Context, player, difficulty)
	if err != nil {
		log.Printf("Error creating bot: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error creating bot"))
		return
	}
	if err := h.game.JoinRoom(roomID, player); err != nil {
		log.Printf("Error joining room: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error joining room"))
		return
	}
	go func() {
		if err := bot.Start(); err != nil {
			log.Printf("Error with bot: %s", err)
		}
		// TODO: If bot fails, broadcasts are stuck because noone is reading from the updates channel.
		// probably should tear down room and remove players.
	}()
	log.Printf("Bot %s joined room %s", player.ID, roomID)
}
