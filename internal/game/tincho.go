package game

import (
	"errors"
	"slices"
)

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

const STARTING_HAND_SIZE = 4

var ErrPendingDiscard = errors.New("someone needs to discard first")
var ErrPlayerNotPendingFirstPeek = errors.New("player not pending first peek")
var ErrPlayerAlreadyInRoom = errors.New("player already in room")
var ErrGameAlreadyStarted = errors.New("game already started")
var ErrNoWinner = errors.New("no winner")

type Round struct {
	Cutter PlayerID `json:"cutter"`

	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`

	Scores map[PlayerID]int  `json:"scores"`
	Hands  map[PlayerID]Hand `json:"hands"`
}

type Tincho struct {
	players      []*Player
	playing      bool
	currentTurn  int
	drawPile     Deck
	discardPile  Deck
	cpyDeck      Deck
	totalTurns   int
	totalRounds  int
	roundHistory []Round

	// the last card drawn that has not been stored into a player's hand
	pendingStorage Card
}

func NewTinchoWithDeck(deck Deck) *Tincho {
	return &Tincho{
		players:      make([]*Player, 0),
		playing:      false,
		drawPile:     deck,
		discardPile:  make(Deck, 0),
		cpyDeck:      slices.Clone(deck),
		totalTurns:   0,
		totalRounds:  0,
		roundHistory: make([]Round, 0),
	}
}

func (t *Tincho) Winner() (*Player, error) {
	if !t.IsWinConditionMet() {
		return nil, ErrNoWinner
	}
	winner := &Player{Points: 9999}
	for _, p := range t.players {
		if p.Points < winner.Points {
			winner = p
		}
	}
	return winner, nil
}

func (t *Tincho) TotalTurns() int {
	return t.totalTurns
}

func (t *Tincho) TotalRounds() int {
	return t.totalRounds
}

func (t *Tincho) LastDiscarded() Card {
	if len(t.discardPile) == 0 {
		return Card{}
	}
	return t.discardPile[0]

}

func (t *Tincho) CountBaseDeck() int {
	return len(t.cpyDeck)
}

func (t *Tincho) CountDiscardPile() int {
	return len(t.discardPile)
}

func (t *Tincho) CountDrawPile() int {
	return len(t.drawPile)
}

func (t *Tincho) GetPendingStorage() Card {
	return t.pendingStorage
}

// Playing returns whether the game has started or not. The game starts after all players complete their first peek.
func (t *Tincho) Playing() bool {
	return t.playing
}

func (t *Tincho) PlayerToPlay() *Player {
	return t.players[t.currentTurn]
}

func (t *Tincho) passTurn() {
	t.currentTurn = (t.currentTurn + 1) % len(t.players)
	t.totalTurns += 1
}

func (t *Tincho) GetPlayers() []*Player {
	return t.players
}

func (t *Tincho) GetPlayer(playerID PlayerID) (*Player, bool) {
	for _, player := range t.players {
		if player.ID == playerID {
			return player, true
		}
	}
	return nil, false
}

func (t *Tincho) AddPlayer(p *Player) error {
	if t.playing {
		return ErrGameAlreadyStarted
	}
	if _, exists := t.GetPlayer(p.ID); exists {
		return ErrPlayerAlreadyInRoom
	}
	t.players = append(t.players, p)
	return nil
}

func (t *Tincho) IsWinConditionMet() bool {
	for _, p := range t.players {
		if p.Points > 100 {
			return true
		}
	}
	return false
}
