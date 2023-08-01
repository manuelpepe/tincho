package tincho

import "encoding/json"

type UpdateType string

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

const UpdateTypePlayersChanged UpdateType = "players_changed"

type UpdatePlayersChangedData struct {
	Players []Player `json:"players"`
}

const UpdateTypeStartRound UpdateType = "start_round"

type UpdateStartRoundData struct {
	Players []Player `json:"players"`
}

const UpdateTypeDraw UpdateType = "draw"

type UpdateDrawData struct {
	Source DrawSource `json:"source"`
	Card   Card       `json:"card"`
}

const UpdateTypePeekOwnCard UpdateType = "effect_peek_own"

type UpdatePeekOwnCardData struct {
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
	Player       string `json:"player"`
}

const UpdateTypePeekCartaAjena UpdateType = "effect_peek_carta_ajena"

type UpdatePeekCartaAjenaData struct {
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
	Player       string `json:"player"`
}

const UpdateTypeSwapCards UpdateType = "effect_swap_card"

type UpdateSwapCardsData struct {
	CardPositions []int    `json:"cardPositions"`
	Players       []string `json:"players"`
}

const UpdateTypeDiscard UpdateType = "discard"

type UpdateDiscardData struct {
	CardPosition int  `json:"cardPosition"`
	Card         Card `json:"card"`
}

const UpdateTypeCut UpdateType = "cut"

type UpdateCutData struct {
	WithCount bool   `json:"withCount"`
	Declared  int    `json:"declared"`
	Success   bool   `json:"success"`
	Player    string `json:"player"`
}

const UpdateShuffledPiles UpdateType = "shuffled_piles"
