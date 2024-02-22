package tincho

import (
	"context"
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

type RoomConfig struct {
	DeckOptions DeckOptions `json:"deck"`
}

type DeckOptions struct {
	Extended bool `json:"extended"`
	Chaos    bool `json:"chaos"`
}

func buildDeck(options DeckOptions) Deck {
	deck := NewDeck()
	if options.Extended {
		deck = AddExtendedVariation(deck)
	}
	if options.Chaos {
		deck = AddChaosVariation(deck)
	}
	deck.Shuffle()
	return deck
}

func (h *Handlers) NewRoom(w http.ResponseWriter, r *http.Request) {
	var roomConfig RoomConfig
	if err := json.NewDecoder(r.Body).Decode(&roomConfig); err != nil {
		log.Printf("Error decoding room config: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error decoding room config"))
		return
	}
	deck := buildDeck(roomConfig.DeckOptions)
	roomID, err := h.game.NewRoom(deck)
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
	room, exists := h.game.GetRoom(roomID)
	if !exists {
		log.Printf("Error getting room index")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error getting room index"))
		return
	}
	sessionToken := ""
	if token, err := r.Cookie("session_token"); err != nil {
		if token != nil {
			sessionToken = token.Value
		}
	}
	curPlayer, exists := room.GetPlayer(playerID)
	if !exists {
		h.connect(w, r, playerID, room)
	} else if curPlayer.SessionToken == sessionToken {
		h.reconnect(w, r, &curPlayer, room)
	} else {
		log.Printf("Player %s already exists in room %s", playerID, roomID)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("player already exists in room"))
	}
}

func (h *Handlers) connect(w http.ResponseWriter, r *http.Request, playerID string, room *Room) {
	player := NewPlayer(playerID)
	sesCookie := &http.Cookie{
		Name:    "session_token",
		Value:   player.SessionToken,
		Expires: time.Now().Add(1 * time.Hour),
	}
	ws, err := upgradeConnection(w, r, sesCookie)
	if err != nil {
		log.Printf("Error upgrading connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	if err := h.game.JoinRoom(room.ID, player); err != nil {
		log.Printf("Error joining room: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error joining room"))
		return
	}
	go handleWS(ws, player, room)
	log.Printf("Player %s joined room %s", playerID, room.ID)
}

func (h *Handlers) reconnect(w http.ResponseWriter, r *http.Request, player *Player, room *Room) {
	ws, err := upgradeConnection(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	// TODO: Send state to player
	go handleWS(ws, player, room)
	log.Printf("Player %s reconnected to room %s", player.ID, room.ID)
}

func upgradeConnection(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) (*websocket.Conn, error) {
	var header http.Header
	if cookie != nil {
		header = http.Header{"Cookie": []string{cookie.String()}}
	}
	ws, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		return nil, fmt.Errorf("error upgrading connection: %w", err)
	}
	return ws, nil
}

func handleWS(ws *websocket.Conn, player *Player, room *Room) {
	tick := time.NewTicker(1 * time.Second) // TODO: Make global
	ctx, cancelWSContext := context.WithCancel(room.Context)
	stopWS := func() {
		cancelWSContext()
		ws.Close()
	}
	go func() {
		for {
			select {
			case update := <-player.Updates:
				log.Printf("Sending update to player %s: {Type:%s, Data:\"%s\"}", player.ID, update.Type, update.Data)
				if err := ws.WriteJSON(update); err != nil {
					log.Printf("error sending update to player %s: %s", player.ID, err)
					stopWS()
					return
				}
			case <-ctx.Done():
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
					stopWS()
					return
				}
				var action Action
				if err := json.Unmarshal(message, &action); err != nil {
					log.Println(err)
					return // TODO: Prevent disconnect
				}
				player.QueueAction(action)
			case <-ctx.Done():
				log.Printf("Stopping socket read for player %s", player.ID)
				return
			}
		}
	}()
}
