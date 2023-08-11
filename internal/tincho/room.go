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
	Context context.Context
	ID      string
	state   *Tincho

	// actions recieved from all players
	ActionsChan chan Action

	// channel used to update goroutine state
	NewPlayersChan chan Player
}

func NewRoomWithDeck(ctx context.Context, roomID string, deck Deck) Room {
	return Room{
		Context:        ctx,
		ID:             roomID,
		ActionsChan:    make(chan Action),
		NewPlayersChan: make(chan Player),
		state:          NewTinchoWithDeck(deck),
	}
}

// Start initiates a goroutine that processes messages from all websocket connections.
func (r *Room) Start() {
	for {
		select {
		case player := <-r.NewPlayersChan:
			if err := r.state.AddPlayer(player); err != nil {
				fmt.Printf("tsm.AddPlayer: %s\n", err)
			}
			go r.watchPlayer(&player)
			go r.updatePlayer(&player)
			fmt.Printf("Player joined #%s: %+v\n", r.ID, player)
		case action := <-r.ActionsChan:
			fmt.Printf("Recieved from %s: {Type: %s Data:%s}\n", action.PlayerID, action.Type, action.Data)
			r.doAction(action)
		case <-r.Context.Done():
			log.Printf("Stopping room %s", r.ID)
			return
		}
	}
}

var ErrNotYourTurn = fmt.Errorf("not your turn")

func (r *Room) doAction(action Action) {
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

// watchPlayer functions as a goroutine that watches for messages from a given player.
func (r *Room) watchPlayer(player *Player) {
	log.Printf("Watching player %+v on room %s", player, r.ID)
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
