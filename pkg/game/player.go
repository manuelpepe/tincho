package game

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
