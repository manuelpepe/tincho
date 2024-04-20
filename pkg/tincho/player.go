package tincho

import (
	"log/slog"

	"github.com/manuelpepe/tincho/pkg/game"
)

type Connection struct {
	*game.Player
	SessionToken string
	Actions      chan Action
	Updates      chan Typed
}

func NewConnection(id game.PlayerID) *Connection {
	return &Connection{
		Player:       game.NewPlayer(id),
		SessionToken: generateRandomString(20),
		Actions:      make(chan Action),
		Updates:      make(chan Typed, 20),
	}
}

func (p *Connection) QueueAction(action Action) {
	action.PlayerID = p.ID
	p.Actions <- action
}

func (p *Connection) SendUpdateOrDrop(update Typed) {
	// TODO: instead of default dropping maybe this could block until player reconnects
	// 	and timeout on room close (important so goroutine it doesn't get stuck forever).
	//  also could kick player of room after a certain timeout.
	select {
	case p.Updates <- update:
	default:
		slog.Error("Dropping update", "player", p.ID, "update", update)
	}
}

func (p *Connection) ClearPendingUpdates() {
loop:
	for {
		select {
		case <-p.Updates:
		default:
			break loop
		}
	}
}
