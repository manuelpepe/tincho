package tincho

import "encoding/json"

type UpdateType string

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

const UpdateTypePlayersChanged UpdateType = "players_changed"

type UpdatePlayersChanged struct {
	Players []Player `json:"players"`
}

const UpdateTypeGameStart UpdateType = "game_start"

type UpdateGameStart struct {
	Players []Player `json:"players"`
}

const UpdateTypePlayerPeeked UpdateType = "player_peeked"

type UpdatePlayerPeekedData struct {
	Player string `json:"player"`
	Cards  []Card `json:"cards"`
}

const UpdateTypeTurn UpdateType = "turn"

type UpdateTurnData struct {
	Player string `json:"player"`
}

const UpdateTypeDraw UpdateType = "draw"

type UpdateDrawData struct {
	Player string     `json:"player"`
	Source DrawSource `json:"source"`
	Card   Card       `json:"card"`
	Effect CardEffect `json:"effect"`
}

const UpdateTypePeekCard UpdateType = "effect_peek"

type UpdatePeekCardData struct {
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
	Player       string `json:"player"`
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
}

const UpdateTypeCut UpdateType = "cut"

type UpdateCutData struct {
	WithCount bool     `json:"withCount"`
	Declared  int      `json:"declared"`
	Player    string   `json:"player"`
	Players   []Player `json:"players"`
}

const UpdateTypeShuffledPiles UpdateType = "shuffled_piles"

const UpdateTypeError UpdateType = "error"

type UpdateErrorData struct {
	Message string `json:"message"`
}

const UpdateTypeEndGame UpdateType = "end_game"
