package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/metrics"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handlers struct {
	service *Service
	logger  *slog.Logger
}

func NewHandlers(logger *slog.Logger, service *Service) *Handlers {
	return &Handlers{service: service, logger: logger.With("component", "tincho-handlers")}
}

type RoomConfig struct {
	Password    string      `json:"password"`
	MaxPlayers  int         `json:"max_players"`
	DeckOptions DeckOptions `json:"deck"`
}

func (rc RoomConfig) Validate() error {
	if rc.MaxPlayers <= 1 {
		return errors.New("max players should be greater than 1")
	}

	playerLimit := 10
	if rc.MaxPlayers > playerLimit {
		return fmt.Errorf("max players should be less than %d", playerLimit)
	}
	return nil
}

type DeckOptions struct {
	Extended bool `json:"extended"`
	Chaos    bool `json:"chaos"`
}

func buildDeck(options DeckOptions) game.Deck {
	deck := game.NewDeck()
	if options.Extended {
		deck = game.AddExtendedVariation(deck)
	}
	if options.Chaos {
		deck = game.AddChaosVariation(deck)
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
	if err := roomConfig.Validate(); err != nil {
		h.logger.Warn(fmt.Sprintf("Error validating room config: %s", err), "err", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error validating room config"))
		return
	}
	deck := buildDeck(roomConfig.DeckOptions)
	roomID, err := h.service.NewRoom(h.logger, deck, roomConfig.MaxPlayers, roomConfig.Password)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error creating room: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error: %s", err)))
	}
	h.logger.Info(fmt.Sprintf("New room created: %s", roomID))
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(roomID))
	metrics.IncGamesTotal()
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
	playerID := game.PlayerID(r.URL.Query().Get("player"))
	password := r.URL.Query().Get("password")
	if playerID == "" || roomID == "" {
		remove_cookie(r, w)
		h.logger.Warn("Missing attributes")
		return
	}

	room, exists := h.service.GetRoom(roomID)
	if !exists {
		remove_cookie(r, w)
		h.logger.Warn("Error getting room index")
		return
	} else if room.Context.Err() != nil {
		remove_cookie(r, w)
		h.logger.Warn("Room has been closed")
		return
	}

	tokenPlayerID, _, sessionToken, err := decode_cookie(r, w)
	if err == nil {
		playerID = tokenPlayerID
	} else if errors.Is(err, ErrInvalidCookie) {
		h.logger.Warn("Invalid token")
		return
	}

	curPlayer, exists := room.GetConnection(playerID)
	if !exists {
		h.connect(w, r, playerID, room, password)
	} else if curPlayer.SessionToken == sessionToken {
		h.reconnect(w, r, curPlayer, room)
	} else {
		h.logger.Warn(fmt.Sprintf("Player %s already exists in room %s", playerID, roomID))
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("player already exists in room"))
	}
}

