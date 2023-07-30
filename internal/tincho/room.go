package tincho

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Room represents an ongoing game and contains all necessary state to represent it.
type Room struct {
	Context     context.Context
	ID          string
	Players     []Player
	Playing     bool
	DrawPile    Deck
	DiscardPile Deck

	// the last card drawn that has not been stored into a player's hand
	PendingStorage Card

	// actions recieved from all players
	Actions chan Action

	// index of the player whose turn it is
	CurrentTurn int
}

func NewRoom(roomID string) Room {
	return Room{
		Context:     context.Background(),
		ID:          roomID,
		Playing:     false,
		DrawPile:    NewDeck(),
		DiscardPile: make(Deck, 0),
		Actions:     make(chan Action),
	}
}

// Start initiates a goroutine that processes messages from all websocket connections.
func (r *Room) Start() {
	for {
		select {
		case action := <-r.Actions:
			fmt.Printf("Recieved: %+v\n", action)
			r.doAction(action)
		case <-r.Context.Done():
			log.Printf("Stopping room %s", r.ID)
			return
		}
	}
}

func (r *Room) doAction(action Action) {
	// TODO: Check action is performed by the player whose turn it is
	switch action.Type {
	case ActionStart:
		r.doStartGame(action)
	case ActionDraw:
		if err := r.doDraw(action); err != nil {
			log.Println(err)
			return
		}
	case ActionDiscard:
		if err := r.doDiscard(action); err != nil {
			log.Println(err)
			return
		}
	case ActionCut:
		if err := r.doCut(action); err != nil {
			log.Println(err)
			return
		}
	case ActionPeekOwnCard:
		return
	case ActionPeekCartaAjena:
		return
	case ActionSwapCards:
		return
	default:
		log.Println("unknown action")
	}
}

func (r *Room) BroadcastUpdate(update Update) {
	for _, player := range r.Players {
		player.Updates <- update
	}
}

func (r *Room) TargetedUpdate(player string, update Update) {
	for _, p := range r.Players {
		if p.ID == player {
			p.Updates <- update
			return
		}
	}
}

func (r *Room) ReshufflePiles() error {
	r.DrawPile = r.DiscardPile
	r.DrawPile.Shuffle()
	r.DiscardPile = make(Deck, 0)
	return r.DiscardTopCard()
}

func (r *Room) DiscardTopCard() error {
	card, err := r.DrawPile.Draw()
	if err != nil {
		return err
	}
	r.DiscardPile = append(r.DiscardPile, card)
	return nil
}

func (r *Room) Deal() error {
	for i := 0; i < 4; i++ {
		for pid := range r.Players {
			card, err := r.DrawPile.Draw()
			if err != nil {
				return err
			}
			r.Players[pid].Hand = append(r.Players[pid].Hand, card)
		}
	}
	return nil
}

func (r *Room) AddPlayer(p Player) error {
	// TODO: Implement reconnection with auth
	if r.Playing {
		return fmt.Errorf("%w: %s", ErrGameAlreadyStarted, r.ID)
	}
	if _, exists := r.GetPlayer(p.ID); exists {
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
			action.PlayerID = player.ID
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
