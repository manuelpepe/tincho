package tincho

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handlers struct {
	game *Game
}

func NewHandlers(game *Game) Handlers {
	return Handlers{game: game}
}

func (h *Handlers) NewRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := h.game.NewRoom()
	if err != nil {
		log.Printf("Error creating room: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error: %s", err)))
	} else {
		log.Printf("New room created: %s", roomID)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(roomID))
	}
}

type RoomInfo struct {
	ID      string `json:"id"`
	Players int    `json:"players"`
}

func (h *Handlers) ListRooms(w http.ResponseWriter, r *http.Request) {
	log.Printf("Listing rooms")
	rooms := make([]RoomInfo, 0, len(h.game.rooms))
	for _, room := range h.game.rooms {
		rooms = append(rooms, RoomInfo{
			ID:      room.ID,
			Players: len(room.state.GetPlayers()),
		})
	}
	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		log.Printf("Error encoding rooms: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) JoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	playerID := r.URL.Query().Get("player")
	if playerID == "" || roomID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing attributes"))
		return
	}
	player, err := upgradeToPlayer(w, r, playerID)
	if err != nil {
		log.Printf("Error upgrading connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	if err := h.game.JoinRoom(roomID, player); err != nil {
		log.Printf("Error joining room: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error joining room"))
		return
	}
	log.Printf("Player %s joined room %s", playerID, roomID)
}

func upgradeToPlayer(w http.ResponseWriter, r *http.Request, playerID string) (Player, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return Player{}, fmt.Errorf("error upgrading connection: %w", err)
	}
	return NewPlayer(playerID, ws), nil
}
