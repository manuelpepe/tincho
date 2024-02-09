package tincho

import "encoding/json"

type UpdateType string

const (
	UpdateTypePlayersChanged      UpdateType = "players_changed"
	UpdateTypeGameStart           UpdateType = "game_start"
	UpdateTypePlayerPeeked        UpdateType = "player_peeked"
	UpdateTypeTurn                UpdateType = "turn"
	UpdateTypeDraw                UpdateType = "draw"
	UpdateTypePeekCard            UpdateType = "effect_peek"
	UpdateTypeSwapCards           UpdateType = "effect_swap"
	UpdateTypeDiscard             UpdateType = "discard"
	UpdateTypeFailedDoubleDiscard UpdateType = "failed_double_discard"
	UpdateTypeCut                 UpdateType = "cut"
	UpdateTypeShuffledPiles       UpdateType = "shuffled_piles"
	UpdateTypeError               UpdateType = "error"
	UpdateTypeEndGame             UpdateType = "end_game"
)

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

type UpdatePlayersChanged struct {
	Players []Player `json:"players"`
}

type UpdateGameStart struct {
	Players []Player `json:"players"`
}

type UpdatePlayerPeekedData struct {
	Player string `json:"player"`
	Cards  []Card `json:"cards"`
}

type UpdateTurnData struct {
	Player string `json:"player"`
}

type UpdateDrawData struct {
	Player string     `json:"player"`
	Source DrawSource `json:"source"`
	Card   Card       `json:"card"`
	Effect CardEffect `json:"effect"`
}

type UpdatePeekCardData struct {
	CardPosition int    `json:"cardPosition"`
	Card         Card   `json:"card"`
	Player       string `json:"player"`
}

type UpdateSwapCardsData struct {
	CardPositions []int    `json:"cardPositions"`
	Players       []string `json:"players"`
}

type UpdateDiscardData struct {
	Player         string `json:"player"`
	CardsPositions []int  `json:"cardPosition"`
	Cards          []Card `json:"card"`
}

type UpdateTypeFailedDoubleDiscardData struct {
	Player         string `json:"player"`
	CardsPositions []int  `json:"cardPosition"`
	Cards          []Card `json:"card"`
}

type UpdateCutData struct {
	WithCount bool     `json:"withCount"`
	Declared  int      `json:"declared"`
	Player    string   `json:"player"`
	Players   []Player `json:"players"`
}

type UpdateErrorData struct {
	Message string `json:"message"`
}

type UpdateEndGameData struct {
	Winner string `json:"winner"`
}
