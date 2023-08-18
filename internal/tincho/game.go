package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrRoomsLimitReached = errors.New("rooms limit reached")
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

func (p Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID               string `json:"id"`
		Points           int    `json:"points"`
		PendingFirstPeek bool   `json:"pending_first_peek"`
		CardsInHand      int    `json:"cards_in_hand"`
	}{
		ID:               p.ID,
		Points:           p.Points,
		PendingFirstPeek: p.PendingFirstPeek,
		CardsInHand:      len(p.Hand),
	})
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

type GameConfig struct {
	MaxRooms    int
	RoomTimeout time.Duration
}

// Game is the object keeping state of all games.
// Contains a map of rooms, where the key is the room ID.
type Game struct {
	context context.Context
	rooms   []*Room
	cfg     GameConfig
}

func NewGame(ctx context.Context, cfg GameConfig) Game {
	return Game{
		context: ctx,
		rooms:   make([]*Room, 0, cfg.MaxRooms),
		cfg:     cfg,
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

func (g *Game) NewRoom() (string, error) {
	deck := NewDeck()
	deck.Shuffle()
	return g.NewRoomWithDeck(deck)
}

func (g *Game) NewRoomWithDeck(deck Deck) (string, error) {
	if g.ActiveRooms() >= g.cfg.MaxRooms {
		return "", ErrRoomsLimitReached
	}
	roomID := g.getUnusedID()
	ctx, cancel := context.WithTimeout(g.context, g.cfg.RoomTimeout)
	room := NewRoomWithDeck(ctx, cancel, roomID, deck)
	g.rooms = append(g.rooms, &room)
	go room.Start()
	return roomID, nil
}

func (g *Game) GetRoomIndex(roomID string) (int, bool) {
	for idx, room := range g.rooms {
		if room != nil && room.ID == roomID {
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

func (g *Game) ActiveRooms() int {
	g.ClearClosedRooms()
	return len(g.rooms)
}

func (g *Game) ClearClosedRooms() {
	toRemove := make([]int, 0)
	for idx, room := range g.rooms {
		if room != nil && room.HasClosed() {
			toRemove = append([]int{idx}, toRemove...)
		}
	}
	for _, idx := range toRemove {
		g.rooms = unordered_remove(g.rooms, idx)
	}
}

func unordered_remove[T any](a []T, i int) []T {
	a[i] = a[len(a)-1]
	a = a[:len(a)-1]
	return a
}
