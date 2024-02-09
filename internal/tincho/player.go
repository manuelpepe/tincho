package tincho

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

// marshalledPlayer is a struct used to marshal a Player into JSON.
type marshalledPlayer struct {
	ID               string `json:"id"`
	Points           int    `json:"points"`
	PendingFirstPeek bool   `json:"pending_first_peek"`
	CardsInHand      int    `json:"cards_in_hand"`
}

// Player is the internal representation of a player in a game.
// Currently holds both game state and connection state,
// but this should probably be split.
type Player struct {
	ID               string
	Points           int
	PendingFirstPeek bool
	Hand             Hand
	socket           *websocket.Conn
	Updates          chan Update
}

func (p *Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(marshalledPlayer{
		ID:               p.ID,
		Points:           p.Points,
		PendingFirstPeek: p.PendingFirstPeek,
		CardsInHand:      len(p.Hand),
	})
}

func (p *Player) UnmarshalJSON(data []byte) error {
	var mp marshalledPlayer
	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}
	p.ID = mp.ID
	p.PendingFirstPeek = mp.PendingFirstPeek
	p.Points = mp.Points
	return nil
}

func NewPlayer(id string, socket *websocket.Conn) Player {
	return Player{
		ID:      id,
		Hand:    make(Hand, 0),
		socket:  socket,
		Updates: make(chan Update),
		Points:  0,
	}
}
