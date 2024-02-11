package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
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

// Start initiates a goroutine that processes messages from all websocket connections.
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

func (r *Room) addPlayer(player Player) error {
	if err := r.state.AddPlayer(player); err != nil {
		return fmt.Errorf("tsm.AddPlayer: %w", err)
	}
	go r.watchPlayer(&player)
	go r.updatePlayer(&player)
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

func (r *Room) BroadcastUpdate(update Update) {
	for _, player := range r.state.GetPlayers() {
		player.Updates <- update
	}
}

func (r *Room) BroadcastUpdateExcept(update Update, player string) {
	for _, p := range r.state.GetPlayers() {
		if p.ID != player {
			p.Updates <- update
		}
	}
}

func (r *Room) TargetedUpdate(player string, update Update) {
	for _, p := range r.state.GetPlayers() {
		if p.ID == player {
			p.Updates <- update
			return
		}
	}
}

func (r *Room) TargetedError(player string, err error) {
	data, err := json.Marshal(UpdateErrorData{
		Message: err.Error(),
	})
	if err != nil {
		log.Println(err)
		return
	}
	r.TargetedUpdate(player, Update{
		Type: UpdateTypeError,
		Data: data,
	})
}

func (r *Room) AddPlayer(p Player) {
	r.NewPlayersChan <- p
}

func (r *Room) GetPlayer(playerID string) (Player, bool) {
	for _, room := range r.state.GetPlayers() {
		if room.ID == playerID {
			return room, true
		}
	}
	return Player{}, false
}

func (r *Room) PlayerCount() int {
	return len(r.state.GetPlayers())
}

// watchPlayer functions as a goroutine that watches for messages from a given player.
func (r *Room) watchPlayer(player *Player) {
	log.Printf("Watching player '%s' on room '%s'", player.ID, r.ID)
	tick := time.NewTicker(1 * time.Second) // TODO: Make global
	for {
		select {
		case <-tick.C:
			_, message, err := player.socket.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from player %s: %s", player.ID, err)
				return
			}
			var action Action
			if err := json.Unmarshal(message, &action); err != nil {
				log.Println(err)
				return // TODO: Prevent disconnect
			}
			action.PlayerID = player.ID
			r.ActionsChan <- action
		case <-r.Context.Done():
			log.Printf("Stopping watch loop for player %s", player.ID)
			return
		}

	}
}

// updatePlayer functions as a goroutine that sends updates to a given player.
func (r *Room) updatePlayer(player *Player) {
	log.Printf("Updating player %s on room %s", player.ID, r.ID)
	for {
		select {
		case update := <-player.Updates:
			log.Printf("Sending update to player %s: {Type:%s, Data:\"%s\"}", player.ID, update.Type, update.Data)
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