func (h *Handlers) connect(w http.ResponseWriter, r *http.Request, playerID game.PlayerID, room *Room, password string) {
	connection := NewConnection(playerID)
	sesCookie := encode_cookie(connection.ID, room.ID, connection.SessionToken)

	ws, err := upgradeConnection(w, r, sesCookie)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error upgrading connection: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	wslogger := h.logger.With("room_id", room.ID, "player_id", connection.ID)
	stopWS := handleWS(ws, connection, room, wslogger)
	if err := h.service.JoinRoom(room.ID, connection, password); err != nil {
		stopWS()
		h.logger.Warn(fmt.Sprintf("Error joining room: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError) // FIXME: headers already sent
		w.Write([]byte("error joining room"))
		return
	}
	h.logger.Info(fmt.Sprintf("Player %s joined room %s", playerID, room.ID))
	metrics.IncConnectionsTotal(false)
}

func (h *Handlers) reconnect(w http.ResponseWriter, r *http.Request, conn *Connection, room *Room) {
	ws, err := upgradeConnection(w, r, nil)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error upgrading connection: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error upgrading connection"))
		return
	}
	wslogger := h.logger.With("room_id", room.ID, "player_id", conn.Player.ID)
	stopWS := handleWS(ws, conn, room, wslogger)
	if err := h.service.JoinRoomWithoutPassword(room.ID, conn); err != nil {
		stopWS()
		h.logger.Warn(fmt.Sprintf("Error joining room: %s", err), "err", err)
		w.WriteHeader(http.StatusInternalServerError) // FIXME: headers already sent
		w.Write([]byte("error joining room"))
		return
	}
	h.logger.Info(fmt.Sprintf("Player %s reconnected to room %s", conn.Player.ID, room.ID))
	metrics.IncConnectionsTotal(true)
}

const TOKEN_COOKIE_NAME = "session_token"
const TOKEN_SEPARATOR = "::"

var ErrInvalidCookie = fmt.Errorf("invalid token")

func decode_cookie(r *http.Request, w http.ResponseWriter) (game.PlayerID, string, string, error) {
	cookie, err := r.Cookie(TOKEN_COOKIE_NAME)
	if err != nil {
		return "", "", "", err
	}

	if cookie == nil {
		return "", "", "", http.ErrNoCookie
	}

	parts := strings.Split(cookie.Value, TOKEN_SEPARATOR)
	if len(parts) != 3 {
		if err := remove_cookie(r, w); err != nil {
			return "", "", "", fmt.Errorf("error upgrading connection on decode: %w", err)
		}
		return "", "", "", ErrInvalidCookie
	}
	return game.PlayerID(parts[0]), parts[1], parts[2], nil
}

func encode_cookie(player game.PlayerID, room string, token string) *http.Cookie {
	encoded := fmt.Sprintf("%s%s%s%s%s", player, TOKEN_SEPARATOR, room, TOKEN_SEPARATOR, token)
	return &http.Cookie{
		Name:    TOKEN_COOKIE_NAME,
		Value:   encoded,
		Expires: time.Now().Add(24 * time.Hour),
	}
}

func remove_cookie(r *http.Request, w http.ResponseWriter) error {
	c := &http.Cookie{
		Name:    TOKEN_COOKIE_NAME,
		Value:   "",
		Expires: time.Now().Add(-24 * time.Hour),
	}
	ws, err := upgradeConnection(w, r, c)
	defer ws.Close()
	if err != nil {
		return fmt.Errorf("error upgrading connection to remove token: %w", err)
	}
	return nil
}

func upgradeConnection(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) (*websocket.Conn, error) {
	var header http.Header
	if cookie != nil {
		header = http.Header{"Set-Cookie": []string{cookie.String()}}
	}
	ws, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		return nil, fmt.Errorf("error upgrading connection: %w", err)
	}
	return ws, nil
}

func handleWS(ws *websocket.Conn, conn *Connection, room *Room, logger *slog.Logger) func() {
	ctx, cancelWSContext := context.WithCancel(room.Context)
	stopWS := func() {
		cancelWSContext()
		ws.Close()
	}
	player := conn.Player

	go func() {
		logger.Info(fmt.Sprintf("Started socket write loop for player %s", player.ID))
		tick := time.NewTicker(10 * time.Second)
		for {
			select {
			case update := <-conn.Updates:
				metrics.IncWebsocketOutgoing()
				logger.Info(
					fmt.Sprintf("Sending update to player %s", player.ID),
					"update", update,
				)
				if err := ws.WriteJSON(update); err != nil {
					logger.Error(fmt.Sprintf("error sending update to player %s: %s", player.ID, err))
					stopWS()
					return
				}
			case <-tick.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.Error(fmt.Sprintf("error sending ping to player %s: %s", player.ID, err))
					stopWS()
					return
				}
			case <-ctx.Done():
				logger.Info(fmt.Sprintf("Stopping socket write loop for player %s", player.ID))
				for i := 0; i < len(conn.Updates); i++ {
					update := <-conn.Updates
					logger.Info(fmt.Sprintf("Sending last buffered messages for player %s", player.ID), "update", update)
					if err := ws.WriteJSON(update); err != nil {
						logger.Error(fmt.Sprintf("error sending update to player %s: %s", player.ID, err))
						stopWS()
						return
					}
				}
				logger.Info(fmt.Sprintf("Stopped socket write loop for player %s", player.ID))
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
					logger.Info(fmt.Sprintf("Stopping socket read loop for player %s", player.ID))
					stopWS()
					return
				}
				metrics.IncWebsocketIncoming()
				metrics.ObserveWebsocketIncomingSize(float64(len(message)))
				action, err := NewActionFromRawMessage(message)
				if err != nil {
					logger.Error(fmt.Sprintf("Error unmarshalling action from player %s: %s", player.ID, err), "err", err)
					continue
				}
				conn.QueueAction(action)
			case <-ctx.Done():
				logger.Info(fmt.Sprintf("Stopping socket read loop for player %s", player.ID))
				return
			}
		}
	}()

	return stopWS
}

func unmarshallTypeFromMessage(message []byte) string {
	var actionType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &actionType); err != nil {
		return ""
	}
	return actionType.Type
}
