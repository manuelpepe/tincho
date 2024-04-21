package game

import "encoding/json"

type PlayerID string

// Player is the internal representation of a player in a game.
// Currently holds both game state and connection state,
// but this should probably be split.
type Player struct {
	ID               PlayerID
	Points           int
	PendingFirstPeek bool
	Hand             Hand
}

func NewPlayer(id PlayerID) *Player {
	return &Player{
		ID:     id,
		Hand:   make(Hand, 0),
		Points: 0,
	}
}

// MarshalledPlayer is a struct used to marshal a Player into JSON.
type MarshalledPlayer struct {
	ID               PlayerID `json:"id"`
	Points           int      `json:"points"`
	PendingFirstPeek bool     `json:"pending_first_peek"`
	CardsInHand      int      `json:"cards_in_hand"`
}

func (p *Player) Marshalled() MarshalledPlayer {
	return MarshalledPlayer{
		ID:               p.ID,
		Points:           p.Points,
		PendingFirstPeek: p.PendingFirstPeek,
		CardsInHand:      len(p.Hand),
	}
}

// MarshalJSON implements json.Marshaller
func (p *Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(MarshalledPlayer{
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
	var mp MarshalledPlayer
	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}
	p.ID = mp.ID
	p.PendingFirstPeek = mp.PendingFirstPeek
	p.Points = mp.Points
	p.Hand = make(Hand, mp.CardsInHand)
	return nil
}
