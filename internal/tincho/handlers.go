package tincho

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	ws, player, err := upgradeToPlayer(w, r, playerID)
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
	room, ok := h.game.GetRoom(roomID)
	if !ok {
		log.Printf("Error getting room index")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error getting room index"))
		return
	}
	go handleWebsocket(ws, player, room)
	log.Printf("Player %s joined room %s", playerID, roomID)
}

func upgradeToPlayer(w http.ResponseWriter, r *http.Request, playerID string) (*websocket.Conn, *Player, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error upgrading connection: %w", err)
	}
	return ws, NewPlayer(playerID), nil
}

func handleWebsocket(ws *websocket.Conn, player *Player, room *Room) {
	tick := time.NewTicker(1 * time.Second) // TODO: Make global
	go func() {
		for {
			select {
			case update := <-player.Updates:
				log.Printf("Sending update to player %s: {Type:%s, Data:\"%s\"}", player.ID, update.Type, update.Data)
				if err := ws.WriteJSON(update); err != nil {
					log.Println(err)
					return
				}
			case <-room.Context.Done():
				log.Printf("Stopping socket write loop for player %s", player.ID)
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-tick.C:
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Printf("Error reading message from player %s: %s", player.ID, err)
					return
				}
				var action Action
				if err := json.Unmarshal(message, &action); err != nil {
					log.Println(err)
					return // TODO: Prevent disconnect
				}
				player.QueueAction(action)
			case <-room.Context.Done():
				log.Printf("Stopping socket read for player %s", player.ID)
				return
			}
		}
	}()
}
