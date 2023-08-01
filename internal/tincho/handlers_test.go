package tincho

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func NewServer() (*Game, *httptest.Server) {
	game := NewGame()
	r := mux.NewRouter()
	handlers := NewHandlers(&game)
	r.HandleFunc("/join", handlers.JoinRoom)
	return &game, httptest.NewServer(r)
}

func NewSocket(server *httptest.Server, user string, room string) *websocket.Conn {
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/join?room=" + room + "&player=" + user
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		panic(err)
	}
	return ws
}

func TestHandlers_PlayersJoinRoom(t *testing.T) {
	g, s := NewServer()
	roomID := g.NewRoom()
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer s.Close()
	defer ws1.Close()
	defer ws2.Close()
	assert.Equal(t, 2, len(g.rooms[0].Players))
}

func TestHandlers_GameStarts(t *testing.T) {
	g, s := NewServer()
	roomID := g.NewRoom()
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer s.Close()
	defer ws1.Close()
	defer ws2.Close()

	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))

	var update Update
	_, message, err := ws1.ReadMessage()
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(message, &update))
	assert.Equal(t, UpdateTypeStart, update.Type)

	_, message, err = ws2.ReadMessage()
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(message, &update))
	assert.Equal(t, UpdateTypeStart, update.Type)
}
