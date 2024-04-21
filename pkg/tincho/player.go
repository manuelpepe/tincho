package tincho

import (
	"log/slog"

	"github.com/manuelpepe/tincho/pkg/game"
)

// MarshalledPlayer is a struct used to marshal a Player into JSON.
type MarshalledPlayer struct {
	ID               game.PlayerID `json:"id"`
	Points           int           `json:"points"`
	PendingFirstPeek bool          `json:"pending_first_peek"`
	CardsInHand      int           `json:"cards_in_hand"`
}

func NewMarshalledPlayer(p *game.Player) MarshalledPlayer {
	return MarshalledPlayer{
		ID:               p.ID,
		Points:           p.Points,
		PendingFirstPeek: p.PendingFirstPeek,
		CardsInHand:      len(p.Hand),
	}
}

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
