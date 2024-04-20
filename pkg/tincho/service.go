package tincho

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/manuelpepe/tincho/pkg/game"
)

const ROOM_ID_LENGTH = 4

var ErrRoomNotFound = errors.New("room not found")
var ErrRoomsLimitReached = errors.New("rooms limit reached")

type ServiceConfig struct {
	MaxRooms    int
	RoomTimeout time.Duration
}

// Service is the object keeping state of all games.
// Contains a map of rooms, where the key is the room ID.
type Service struct {
	context   context.Context
	rooms     []*Room
	passwords map[string]string
	cfg       ServiceConfig
}

func NewService(ctx context.Context, cfg ServiceConfig) Service {
	return Service{
		context:   ctx,
		rooms:     make([]*Room, 0, cfg.MaxRooms),
		passwords: make(map[string]string),
		cfg:       cfg,
	}
}

func (g *Service) NewRoom(logger *slog.Logger, deck game.Deck, maxPlayers int, password string) (string, error) {
	if maxPlayers <= 0 {
		return "", fmt.Errorf("max players should be greater than 0, got %d", maxPlayers)
	}
	if g.ActiveRoomCount() >= g.cfg.MaxRooms {
		return "", ErrRoomsLimitReached
	}
	ctx, cancel := context.WithTimeout(g.context, g.cfg.RoomTimeout)
	roomID := g.getUnusedID()
	roomLogger := logger.With("room_id", roomID, "component", "room")
	room := NewRoomWithDeck(roomLogger, ctx, cancel, roomID, deck, maxPlayers)
	g.rooms = append(g.rooms, &room)
	if password != "" {
		g.passwords[roomID] = password
	}
	go room.Start()
	return room.ID, nil
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

func (g *Service) GetRoomPassword(roomID string) string {
	return g.passwords[roomID]
}

func (g *Service) JoinRoom(roomID string, player *Connection, password string) error {
	if pass, exists := g.passwords[roomID]; exists && pass != password {
		return fmt.Errorf("invalid password")
	}
	return g.JoinRoomWithoutPassword(roomID, player)
}

func (g *Service) JoinRoomWithoutPassword(roomID string, player *Connection) error {
	room, exists := g.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrRoomNotFound, roomID)
	}
	if err := room.AddPlayer(player); err != nil {
		return err
	}
	return nil
}

func (g *Service) ActiveRoomCount() int {
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
		delete(g.passwords, g.rooms[idx].ID)
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

func (g *Service) getUnusedID() string {
	roomID := generateRandomString(ROOM_ID_LENGTH)
	for exists := true; exists; _, exists = g.getRoomIndex(roomID) {
		roomID = generateRandomString(ROOM_ID_LENGTH)
	}
	return roomID
}

// Function to generate a random string with a given length
func generateRandomString(length int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
