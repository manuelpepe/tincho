package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// Room represents an ongoing game and contains all necessary state to represent it.
type Room struct {
	Context   context.Context
	closeRoom context.CancelFunc
	ID        string
	state     *Tincho

	// actions recieved from all players
	ActionsChan chan Action

	// channel used to update goroutine state
	NewPlayersChan chan Player

	started bool
	closed  bool
}

func NewRoomWithDeck(ctx context.Context, ctxCancel context.CancelFunc, roomID string, deck Deck) Room {
	return Room{
		Context:        ctx,
		closeRoom:      ctxCancel,
		ID:             roomID,
		ActionsChan:    make(chan Action),
		NewPlayersChan: make(chan Player),
		state:          NewTinchoWithDeck(deck),
		closed:         false,
	}
}

func (r *Room) HasClosed() bool {
	return r.started && r.closed
}

func (r *Room) Close() {
	r.closeRoom()
	r.closed = true
}
func (r *Room) AddPlayer(p Player) {
	r.NewPlayersChan <- p
}

func (r *Room) addPlayer(player Player) error {
	if err := r.state.AddPlayer(player); err != nil {
		return fmt.Errorf("tsm.AddPlayer: %w", err)
	}
	go r.watchPlayer(&player)
	data, err := json.Marshal(UpdatePlayersChanged{
		Players: r.state.GetPlayers(),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	update := Update{
		Type: UpdateTypePlayersChanged,
		Data: data,
	}
	r.BroadcastUpdate(update)
	return nil
}

func (r *Room) Start() {
	r.started = true
	for {
		select {
		case player := <-r.NewPlayersChan:
			if err := r.addPlayer(player); err != nil {
				fmt.Printf("r.addPlayer: %s\n", err)
			}
			log.Printf("Player joined #%s: %+v\n", r.ID, player)
		case action := <-r.ActionsChan:
			log.Printf("Recieved from %s: {Type: %s Data:%s}\n", action.PlayerID, action.Type, action.Data)
			r.doAction(action)
		case <-r.Context.Done():
			log.Printf("Stopping room %s", r.ID)
			r.Close()
			return
		}
	}
}

var ErrNotYourTurn = fmt.Errorf("not your turn")
var ErrActionOnClosedRoom = errors.New("action on closed room")

func (r *Room) doAction(action Action) {
	if r.HasClosed() {
		log.Printf("ERR: %s", ErrActionOnClosedRoom)
		r.TargetedError(action.PlayerID, ErrActionOnClosedRoom)
	}
	switch action.Type {
	case ActionStart:
		if err := r.doStartGame(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionFirstPeek:
		if err := r.doPeekTwo(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	}
	if !r.state.playing || action.PlayerID != r.state.PlayerToPlay().ID {
		log.Printf("Player %s tried to perform action '%s' out of turn", action.PlayerID, action.Type)
		r.TargetedError(action.PlayerID, ErrNotYourTurn)
		return
	}
	switch action.Type {
	case ActionDraw:
		if err := r.doDraw(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionDiscard:
		if err := r.doDiscard(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionCut:
		if err := r.doCut(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionPeekOwnCard:
		if err := r.doEffectPeekOwnCard(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionPeekCartaAjena:
		if err := r.doEffectPeekCartaAjena(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionSwapCards:
		if err := r.doEffectSwapCards(action); err != nil {
			log.Println(err)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	default:
		log.Println("unknown action")
	}
}

// watchPlayer functions as a goroutine that watches for new actions from a given player.
func (r *Room) watchPlayer(player *Player) {
	log.Printf("Watching player '%s' on room '%s'", player.ID, r.ID)
	for {
		select {
		case action := <-player.Actions:
			r.ActionsChan <- action
		case <-r.Context.Done():
			log.Printf("Stopping watch loop for player %s", player.ID)
			return
		}

	}
}
