package tincho

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrRoomsLimitReached = errors.New("rooms limit reached")

type ServiceConfig struct {
	MaxRooms    int
	RoomTimeout time.Duration
}

// Service is the object keeping state of all games.
// Contains a map of rooms, where the key is the room ID.
type Service struct {
	context context.Context
	rooms   []*Room
	cfg     ServiceConfig
}

func NewService(ctx context.Context, cfg ServiceConfig) Service {
	return Service{
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

func (g *Service) getUnusedID() string {
	roomID := generateRandomString(6)
	for exists := true; exists; _, exists = g.getRoomIndex(roomID) {
		roomID = generateRandomString(6)
	}
	return roomID
}

func (g *Service) NewRoomBasic() (string, error) {
	deck := NewDeck()
	deck.Shuffle()
	return g.NewRoom(deck)
}

func (g *Service) NewRoom(deck Deck) (string, error) {
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

func (g *Service) getRoomIndex(roomID string) (int, bool) {
	for idx, room := range g.rooms {
		if room != nil && room.ID == roomID {
			return idx, true
		}
	}
	return 0, false
}

func (g *Service) GetRoom(roomID string) (*Room, bool) {
	roomix, exists := g.getRoomIndex(roomID)
	if !exists {
		return nil, false
	}
	return g.rooms[roomix], true
}

func (g *Service) JoinRoom(roomID string, player *Player) error {
	room, exists := g.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrRoomNotFound, roomID)
	}
	room.AddPlayer(player)
	return nil
}

func (g *Service) ActiveRooms() int {
	g.ClearClosedRooms()
	return len(g.rooms)
}

func (g *Service) ClearClosedRooms() {
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

// unordered_remove removes the element at index i from a slice changing its length
// and without preserving the order of the elements.
func unordered_remove[T any](a []T, i int) []T {
	a[i] = a[len(a)-1]
	a = a[:len(a)-1]
	return a
}
