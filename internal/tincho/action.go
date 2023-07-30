package tincho

import "encoding/json"

type ActionType string

const (
	ActionStart   ActionType = "start"
	ActionDraw    ActionType = "draw"
	ActionDiscard ActionType = "discard"
	ActionCut     ActionType = "cut"
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
