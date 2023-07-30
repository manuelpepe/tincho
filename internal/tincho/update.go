package tincho

import "encoding/json"

type UpdateType string

const (
	UpdatePlayerJoined       UpdateType = "player_joined"
	UpdatePlayerLeft         UpdateType = "player_left"
	UpdateTypeStart          UpdateType = "start"
	UpdateTypeDraw           UpdateType = "draw"
	UpdateTypePeekOwnCard    UpdateType = "effect_peek_own"
	UpdateTypePeekCartaAjena UpdateType = "effect_peek_carta_ajena"
	UpdateTypeSwapCards      UpdateType = "effect_swap_card"
	UpdateTypeDiscard        UpdateType = "discard"
	UpdateTypeCut            UpdateType = "cut"
	UpdateShuffledPiles      UpdateType = "shuffled_piles"
)

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

type UpdatePlayerJoinedData struct {
	Player Player `json:"player"`
}

type UpdatePlayerLeftData struct {
	Player Player `json:"player"`
}

type UpdateStartData struct{}

type UpdateDrawData struct {
	Source DrawSource `json:"source"`
	Card   Card       `json:"card"`
}

type UpdatePeekOwnCardData struct {
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
	Player       string `json:"player"`
}

type UpdatePeekCartaAjenaData struct {
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
	Player       string `json:"player"`
}

type UpdateSwapCardsData struct {
	CardPositions []int    `json:"cardPositions"`
	Players       []string `json:"players"`
}

type UpdateDiscardData struct {
	CardPosition int  `json:"cardPosition"`
	Card         Card `json:"card"`
}

type UpdateCutData struct {
	WithCount bool   `json:"withCount"`
	Declared  int    `json:"declared"`
	Success   bool   `json:"success"`
	Player    string `json:"player"`
}
