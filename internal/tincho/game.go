package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")

type Player struct {
	ID      string
	socket  *websocket.Conn
	Updates chan Update
}

// Room represents an ongoing game and contains all necessary state to represent it.
type Room struct {
	Context context.Context
	ID      string
	Players []Player
	Deck    Deck
	Playing bool

	// actions recieved from all players
	Actions chan Action

	// index of the player whose turn it is
	CurrentTurn int
}

func NewRoom(roomID string) Room {
	return Room{
		Context: context.Background(),
		ID:      roomID,
		Playing: false,
		Deck:    NewDeck(),
		Actions: make(chan Action),
	}
}

// Start initiates a goroutine that processes messages from all websocket connections.
func (r *Room) Start() {
	for {
		select {
		case action := <-r.Actions:
			fmt.Printf("%+v\n", action)
			switch action.Type {
			case ActionStart:
				log.Println("start")
			case ActionDraw:
				log.Println("draw")
			case ActionDiscard:
				log.Println("discard")
			case ActionCut:
				log.Println("cut")
			default:
				log.Println("unknown action")
			}
		case <-r.Context.Done():
			log.Printf("Stopping room %s", r.ID)
			return
		}
	}
}

func (r *Room) AddPlayer(p Player) error {
	if _, exists := r.GetPlayer(p.ID); exists {
		// TODO: Implement reconnection with auth
		return fmt.Errorf("%w: %s in %s", ErrPlayerAlreadyInRoom, p.ID, r.ID)
	}
	r.Players = append(r.Players, p)
	go r.watchPlayer(p)
	go r.updatePlayer(p)
	return nil
}

func (r *Room) GetPlayer(playerID string) (Player, bool) {
	for _, room := range r.Players {
		if room.ID == playerID {
			return room, true
		}
	}
	return Player{}, false
}

// watchPlayer functions as a goroutine that watches for messages from a given player.
func (r *Room) watchPlayer(player Player) {
	log.Printf("Watching player %s on room %s", player.ID, r.ID)
	tick := time.NewTicker(1 * time.Second) // TODO: Make global
	for {
		select {
		case <-tick.C:
			_, message, err := player.socket.ReadMessage()
			if err != nil {
				log.Println(err)
				return
			}
			var action Action
			if err := json.Unmarshal(message, &action); err != nil {
				log.Println(err)
				return // TODO: Prevent disconnect
			}
			r.Actions <- action
		case <-r.Context.Done():
			log.Printf("Stopping watch loop for player %s", player.ID)
			return
		}

	}
}

// updatePlayer functions as a goroutine that sends updates to a given player.
func (r *Room) updatePlayer(player Player) {
	log.Printf("Updating player %s on room %s", player.ID, r.ID)
	for {
		select {
		case update := <-player.Updates:
			log.Printf("Sending update to player %s: %+v", player.ID, update)
			if err := player.socket.WriteJSON(update); err != nil {
				log.Println(err)
				return
			}
		case <-r.Context.Done():
			log.Printf("Stopping update loop for player %s", player.ID)
			return
		}
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

// NewRoom creates a new room with an unused random ID and no players.
func (g *Game) NewRoom() string {
	roomID := generateRandomString(6)
	for exists := true; exists; _, exists = g.GetRoomIndex(roomID) {
		roomID = generateRandomString(6)
	}
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
