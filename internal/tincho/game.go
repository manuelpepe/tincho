package tincho

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")

type Player struct {
	ID     string
	Name   string
	socket *websocket.Conn
}

// Room represents an ongoing game and contains all necessary state to represent it.
type Room struct {
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
		ID:      roomID,
		Playing: false,
		Players: make([]Player, 0),
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
		}
	}
}

func (r *Room) AddPlayer(p Player) error {
	if _, exists := r.GetPlayer(p.ID); exists {
		return fmt.Errorf("%w: %s in %s", ErrPlayerAlreadyInRoom, p.ID, r.ID)
	}
	r.Players = append(r.Players, p)
	go r.watchPlayer(p)
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

func (r *Room) watchPlayer(player Player) {
	log.Printf("Watching player %s on room %s", player.ID, r.ID)
	for {
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
	for exists := true; exists; _, exists = g.GetRoom(roomID) {
		roomID = generateRandomString(6)
	}
	room := NewRoom(roomID)
	g.rooms = append(g.rooms, room)
	go room.Start()
	return roomID
}

func (g *Game) GetRoom(roomID string) (*Room, bool) {
	for _, room := range g.rooms {
		if room.ID == roomID {
			return &room, true
		}
	}
	return &Room{}, false
}

func (g *Game) JoinRoom(roomID string, player Player) error {
	room, exists := g.GetRoom(roomID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrRoomNotFound, roomID)
	}
	if err := room.AddPlayer(player); err != nil {
		return fmt.Errorf("JoinRoom: %w", err)
	}
	return nil
}
