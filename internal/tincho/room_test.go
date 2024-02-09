package tincho

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func NewServer() (*Game, *httptest.Server, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	game := NewGame(ctx, GameConfig{MaxRooms: 3, RoomTimeout: 5 * time.Minute})
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

func TestRoomLimit(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	for i := 0; i < 3; i++ {
		_, err := g.NewRoom()
		assert.NoError(t, err)
	}
	_, err := g.NewRoom()
	assert.ErrorIs(t, err, ErrRoomsLimitReached)
	for _, room := range g.rooms {
		if room != nil {
			room.Close()
		}
	}
	time.Sleep(1 * time.Second) // wait for rooms to close
	_, err = g.NewRoom()
	assert.NoError(t, err)
}

func TestPlayersJoinRoom(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	roomID, err := g.NewRoom()
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()
	time.Sleep(1 * time.Second) // wait for connection
	roomix, exists := g.GetRoomIndex(roomID)
	assert.True(t, exists)
	assert.Equal(t, 2, g.rooms[roomix].PlayerCount())
}

func TestDoubleDiscard(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	deck := Deck{
		{Suit: SuitClubs, Value: 1},
		{Suit: SuitDiamonds, Value: 1},
		{Suit: SuitClubs, Value: 2},
		{Suit: SuitClubs, Value: 3},
		{Suit: SuitClubs, Value: 4},
		{Suit: SuitClubs, Value: 5},
		{Suit: SuitClubs, Value: 6},
		{Suit: SuitClubs, Value: 7},
		{Suit: SuitClubs, Value: 8},
		{Suit: SuitClubs, Value: 9},
		{Suit: SuitClubs, Value: 10},
	}
	roomID, err := g.NewRoomWithDeck(deck)
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()

	assertRecieved(t, ws1, UpdateTypePlayersChanged)
	assertRecieved(t, ws1, UpdateTypePlayersChanged)
	assertRecieved(t, ws2, UpdateTypePlayersChanged)

	// p1 starts game
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))

	// both players prompted to peek
	u1 := assertRecieved(t, ws1, UpdateTypeGameStart)
	u2 := assertRecieved(t, ws2, UpdateTypeGameStart)
	assertDataMatches(t, u1, UpdateGameStart{Players: []Player{{ID: "p1", PendingFirstPeek: true}, {ID: "p2", PendingFirstPeek: true}}})
	assertDataMatches(t, u2, UpdateGameStart{Players: []Player{{ID: "p1", PendingFirstPeek: true}, {ID: "p2", PendingFirstPeek: true}}})

	// p1 peeks
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionFirstPeek}))
	u1 = assertRecieved(t, ws1, UpdateTypePlayerPeeked)
	u2 = assertRecieved(t, ws2, UpdateTypePlayerPeeked)
	assertDataMatches(t, u1, UpdatePlayerPeekedData{Player: "p1", Cards: deck[:2]})
	assertDataMatches(t, u2, UpdatePlayerPeekedData{Player: "p1", Cards: nil})

	// p2 peeks
	assert.NoError(t, ws2.WriteJSON(Action{Type: ActionFirstPeek}))
	u1 = assertRecieved(t, ws1, UpdateTypePlayerPeeked)
	u2 = assertRecieved(t, ws2, UpdateTypePlayerPeeked)
	assertDataMatches(t, u1, UpdatePlayerPeekedData{Player: "p2", Cards: nil})
	assertDataMatches(t, u2, UpdatePlayerPeekedData{Player: "p2", Cards: deck[4:6]})

	// both recieve game start
	u1 = assertRecieved(t, ws1, UpdateTypeTurn)
	u2 = assertRecieved(t, ws2, UpdateTypeTurn)
	assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
	assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})

	// p1 draws
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: DrawSourcePile, Effect: CardEffectPeekCartaAjena, Card: deck[8]})
	assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: DrawSourcePile})

	// p1 discards two equal cards
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: 1, CardPosition2: toIntPointer(0)}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeDiscard)
	assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1, 0}, Cards: []Card{deck[1], deck[0]}})
	assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1, 0}, Cards: []Card{deck[1], deck[0]}})

	// turn changes
	u1 = assertRecieved(t, ws1, UpdateTypeTurn)
	u2 = assertRecieved(t, ws2, UpdateTypeTurn)
	assertDataMatches(t, u1, UpdateTurnData{Player: "p2"})
	assertDataMatches(t, u2, UpdateTurnData{Player: "p2"})

	// p2 draws
	assert.NoError(t, ws2.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Player: "p2", Source: DrawSourcePile})
	assertDataMatches(t, u2, UpdateDrawData{Player: "p2", Source: DrawSourcePile, Effect: CardEffectSwapCards, Card: deck[9]})

	// p2 discards two non-equal cards and fails
	assert.NoError(t, ws2.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: 1, CardPosition2: toIntPointer(0)}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeFailedDoubleDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeFailedDoubleDiscard)
	assertDataMatches(t, u1, UpdateTypeFailedDoubleDiscardData{Player: "p2", CardsPositions: []int{1, 0}, Cards: []Card{deck[5], deck[4]}})
	assertDataMatches(t, u2, UpdateTypeFailedDoubleDiscardData{Player: "p2", CardsPositions: []int{1, 0}, Cards: []Card{deck[5], deck[4]}})
}

