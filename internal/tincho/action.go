package tincho

import "encoding/json"

type ActionType string

const (
	ActionStart          ActionType = "start"
	ActionDraw           ActionType = "draw"
	ActionPeekOwnCard    ActionType = "effect_peek_own"
	ActionPeekCartaAjena ActionType = "effect_peek_carta_ajena"
	ActionSwapCards      ActionType = "effect_swap_card"
	ActionDiscard        ActionType = "discard"
	ActionCut            ActionType = "cut"
)

type Action struct {
	Type ActionType      `json:"type"`
	Data json.RawMessage `json:"payload"`
}

type DrawSource string

const (
	DrawSourcePile    DrawSource = "pile"
	DrawSourceDiscard DrawSource = "discard"
)

type DrawAction struct {
	Source DrawSource `json:"source"`
}

type DiscardAction struct {
	Card Card `json:"card"`
}

type CutAction struct {
	WithCount bool `json:"withCount"`
	Declared  int  `json:"declared"`
}

func (r *Room) StartGame() {}

func (r *Room) DrawCard(source DrawSource) {}

func (r *Room) PassTurn() {}

func (r *Room) DiscardCard(card Card) error { return nil }

func (r *Room) Cut(withCount bool, decaled int) {}
