package tincho

import (
	"encoding/json"
	"fmt"
)

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
	Type   ActionType      `json:"type"`
	Data   json.RawMessage `json:"data"`
	Player string
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

func (r *Room) PassTurn() {}

func (r *Room) doStartGame(action Action) error {
	r.Playing = true
	return nil
}

func (r *Room) doDraw(action Action) error {
	var data DrawAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	r.DrawCard(data.Source)
	r.PassTurn()
	return nil
}

func (r *Room) DrawCard(source DrawSource) {}

func (r *Room) doDiscard(action Action) error {
	var data DiscardAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if err := r.DiscardCard(data.Card); err != nil {
		return fmt.Errorf("DiscardCard: %w", err)
	}
	r.PassTurn()
	return nil
}

func (r *Room) DiscardCard(card Card) error { return nil }

func (r *Room) doCut(action Action) error {
	var data CutAction
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	r.Cut(data.WithCount, data.Declared)
	r.PassTurn()
	return nil
}

func (r *Room) Cut(withCount bool, decaled int) {}
