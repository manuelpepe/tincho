package tincho

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/metrics"
)

type AddConnectionRequest struct {
	Conn *Connection
	Res  chan error
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
	actionsChan chan TypedAction

	// channel used to update goroutine state
	connectionsChan chan AddConnectionRequest

	maxPlayers int

	started bool
	closed  bool

	sync.RWMutex
}

func NewRoomWithDeck(logger *slog.Logger, ctx context.Context, ctxCancel context.CancelFunc, roomID string, deck game.Deck, maxPlayers int) Room {
	return Room{
		Context:         ctx,
		closeRoom:       ctxCancel,
		logger:          logger,
		ID:              roomID,
		actionsChan:     make(chan TypedAction),
		connectionsChan: make(chan AddConnectionRequest),
		maxPlayers:      maxPlayers,
		state:           game.NewTinchoWithDeck(deck),
		connections:     make(map[game.PlayerID]*Connection),
		closed:          false,
	}
}

func (r *Room) Winner() (*game.Player, error) {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	return r.state.Winner()
}

func (r *Room) TotalTurns() int {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	return r.state.TotalTurns()
}

func (r *Room) TotalRounds() int {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
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

func (r *Room) close() {
	if !r.closed {
		r.closeRoom()
		r.closed = true
	}
}

func (r *Room) getMarshalledPlayers() []MarshalledPlayer {
	ps := r.state.GetPlayers()
	marshalled := make([]MarshalledPlayer, 0, len(ps))
	for _, p := range ps {
		marshalled = append(marshalled, NewMarshalledPlayer(p))
	}
	return marshalled
}

func (r *Room) GetConnection(id game.PlayerID) (*Connection, bool) {
	r.RWMutex.RLock()
	defer r.RWMutex.RUnlock()
	return r.getConnection(id)
}

func (r *Room) getConnection(id game.PlayerID) (*Connection, bool) {
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

func (r *Room) AddConnection(c *Connection) error {
	req := AddConnectionRequest{
		Conn: c,
		Res:  make(chan error),
	}
	r.connectionsChan <- req
	return <-req.Res
}

func (r *Room) addPlayer(conn *Connection) error {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	if len(r.state.GetPlayers()) >= r.maxPlayers {
		return fmt.Errorf("room is full")
	}
	if err := r.state.AddPlayer(conn.Player); err != nil {
		return fmt.Errorf("tsm.AddPlayer: %w", err)
	}

	r.connections[conn.ID] = conn
	go r.watchPlayer(conn)
	r.BroadcastUpdate(Update[UpdatePlayersChangedData]{
		Type: UpdateTypePlayersChanged,
		Data: UpdatePlayersChangedData{
			Players: r.getMarshalledPlayers(),
		},
	})
	return nil
}

func (r *Room) IsPlayerInRoom(playerID game.PlayerID) bool {
	_, exists := r.state.GetPlayer(playerID)
	return exists
}

func (r *Room) Start() {
	r.logger.Info("Starting room")
	r.started = true
	defer metrics.IncGamesEnded()
	for {
		select {
		case req := <-r.connectionsChan:
			if r.IsPlayerInRoom(req.Conn.ID) {
				req.Conn.ClearPendingUpdates()
				r.sendRejoinState(req.Conn)
				r.logger.Info(fmt.Sprintf("Player rejoined #%s: %s", r.ID, req.Conn.ID))
				req.Res <- nil
			} else {
				if err := r.addPlayer(req.Conn); err != nil {
					r.logger.Error("Error adding new player to room", "err", err, "player_id", req.Conn.Player.ID)
					req.Res <- err
				} else {
					r.logger.Info(fmt.Sprintf("Player joined #%s: %s", r.ID, req.Conn.ID))
					req.Res <- nil
				}
			}
		case action := <-r.actionsChan:
			r.logger.Info(fmt.Sprintf("Recieved action from %s", action.GetPlayerID()), "action", action)
			r.doAction(action)
		case <-r.Context.Done():
			r.logger.Info("Stopping room")
			r.RWMutex.Lock()
			defer r.RWMutex.Unlock()
			r.close()
			return
		}
	}
}

var ErrNotYourTurn = fmt.Errorf("not your turn")
var ErrActionOnClosedRoom = errors.New("action on closed room")

func (r *Room) doAction(action TypedAction) {
	if r.HasClosed() {
		r.logger.Error(ErrActionOnClosedRoom.Error())
		r.TargetedError(action.GetPlayerID(), ErrActionOnClosedRoom)
		return
	}

	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	switch action.GetType() {
	case ActionStart:
		act, ok := action.(*Action[ActionWithoutData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doStartGame(*act); err != nil {
			r.logger.Warn("error starting game", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
		return
	case ActionFirstPeek:
		act, ok := action.(*Action[ActionWithoutData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doPeekTwo(*act); err != nil {
			r.logger.Warn("error on first peek", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
		return
	}
	if !r.state.Playing() || action.GetPlayerID() != r.state.PlayerToPlay().ID {
		r.logger.Warn(
			fmt.Sprintf("Player %s tried to perform action out of turn", action.GetPlayerID()),
			"player_id", action.GetPlayerID(),
			"action", action)
		r.TargetedError(action.GetPlayerID(), ErrNotYourTurn)
		return
	}

	switch action.GetType() {
	case ActionDraw:
		act, ok := action.(*Action[ActionDrawData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doDraw(*act); err != nil {
			r.logger.Warn("error on draw", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
	case ActionDiscard:
		act, ok := action.(*Action[ActionDiscardData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doDiscard(*act); err != nil {
			r.logger.Warn("error on discard", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
	case ActionCut:
		act, ok := action.(*Action[ActionCutData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doCut(*act); err != nil {
			r.logger.Warn("error on cut", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
	case ActionPeekOwnCard:
		act, ok := action.(*Action[ActionPeekOwnCardData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doEffectPeekOwnCard(*act); err != nil {
			r.logger.Warn("error on peek own", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
		return
	case ActionPeekCartaAjena:
		act, ok := action.(*Action[ActionPeekCartaAjenaData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doEffectPeekCartaAjena(*act); err != nil {
			r.logger.Warn("error on peek carta ajena", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
		return
	case ActionSwapCards:
		act, ok := action.(*Action[ActionSwapCardsData])
		if !ok {
			r.logger.Error("error casting action", "action", act, "player_id", act.GetPlayerID())
			return
		}
		if err := r.doEffectSwapCards(*act); err != nil {
			r.logger.Warn("error on swap cards", "err", err, "player_id", act.GetPlayerID())
			r.TargetedError(act.GetPlayerID(), err)
			return
		}
		return
	default:
		r.logger.Warn("unknown action", "player_id", action.GetPlayerID(), "action", action)
	}
}

// watchPlayer functions as a goroutine that watches for new actions from a given player.
func (r *Room) watchPlayer(conn *Connection) {
	r.logger.Info(fmt.Sprintf("Started watch loop for player '%s' on room '%s'", conn.ID, r.ID))
	for {
		select {
		case action := <-conn.Actions:
			r.actionsChan <- action
		case <-r.Context.Done():
			r.logger.Info(fmt.Sprintf("Stopping watch loop for player '%s' on room '%s'", conn.ID, r.ID))
			return
		}

	}
}
