package tincho

import (
	"log/slog"

	"github.com/manuelpepe/tincho/internal/game"
)

type Connection struct {
	*game.Player
	SessionToken string
	Actions      chan Action
	Updates      chan Update
}

func NewConnection(id game.PlayerID) *Connection {
	return &Connection{
		Player:       game.NewPlayer(id),
		SessionToken: generateRandomString(20),
		Actions:      make(chan Action),
		Updates:      make(chan Update, 10),
	}
}

func (p *Connection) QueueAction(action Action) {
	action.PlayerID = p.ID
	p.Actions <- action
}

func (p *Connection) SendUpdateOrDrop(update Update) {
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
