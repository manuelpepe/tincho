package tincho

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/manuelpepe/tincho/internal/game"
	"github.com/stretchr/testify/assert"
)

func NewServer() (*Service, *httptest.Server, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	game := NewService(ctx, ServiceConfig{MaxRooms: 3, RoomTimeout: 5 * time.Minute})
	r := mux.NewRouter()
	handlers := NewHandlers(slog.Default(), &game)
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

func NewRoomBasic(g *Service) (string, error) {
	deck := game.NewDeck()
	deck.Shuffle()
	return g.NewRoom(slog.Default(), deck, 4, "")
}

func TestRoomLimit(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	for i := 0; i < 3; i++ {
		_, err := NewRoomBasic(g)
		assert.NoError(t, err)
	}
	_, err := NewRoomBasic(g)
	assert.ErrorIs(t, err, ErrRoomsLimitReached)
	for _, room := range g.rooms {
		if room != nil {
			room.closeRoom()
		}
	}
	time.Sleep(1 * time.Second) // wait for rooms to close
	_, err = NewRoomBasic(g)
	assert.NoError(t, err)
}

func TestPlayersJoinRoom(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	roomID, err := NewRoomBasic(g)
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()
	time.Sleep(1 * time.Second) // wait for connection
	room, exists := g.GetRoom(roomID)
	assert.True(t, exists)
	assert.Equal(t, 2, room.CurrentPlayers())
}

func TestDoubleDiscard(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	deck := game.Deck{
		{Suit: game.SuitClubs, Value: 1},    // p1
		{Suit: game.SuitDiamonds, Value: 1}, // p1
		{Suit: game.SuitClubs, Value: 2},    // p1
		{Suit: game.SuitClubs, Value: 3},    // p1
		{Suit: game.SuitClubs, Value: 4},    // p2
		{Suit: game.SuitClubs, Value: 5},    // p2
		{Suit: game.SuitClubs, Value: 6},    // p2
		{Suit: game.SuitClubs, Value: 7},    // p2
		{Suit: game.SuitClubs, Value: 8},    // discarded
		{Suit: game.SuitClubs, Value: 9},    // first draw
		{Suit: game.SuitClubs, Value: 10},   // second draw
	}
	roomID, err := g.NewRoom(slog.Default(), deck, 4, "")
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()

	assertRecieved[UpdatePlayersChangedData](t, ws1, UpdateTypePlayersChanged)
	assertRecieved[UpdatePlayersChangedData](t, ws1, UpdateTypePlayersChanged)
	assertRecieved[UpdatePlayersChangedData](t, ws2, UpdateTypePlayersChanged)

	// p1 starts game
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))

	{
		// both players recieve game config
		u1 := assertRecieved[UpdateGameConfig](t, ws1, UpdateTypeGameConfig)
		u2 := assertRecieved[UpdateGameConfig](t, ws2, UpdateTypeGameConfig)
		assertDataMatches(t, u1, UpdateGameConfig{CardsInDeck: 11})
		assertDataMatches(t, u2, UpdateGameConfig{CardsInDeck: 11})
	}

	{
		// both players prompted to peek
		u1 := assertRecieved[UpdateStartNextRoundData](t, ws1, UpdateTypeGameStart)
		u2 := assertRecieved[UpdateStartNextRoundData](t, ws2, UpdateTypeGameStart)
		expected := UpdateStartNextRoundData{
			Players: []*game.Player{
				{ID: "p1", PendingFirstPeek: true, Hand: make(game.Hand, 4)},
				{ID: "p2", PendingFirstPeek: true, Hand: make(game.Hand, 4)},
			},
			TopDiscard: deck[8],
		}
		assertDataMatches(t, u1, expected)
		assertDataMatches(t, u2, expected)
	}

	{
		// p1 peeks
		assert.NoError(t, ws1.WriteJSON(Action{Type: ActionFirstPeek}))
		u1 := assertRecieved[UpdatePlayerFirstPeekedData](t, ws1, UpdateTypePlayerFirstPeeked)
		u2 := assertRecieved[UpdatePlayerFirstPeekedData](t, ws2, UpdateTypePlayerFirstPeeked)
		assertDataMatches(t, u1, UpdatePlayerFirstPeekedData{Player: "p1", Cards: deck[:2]})
		assertDataMatches(t, u2, UpdatePlayerFirstPeekedData{Player: "p1", Cards: nil})

		// p2 peeks
		assert.NoError(t, ws2.WriteJSON(Action{Type: ActionFirstPeek}))
		u1 = assertRecieved[UpdatePlayerFirstPeekedData](t, ws1, UpdateTypePlayerFirstPeeked)
		u2 = assertRecieved[UpdatePlayerFirstPeekedData](t, ws2, UpdateTypePlayerFirstPeeked)
		assertDataMatches(t, u1, UpdatePlayerFirstPeekedData{Player: "p2", Cards: nil})
		assertDataMatches(t, u2, UpdatePlayerFirstPeekedData{Player: "p2", Cards: deck[4:6]})
	}

	{
		// both recieve game start
		u1 := assertRecieved[UpdateTurnData](t, ws1, UpdateTypeTurn)
		u2 := assertRecieved[UpdateTurnData](t, ws2, UpdateTypeTurn)
		assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
		assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})
	}

	{ // p1 draws
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateDrawData](t, ws1, UpdateTypeDraw)
		u2 := assertRecieved[UpdateDrawData](t, ws2, UpdateTypeDraw)
		assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile, Effect: game.CardEffectSwapCards, Card: deck[9]})
		assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile})
	}
	{ // p1 discards two equal cards
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: 1, CardPosition2: toIntPointer(0)}),
		}))
		u1 := assertRecieved[UpdateDiscardData](t, ws1, UpdateTypeDiscard)
		u2 := assertRecieved[UpdateDiscardData](t, ws2, UpdateTypeDiscard)
		assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1, 0}, Cards: []game.Card{deck[1], deck[0]}})
		assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1, 0}, Cards: []game.Card{deck[1], deck[0]}})
	}
	{
		// turn changes
		u1 := assertRecieved[UpdateTurnData](t, ws1, UpdateTypeTurn)
		u2 := assertRecieved[UpdateTurnData](t, ws2, UpdateTypeTurn)
		assertDataMatches(t, u1, UpdateTurnData{Player: "p2"})
		assertDataMatches(t, u2, UpdateTurnData{Player: "p2"})
	}

	{
		// p2 draws
		assert.NoError(t, ws2.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateDrawData](t, ws1, UpdateTypeDraw)
		u2 := assertRecieved[UpdateDrawData](t, ws2, UpdateTypeDraw)
		assertDataMatches(t, u1, UpdateDrawData{Player: "p2", Source: game.DrawSourcePile})
		assertDataMatches(t, u2, UpdateDrawData{Player: "p2", Source: game.DrawSourcePile, Effect: game.CardEffectNone, Card: deck[10]})
	}
	{
		// p2 discards two non-equal cards and fails
		assert.NoError(t, ws2.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: 1, CardPosition2: toIntPointer(0)}),
		}))
		u1 := assertRecieved[UpdateTypeFailedDoubleDiscardData](t, ws1, UpdateTypeFailedDoubleDiscard)
		u2 := assertRecieved[UpdateTypeFailedDoubleDiscardData](t, ws2, UpdateTypeFailedDoubleDiscard)
		expected2 := UpdateTypeFailedDoubleDiscardData{
			Player:         "p2",
			CardsPositions: []int{1, 0},
			Cards:          []game.Card{deck[5], deck[4]},
			TopOfDiscard:   deck[1],
			CycledPiles:    true,
		}
		assertDataMatches(t, u1, expected2)
		assertDataMatches(t, u2, expected2)
	}
}

