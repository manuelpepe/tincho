package tincho

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func NewServer() (*Game, *httptest.Server, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	game := NewGame(ctx)
	r := mux.NewRouter()
	handlers := NewHandlers(&game)
	r.HandleFunc("/join", handlers.JoinRoom)
	return &game, httptest.NewServer(r), cancel
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
	g, s, cancel := NewServer()
	defer cancel()
	roomID := g.NewRoom()
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer s.Close()
	defer ws1.Close()
	defer ws2.Close()
	assert.Equal(t, 2, len(g.rooms[0].Players))
}

func TestHandlers_BasicGame(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	roomID := g.NewRoom()
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer s.Close()
	defer ws1.Close()
	defer ws2.Close()

	// p1 starts game
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))
	u1 := assertRecieved(t, ws1, UpdateTypeStartRound)
	u2 := assertRecieved(t, ws2, UpdateTypeStartRound)
	assertDataMatches(t, u1, UpdateStartRoundData{Players: []Player{{ID: "p1"}, {ID: "p2"}}})
	assertDataMatches(t, u2, UpdateStartRoundData{Players: []Player{{ID: "p1"}, {ID: "p2"}}})

	// p1 draws
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Source: DrawSourcePile, Effect: CardEffectNone, Card: g.rooms[0].DrawPile[8]})
	assertDataMatches(t, u2, UpdateDrawData{Source: DrawSourcePile, Effect: CardEffectNone})

	// p1 tries to draw again and fails
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeError)
	assertDataMatches(t, u1, UpdateErrorData{Message: ErrPendingDiscard.Error()})

	// p1 discards second card
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
	}))
	discarded := g.rooms[0].Players[0].Hand[1]
	u1 = assertRecieved(t, ws1, UpdateTypeDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeDiscard)
	assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardPosition: 1, Card: discarded})
	assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardPosition: 1, Card: discarded})

	// turn changes
	u1 = assertRecieved(t, ws1, UpdateTypeTurn)
	u2 = assertRecieved(t, ws2, UpdateTypeTurn)
	assertDataMatches(t, u1, UpdateTurnData{Player: "p2"})
	assertDataMatches(t, u2, UpdateTurnData{Player: "p2"})

	// p1 tries to discard again and fails
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeError)
	assertDataMatches(t, u1, UpdateErrorData{Message: ErrNotYourTurn.Error()})

	// p1 tries to draw again and fails
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeError)
	assertDataMatches(t, u1, UpdateErrorData{Message: ErrNotYourTurn.Error()})

	// p2 draws
	assert.NoError(t, ws2.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Source: DrawSourcePile, Effect: CardEffectNone})
	assertDataMatches(t, u2, UpdateDrawData{Source: DrawSourcePile, Effect: CardEffectNone, Card: g.rooms[0].DrawPile[9]})

}

func safeMarshal(t *testing.T, v interface{}) []byte {
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return b
}

func assertRecieved(t *testing.T, ws *websocket.Conn, updateType UpdateType) Update {
	var update Update
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(message, &update))
	assert.Equal(t, updateType, update.Type)
	return update
}

func assertDataMatches[G any](t *testing.T, update Update, expected G) {
	var value G
	assert.NoError(t, json.Unmarshal(update.Data, &value))
	assert.Equal(t, expected, value)
}