func TestBasicGame(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	deck := NewDeck()
	roomID, err := g.NewRoomWithDeck(deck)
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()

	assertRecieved(t, ws1, UpdateTypePlayersChanged)
	assertRecieved(t, ws1, UpdateTypePlayersChanged)
	assertRecieved(t, ws2, UpdateTypePlayersChanged)

	// p1 starts game
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))

	// both players prompted to peek
	u1 := assertRecieved(t, ws1, UpdateTypeGameStart)
	u2 := assertRecieved(t, ws2, UpdateTypeGameStart)
	assertDataMatches(t, u1, UpdateGameStart{Players: []Player{{ID: "p1", PendingFirstPeek: true}, {ID: "p2", PendingFirstPeek: true}}})
	assertDataMatches(t, u2, UpdateGameStart{Players: []Player{{ID: "p1", PendingFirstPeek: true}, {ID: "p2", PendingFirstPeek: true}}})

	// p1 peeks
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionFirstPeek}))
	u1 = assertRecieved(t, ws1, UpdateTypePlayerPeeked)
	u2 = assertRecieved(t, ws2, UpdateTypePlayerPeeked)
	assertDataMatches(t, u1, UpdatePlayerPeekedData{Player: "p1", Cards: deck[:2]})
	assertDataMatches(t, u2, UpdatePlayerPeekedData{Player: "p1", Cards: nil})

	// p2 peeks
	assert.NoError(t, ws2.WriteJSON(Action{Type: ActionFirstPeek}))
	u1 = assertRecieved(t, ws1, UpdateTypePlayerPeeked)
	u2 = assertRecieved(t, ws2, UpdateTypePlayerPeeked)
	assertDataMatches(t, u1, UpdatePlayerPeekedData{Player: "p2", Cards: nil})
	assertDataMatches(t, u2, UpdatePlayerPeekedData{Player: "p2", Cards: deck[4:6]})

	// both recieve game start
	u1 = assertRecieved(t, ws1, UpdateTypeTurn)
	u2 = assertRecieved(t, ws2, UpdateTypeTurn)
	assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
	assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})

	// p1 draws
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: DrawSourcePile, Effect: CardEffectSwapCards, Card: deck[8]})
	assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: DrawSourcePile})

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
	u1 = assertRecieved(t, ws1, UpdateTypeDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeDiscard)
	assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []Card{deck[1]}})
	assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []Card{deck[1]}})

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
	assertDataMatches(t, u1, UpdateDrawData{Player: "p2", Source: DrawSourcePile})
	assertDataMatches(t, u2, UpdateDrawData{Player: "p2", Source: DrawSourcePile, Effect: CardEffectNone, Card: deck[9]})

	// p2 discards drawn card card
	assert.NoError(t, ws2.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: -1}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeDiscard)
	assertDataMatches(t, u1, UpdateDiscardData{Player: "p2", CardsPositions: []int{-1}, Cards: []Card{deck[9]}})
	assertDataMatches(t, u2, UpdateDiscardData{Player: "p2", CardsPositions: []int{-1}, Cards: []Card{deck[9]}})

	// turn changes
	u1 = assertRecieved(t, ws1, UpdateTypeTurn)
	u2 = assertRecieved(t, ws2, UpdateTypeTurn)
	assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
	assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})

	// p1 draws
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDraw,
		Data: safeMarshal(t, ActionDrawData{Source: DrawSourcePile}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDraw)
	u2 = assertRecieved(t, ws2, UpdateTypeDraw)
	assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: DrawSourcePile, Effect: CardEffectNone, Card: deck[10]})
	assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: DrawSourcePile})

	// p1 discards second card
	assert.NoError(t, ws1.WriteJSON(Action{
		Type: ActionDiscard,
		Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
	}))
	u1 = assertRecieved(t, ws1, UpdateTypeDiscard)
	u2 = assertRecieved(t, ws2, UpdateTypeDiscard)
	assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []Card{deck[8]}})
	assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []Card{deck[8]}})

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

func toIntPointer(i int) *int {
	return &i
}
