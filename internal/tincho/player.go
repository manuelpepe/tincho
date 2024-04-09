package tincho

import (
	"encoding/json"
	"log/slog"
)

type PlayerID string

// marshalledPlayer is a struct used to marshal a Player into JSON.
type marshalledPlayer struct {
	ID               PlayerID `json:"id"`
	Points           int      `json:"points"`
	PendingFirstPeek bool     `json:"pending_first_peek"`
	CardsInHand      int      `json:"cards_in_hand"`
}

// Player is the internal representation of a player in a game.
// Currently holds both game state and connection state,
// but this should probably be split.
type Player struct {
	ID               PlayerID
	SessionToken     string // this probably belongs somewhere else, like the room or service layers
	Points           int
	PendingFirstPeek bool
	Hand             Hand
	Actions          chan Action
	Updates          chan Update
}

func NewPlayer(id PlayerID) *Player {
	return &Player{
		ID:           id,
		SessionToken: generateRandomString(20),
		Hand:         make(Hand, 0),
		Actions:      make(chan Action),
		Updates:      make(chan Update, 10),
		Points:       0,
	}
}

// MarshalJSON implements json.Marshaller
func (p *Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(marshalledPlayer{
		ID:               p.ID,
		Points:           p.Points,
		PendingFirstPeek: p.PendingFirstPeek,
		CardsInHand:      len(p.Hand),
	})
}

// MarshalJSON implements json.Unmarshaller
// The marshalled player doesn't contain the values for the cards in hand, so
// an empty hand is created for the player instead.
// The core app wouldn't normally unmarshall a player struct, so this is mostly
// implemented for bots and testing.
func (p *Player) UnmarshalJSON(data []byte) error {
	var mp marshalledPlayer
	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}
	p.ID = mp.ID
	p.PendingFirstPeek = mp.PendingFirstPeek
	p.Points = mp.Points
	p.Hand = make(Hand, mp.CardsInHand)
	return nil
}

func (p *Player) QueueAction(action Action) {
	action.PlayerID = p.ID
	p.Actions <- action
}

func (p *Player) SendUpdateOrDrop(update Update) {
	select {
	case p.Updates <- update:
	default:
		slog.Error("Dropping update", "player", p.ID, "update", update)
	}
}

func (p *Player) ClearPendingUpdates() {
loop:
	for {
		select {
		case <-p.Updates:
		default:
			break loop
		}
	}
}
