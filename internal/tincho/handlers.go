package tincho

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handlers struct {
	service *Service
	logger  *slog.Logger
}

func NewHandlers(logger *slog.Logger, service *Service) Handlers {
	return Handlers{service: service, logger: logger.With("component", "tincho-handlers")}
}

type RoomConfig struct {
	Password    string      `json:"password"`
	MaxPlayers  int         `json:"max_players"`
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
		h.logger.Warn(fmt.Sprintf("Error decoding room config: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error decoding room config"))
		return
	}
	deck := buildDeck(roomConfig.DeckOptions)
	roomID, err := h.service.NewRoom(h.logger, deck, roomConfig.MaxPlayers, roomConfig.Password)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error creating room: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error: %s", err)))
	} else {
		h.logger.Info(fmt.Sprintf("New room created: %s", roomID))
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(roomID))
	}
}

type RoomInfo struct {
	ID      string `json:"id"`
	Players int    `json:"players"`
}

func (h *Handlers) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms := make([]RoomInfo, 0, h.service.ActiveRoomCount())
	for _, room := range h.service.rooms {
		rooms = append(rooms, RoomInfo{
			ID:      room.ID,
			Players: len(room.state.GetPlayers()),
		})
	}
	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		h.logger.Warn(fmt.Sprintf("Error encoding rooms: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) JoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := strings.ToUpper(r.URL.Query().Get("room"))
	playerID := PlayerID(r.URL.Query().Get("player"))
	password := r.URL.Query().Get("password")
	if playerID == "" || roomID == "" {
		h.logger.Warn("Missing attributes")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing attributes"))
		return
	}
	room, exists := h.service.GetRoom(roomID)
	if !exists {
		h.logger.Warn("Error getting room index")
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
		h.connect(w, r, playerID, room, password)
	} else if curPlayer.SessionToken == sessionToken {
		// TODO: Check password for reconnection
		h.reconnect(w, r, &curPlayer, room)
	} else {
		h.logger.Warn(fmt.Sprintf("Player %s already exists in room %s", playerID, roomID))
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("player already exists in room"))
	}
}

func (h *Handlers) connect(w http.ResponseWriter, r *http.Request, playerID PlayerID, room *Room, password string) {
	player := NewPlayer(playerID)
	sesCookie := &http.Cookie{
		Name:    "session_token",
		Value:   player.SessionToken,
		Expires: time.Now().Add(1 * time.Hour),
	}
	ws, err := upgradeConnection(w, r, sesCookie)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error upgrading connection: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	wslogger := h.logger.With("room_id", room.ID, "player_id", player.ID)
	stopWS := handleWS(ws, player, room, wslogger)
	if err := h.service.JoinRoom(room.ID, player, password); err != nil {
		stopWS()
		h.logger.Warn(fmt.Sprintf("Error joining room: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error joining room"))
		return
	}
	h.logger.Info(fmt.Sprintf("Player %s joined room %s", playerID, room.ID))
}

func (h *Handlers) reconnect(w http.ResponseWriter, r *http.Request, player *Player, room *Room) {
	ws, err := upgradeConnection(w, r, nil)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error upgrading connection: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	// TODO: Send state to player
	wslogger := h.logger.With("room_id", room.ID, "player_id", player.ID)
	handleWS(ws, player, room, wslogger)
	h.logger.Info("Player %s reconnected to room %s", player.ID, room.ID)
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

func handleWS(ws *websocket.Conn, player *Player, room *Room, logger *slog.Logger) func() {
	ctx, cancelWSContext := context.WithCancel(room.Context)
	stopWS := func() {
		cancelWSContext()
		ws.Close()
	}
	go func() {
		logger.Info(fmt.Sprintf("Started socket write loop for player %s", player.ID))
		tick := time.NewTicker(10 * time.Second)
		for {
			select {
			case update := <-player.Updates:
				logger.Info(
					fmt.Sprintf("Sending update to player %s", player.ID),
					"update", update,
				)
				if err := ws.WriteJSON(update); err != nil {
					logger.Error("error sending update to player %s: %s", player.ID, err)
					stopWS()
					return
				}
			case <-tick.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.Error("error sending ping to player %s: %s", player.ID, err)
					stopWS()
					return
				}
			case <-ctx.Done():
				logger.Info(fmt.Sprintf("Stopping socket write loop for player %s", player.ID))
				return
			}
		}
	}()
	go func() {
		logger.Info(fmt.Sprintf("Started socket read loop for player %s", player.ID))
		tick := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-tick.C:
				_, message, err := ws.ReadMessage()
				if err != nil {
					logger.Error(fmt.Sprintf("Error reading message from player %s: %s", player.ID, err), "err", err)
					stopWS()
					return
				}
				var action Action
				if err := json.Unmarshal(message, &action); err != nil {
					logger.Error("error unmarshalling action", "err", err)
					return // TODO: Prevent disconnect
				}
				player.QueueAction(action)
			case <-ctx.Done():
				logger.Info(fmt.Sprintf("Stopping socket read for player %s", player.ID))
				return
			}
		}
	}()
	return stopWS
}
