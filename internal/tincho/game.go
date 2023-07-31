package tincho

import (
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")
var ErrGameAlreadyStarted = errors.New("game already started")

type Player struct {
	ID      string
	Hand    Hand
	socket  *websocket.Conn
	Updates chan Update
}

func NewPlayer(id string, socket *websocket.Conn) Player {
	return Player{
		ID:      id,
		Hand:    make(Hand, 0),
		socket:  socket,
		Updates: make(chan Update),
	}
}

// Game is the object keeping state of all games.
// Contains a map of rooms, where the key is the room ID.
type Game struct {
	rooms []Room
}

func NewGame() Game {
	return Game{
		rooms: make([]Room, 0),
	}
}

func (g *Game) getUnusedID() string {
	roomID := generateRandomString(6)
	for exists := true; exists; _, exists = g.GetRoomIndex(roomID) {
		roomID = generateRandomString(6)
	}
	return roomID
}

func (g *Game) NewRoom() string {
	roomID := g.getUnusedID()
	room := NewRoom(roomID)
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
	if err := g.rooms[roomix].AddPlayer(player); err != nil {
		return fmt.Errorf("JoinRoom: %w", err)
	}
	return nil
}
