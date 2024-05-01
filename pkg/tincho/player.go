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
	Actions      chan TypedAction
	Updates      chan TypedUpdate
}

func NewConnection(id game.PlayerID) *Connection {
	return &Connection{
		Player:       game.NewPlayer(id),
		SessionToken: generateRandomString(20),
		Actions:      make(chan TypedAction),
		Updates:      make(chan TypedUpdate, 20),
	}
}

func (c *Connection) QueueAction(action TypedAction) {
	action.SetPlayerID(c.ID)
	c.Actions <- action
}

func (c *Connection) SendUpdateOrDrop(update TypedUpdate) {
	// TODO: instead of default dropping maybe this could block until player reconnects
	// 	and timeout on room close (important so goroutine it doesn't get stuck forever).
	//  also could kick player of room after a certain timeout.
	select {
	case c.Updates <- update:
	default:
		slog.Error("Dropping update", "player", c.ID, "update", update)
	}
}

func (c *Connection) ClearPendingUpdates() {
loop:
	for {
		select {
		case <-c.Updates:
		default:
			break loop
		}
	}
}