func TestBasicGame(t *testing.T) {
	g, s, cancel := NewServer()
	defer cancel()
	defer s.Close()
	deck := game.NewDeck()
	roomID, err := g.NewRoom(slog.Default(), deck, 4, "")
	assert.NoError(t, err)
	ws1 := NewSocket(s, "p1", roomID)
	ws2 := NewSocket(s, "p2", roomID)
	defer ws1.Close()
	defer ws2.Close()

	assertRecieved[UpdatePlayersChangedData](t, ws1, UpdateTypePlayersChanged)
	assertRecieved[UpdatePlayersChangedData](t, ws1, UpdateTypePlayersChanged)
	assertRecieved[UpdatePlayersChangedData](t, ws2, UpdateTypePlayersChanged)

	// p1 starts game
	assert.NoError(t, ws1.WriteJSON(Action{Type: ActionStart}))

	{
		// both players recieve game config
		u1 := assertRecieved[UpdateGameConfig](t, ws1, UpdateTypeGameConfig)
		u2 := assertRecieved[UpdateGameConfig](t, ws2, UpdateTypeGameConfig)
		assertDataMatches(t, u1, UpdateGameConfig{CardsInDeck: 50})
		assertDataMatches(t, u2, UpdateGameConfig{CardsInDeck: 50})
	}

	{
		// both players prompted to peek
		u1 := assertRecieved[UpdateStartNextRoundData](t, ws1, UpdateTypeGameStart)
		u2 := assertRecieved[UpdateStartNextRoundData](t, ws2, UpdateTypeGameStart)
		expected := UpdateStartNextRoundData{
			Players: []*game.Player{
				{ID: "p1", PendingFirstPeek: true, Hand: make(game.Hand, 4)},
				{ID: "p2", PendingFirstPeek: true, Hand: make(game.Hand, 4)},
			},
			TopDiscard: deck[8],
		}
		assertDataMatches(t, u1, expected)
		assertDataMatches(t, u2, expected)
	}

	{
		// p1 peeks
		assert.NoError(t, ws1.WriteJSON(Action{Type: ActionFirstPeek}))
		u1 := assertRecieved[UpdatePlayerFirstPeekedData](t, ws1, UpdateTypePlayerFirstPeeked)
		u2 := assertRecieved[UpdatePlayerFirstPeekedData](t, ws2, UpdateTypePlayerFirstPeeked)
		assertDataMatches(t, u1, UpdatePlayerFirstPeekedData{Player: "p1", Cards: deck[:2]})
		assertDataMatches(t, u2, UpdatePlayerFirstPeekedData{Player: "p1", Cards: nil})

		// p2 peeks
		assert.NoError(t, ws2.WriteJSON(Action{Type: ActionFirstPeek}))
		u1 = assertRecieved[UpdatePlayerFirstPeekedData](t, ws1, UpdateTypePlayerFirstPeeked)
		u2 = assertRecieved[UpdatePlayerFirstPeekedData](t, ws2, UpdateTypePlayerFirstPeeked)
		assertDataMatches(t, u1, UpdatePlayerFirstPeekedData{Player: "p2", Cards: nil})
		assertDataMatches(t, u2, UpdatePlayerFirstPeekedData{Player: "p2", Cards: deck[4:6]})
	}

	// both recieve game start
	{
		u1 := assertRecieved[UpdateTurnData](t, ws1, UpdateTypeTurn)
		u2 := assertRecieved[UpdateTurnData](t, ws2, UpdateTypeTurn)
		assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
		assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})
	}

	{
		// p1 draws
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateDrawData](t, ws1, UpdateTypeDraw)
		u2 := assertRecieved[UpdateDrawData](t, ws2, UpdateTypeDraw)
		assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile, Effect: game.CardEffectNone, Card: deck[9]})
		assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile})
	}
	{
		// p1 tries to draw again and fails
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateErrorData](t, ws1, UpdateTypeError)
		assertDataMatches(t, u1, UpdateErrorData{Message: game.ErrPendingDiscard.Error()})
	}
	{
		// p1 discards second card
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
		}))
		u1 := assertRecieved[UpdateDiscardData](t, ws1, UpdateTypeDiscard)
		u2 := assertRecieved[UpdateDiscardData](t, ws2, UpdateTypeDiscard)
		assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []game.Card{deck[1]}})
		assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []game.Card{deck[1]}})
	}
	{
		// turn changes
		u1 := assertRecieved[UpdateTurnData](t, ws1, UpdateTypeTurn)
		u2 := assertRecieved[UpdateTurnData](t, ws2, UpdateTypeTurn)
		assertDataMatches(t, u1, UpdateTurnData{Player: "p2"})
		assertDataMatches(t, u2, UpdateTurnData{Player: "p2"})
	}
	{
		// p1 tries to discard again and fails
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
		}))
		u1 := assertRecieved[UpdateErrorData](t, ws1, UpdateTypeError)
		assertDataMatches(t, u1, UpdateErrorData{Message: ErrNotYourTurn.Error()})

		// p1 tries to draw again and fails
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 = assertRecieved[UpdateErrorData](t, ws1, UpdateTypeError)
		assertDataMatches(t, u1, UpdateErrorData{Message: ErrNotYourTurn.Error()})
	}
	{
		// p2 draws
		assert.NoError(t, ws2.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateDrawData](t, ws1, UpdateTypeDraw)
		u2 := assertRecieved[UpdateDrawData](t, ws2, UpdateTypeDraw)
		assertDataMatches(t, u1, UpdateDrawData{Player: "p2", Source: game.DrawSourcePile})
		assertDataMatches(t, u2, UpdateDrawData{Player: "p2", Source: game.DrawSourcePile, Effect: game.CardEffectNone, Card: deck[10]})
	}
	{
		// p2 discards drawn card card
		assert.NoError(t, ws2.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: -1}),
		}))
		u1 := assertRecieved[UpdateDiscardData](t, ws1, UpdateTypeDiscard)
		u2 := assertRecieved[UpdateDiscardData](t, ws2, UpdateTypeDiscard)
		assertDataMatches(t, u1, UpdateDiscardData{Player: "p2", CardsPositions: []int{-1}, Cards: []game.Card{deck[10]}})
		assertDataMatches(t, u2, UpdateDiscardData{Player: "p2", CardsPositions: []int{-1}, Cards: []game.Card{deck[10]}})
	}
	{
		// turn changes
		u1 := assertRecieved[UpdateTurnData](t, ws1, UpdateTypeTurn)
		u2 := assertRecieved[UpdateTurnData](t, ws2, UpdateTypeTurn)
		assertDataMatches(t, u1, UpdateTurnData{Player: "p1"})
		assertDataMatches(t, u2, UpdateTurnData{Player: "p1"})
	}
	{
		// p1 draws
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDraw,
			Data: safeMarshal(t, ActionDrawData{Source: game.DrawSourcePile}),
		}))
		u1 := assertRecieved[UpdateDrawData](t, ws1, UpdateTypeDraw)
		u2 := assertRecieved[UpdateDrawData](t, ws2, UpdateTypeDraw)
		assertDataMatches(t, u1, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile, Effect: game.CardEffectNone, Card: deck[11]})
		assertDataMatches(t, u2, UpdateDrawData{Player: "p1", Source: game.DrawSourcePile})
	}
	{
		// p1 discards second card
		assert.NoError(t, ws1.WriteJSON(Action{
			Type: ActionDiscard,
			Data: safeMarshal(t, ActionDiscardData{CardPosition: 1}),
		}))
		u1 := assertRecieved[UpdateDiscardData](t, ws1, UpdateTypeDiscard)
		u2 := assertRecieved[UpdateDiscardData](t, ws2, UpdateTypeDiscard)
		assertDataMatches(t, u1, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []game.Card{deck[9]}})
		assertDataMatches(t, u2, UpdateDiscardData{Player: "p1", CardsPositions: []int{1}, Cards: []game.Card{deck[9]}})

	}
}

func safeMarshal(t *testing.T, v interface{}) []byte {
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return b
}

func assertRecieved[D UpdateData](t *testing.T, ws *websocket.Conn, updateType UpdateType) Update[D] {
	type Temp struct {
		Type UpdateType       `json:"type"`
		Data *json.RawMessage `json:"data"`
	}
	var temp Update[D]
	_, message, err := ws.ReadMessage()
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(message, &temp))
	assert.Equal(t, updateType, temp.Type)
	return temp
}

func assertDataMatches[D UpdateData](t *testing.T, update Update[D], expected D) {
	assert.Equal(t, expected, update.Data)
}

func toIntPointer(i int) *int {
	return &i
}
