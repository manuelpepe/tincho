package tincho

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")
var ErrGameAlreadyStarted = errors.New("game already started")

type Player struct {
	ID               string          `json:"id"`
	Points           int             `json:"points"`
	PendingFirstPeek bool            `json:"pending_first_peek"`
	Hand             Hand            `json:"-"`
	socket           *websocket.Conn `json:"-"`
	Updates          chan Update     `json:"-"`
}

func NewPlayer(id string, socket *websocket.Conn) Player {
	return Player{
		ID:      id,
		Hand:    make(Hand, 0),
		socket:  socket,
		Updates: make(chan Update),
		Points:  0,
	}
}

// Game is the object keeping state of all games.
// Contains a map of rooms, where the key is the room ID.
type Game struct {
	context context.Context
	rooms   []Room
}

func NewGame(ctx context.Context) Game {
	return Game{
		context: ctx,
		rooms:   make([]Room, 0),
	}
}

// Function to generate a random string with a given length
func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func (g *Game) getUnusedID() string {
	roomID := generateRandomString(6)
	for exists := true; exists; _, exists = g.GetRoomIndex(roomID) {
		roomID = generateRandomString(6)
	}
	return roomID
}

func (g *Game) NewRoom() string {
	deck := NewDeck()
	deck.Shuffle()
	return g.NewRoomWithDeck(deck)
}

func (g *Game) NewRoomWithDeck(deck Deck) string {
	roomID := g.getUnusedID()
	room := NewRoomWithDeck(g.context, roomID, deck)
	g.rooms = append(g.rooms, room)
	go room.Start()
	return roomID
}

func (g *Game) GetRoomIndex(roomID string) (int, bool) {
	for idx, room := range g.rooms {
		if room.ID == roomID {
			return idx, true
		}
	}
	return 0, false
}

func (g *Game) JoinRoom(roomID string, player Player) error {
	roomix, exists := g.GetRoomIndex(roomID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrRoomNotFound, roomID)
	}
	g.rooms[roomix].AddPlayer(player)
	return nil
}
