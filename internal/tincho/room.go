package tincho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/manuelpepe/tincho/internal/game"
)

type AddPlayerRequest struct {
	Player *Connection
	Res    chan error
}

// Room represents an ongoing game and contains all necessary state to represent it.
type Room struct {
	Context   context.Context
	closeRoom context.CancelFunc
	logger    *slog.Logger

	ID          string
	state       *game.Tincho
	connections map[game.PlayerID]*Connection

	// actions recieved from all players
	actionsChan chan Action

	// channel used to update goroutine state
	playersChan chan AddPlayerRequest

	maxPlayers int

	started bool
	closed  bool

	sync.RWMutex
}

func NewRoomWithDeck(logger *slog.Logger, ctx context.Context, ctxCancel context.CancelFunc, roomID string, deck game.Deck, maxPlayers int) Room {
	return Room{
		Context:     ctx,
		closeRoom:   ctxCancel,
		logger:      logger,
		ID:          roomID,
		actionsChan: make(chan Action),
		playersChan: make(chan AddPlayerRequest),
		maxPlayers:  maxPlayers,
		state:       game.NewTinchoWithDeck(deck),
		connections: make(map[game.PlayerID]*Connection),
		closed:      false,
	}
}

func (r *Room) Winner() (*game.Player, error) {
	return r.state.Winner()
}

func (r *Room) TotalTurns() int {
	return r.state.TotalTurns()
}

func (r *Room) TotalRounds() int {
	return r.state.TotalRounds()
}

func (r *Room) CurrentPlayers() int {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	return len(r.state.GetPlayers())
}

func (r *Room) HasClosed() bool {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	return r.closed
}

func (r *Room) Close() {
	if !r.closed {
		r.RWMutex.Lock()
		defer r.RWMutex.Unlock()
		r.closeRoom()
		r.closed = true
	}
}

func (r *Room) GetPlayer(id game.PlayerID) (*Connection, bool) {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	_, exists := r.state.GetPlayer(id)
	if !exists {
		return nil, false
	}
	conn, ok := r.connections[id]
	if !ok {
		return nil, false
	}
	return conn, true
}

func (r *Room) AddPlayer(p *Connection) error {
	req := AddPlayerRequest{
		Player: p,
		Res:    make(chan error),
	}
	r.playersChan <- req
	return <-req.Res
}

func (r *Room) addPlayer(player *Connection) error {
	r.RWMutex.Lock()
	if len(r.state.GetPlayers()) >= r.maxPlayers {
		return fmt.Errorf("room is full")
	}
	if err := r.state.AddPlayer(player.Player); err != nil {
		return fmt.Errorf("tsm.AddPlayer: %w", err)
	}
	r.RWMutex.Unlock()

	r.connections[player.ID] = player
	go r.watchPlayer(player)
	data, err := json.Marshal(UpdatePlayersChangedData{
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

func (r *Room) IsPlayerInRoom(playerID game.PlayerID) bool {
	_, exists := r.state.GetPlayer(playerID)
	return exists
}

func (r *Room) Start() {
	r.logger.Info("Starting room")
	r.started = true
	for {
		select {
		case req := <-r.playersChan:
			if r.IsPlayerInRoom(req.Player.ID) {
				req.Player.ClearPendingUpdates()
				if err := r.sendRejoinState(req.Player, r.state.CountBaseDeck(), r.state.CountDrawPile()); err != nil {
					r.logger.Error("r.sendRejoinState: %s", err, "player", req.Player)
					req.Res <- err
				} else {
					r.logger.Info(fmt.Sprintf("Player rejoined #%s: %s", r.ID, req.Player.ID))
					req.Res <- nil
				}
			} else {
				if err := r.addPlayer(req.Player); err != nil {
					r.logger.Error("r.addPlayer: %s", err, "player", req.Player)
					req.Res <- err
				} else {
					r.logger.Info(fmt.Sprintf("Player joined #%s: %s", r.ID, req.Player.ID))
					req.Res <- nil
				}
			}
		case action := <-r.actionsChan:
			r.logger.Info(fmt.Sprintf("Recieved action from %s", action.PlayerID), "action", action)
			r.doAction(action)
		case <-r.Context.Done():
			r.logger.Info("Stopping room")
			r.Close()
			return
		}
	}
}

var ErrNotYourTurn = fmt.Errorf("not your turn")
var ErrActionOnClosedRoom = errors.New("action on closed room")

func (r *Room) doAction(action Action) {
	if r.HasClosed() {
		r.logger.Error(ErrActionOnClosedRoom.Error())
		r.TargetedError(action.PlayerID, ErrActionOnClosedRoom)
		return
	}
	switch action.Type {
	case ActionStart:
		if err := r.doStartGame(action); err != nil {
			r.logger.Warn("error starting game", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionFirstPeek:
		if err := r.doPeekTwo(action); err != nil {
			r.logger.Warn("error on first peek", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	}
	if !r.state.Playing() || action.PlayerID != r.state.PlayerToPlay().ID {
		r.logger.Warn(
			fmt.Sprintf("Player %s tried to perform action out of turn", action.PlayerID),
			"player_id", action.PlayerID,
			"action", action)
		r.TargetedError(action.PlayerID, ErrNotYourTurn)
		return
	}
	switch action.Type {
	case ActionDraw:
		if err := r.doDraw(action); err != nil {
			r.logger.Warn("error on draw", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionDiscard:
		if err := r.doDiscard(action); err != nil {
			r.logger.Warn("error on discard", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionCut:
		if err := r.doCut(action); err != nil {
			r.logger.Warn("error on cut", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
	case ActionPeekOwnCard:
		if err := r.doEffectPeekOwnCard(action); err != nil {
			r.logger.Warn("error on peek own", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionPeekCartaAjena:
		if err := r.doEffectPeekCartaAjena(action); err != nil {
			r.logger.Warn("error on peek carta ajena", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	case ActionSwapCards:
		if err := r.doEffectSwapCards(action); err != nil {
			r.logger.Warn("error on swap cards", "err", err, "player_id", action.PlayerID)
			r.TargetedError(action.PlayerID, err)
			return
		}
		return
	default:
		r.logger.Warn("unknown action", "player_id", action.PlayerID, "action", action)
	}
}

// watchPlayer functions as a goroutine that watches for new actions from a given player.
func (r *Room) watchPlayer(player *Connection) {
	r.logger.Info(fmt.Sprintf("Started watch loop for player '%s' on room '%s'", player.ID, r.ID))
	for {
		select {
		case action := <-player.Actions:
			r.actionsChan <- action
		case <-r.Context.Done():
			r.logger.Info(fmt.Sprintf("Stopping watch loop for player '%s' on room '%s'", player.ID, r.ID))
			return
		}

	}
}
