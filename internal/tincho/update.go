package tincho

import "encoding/json"

type UpdateType string

const (
	UpdateTypePlayersChanged      UpdateType = "players_changed"
	UpdateTypeGameStart           UpdateType = "game_start"
	UpdateTypePlayerFirstPeeked   UpdateType = "player_peeked"
	UpdateTypeTurn                UpdateType = "turn"
	UpdateTypeDraw                UpdateType = "draw"
	UpdateTypePeekCard            UpdateType = "effect_peek"
	UpdateTypeSwapCards           UpdateType = "effect_swap"
	UpdateTypeDiscard             UpdateType = "discard"
	UpdateTypeFailedDoubleDiscard UpdateType = "failed_double_discard"
	UpdateTypeCut                 UpdateType = "cut"
	UpdateTypeError               UpdateType = "error"
	UpdateTypeStartNextRound      UpdateType = "start_next_round"
	UpdateTypeEndGame             UpdateType = "end_game"
)

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

type UpdatePlayersChangedData struct {
	Players []*Player `json:"players"`
}

type UpdateStartNextRoundData struct {
	Players []*Player `json:"players"`
}

type UpdatePlayerFirstPeekedData struct {
	Player PlayerID `json:"player"`
	Cards  []Card   `json:"cards"`
}

type UpdateTurnData struct {
	Player PlayerID `json:"player"`
}

type UpdateDrawData struct {
	Player PlayerID   `json:"player"`
	Source DrawSource `json:"source"`
	Card   Card       `json:"card"`
	Effect CardEffect `json:"effect"`
}

type UpdatePeekCardData struct {
	CardPosition int      `json:"cardPosition"`
	Card         Card     `json:"card"`
	Player       PlayerID `json:"player"`
}

type UpdateSwapCardsData struct {
	CardsPositions []int      `json:"cardsPositions"`
	Players        []PlayerID `json:"players"`
}

type UpdateDiscardData struct {
	Player         PlayerID `json:"player"`
	CardsPositions []int    `json:"cardsPositions"`
	Cards          []Card   `json:"cards"`
}

type UpdateTypeFailedDoubleDiscardData struct {
	Player         PlayerID `json:"player"`
	CardsPositions []int    `json:"cardsPositions"`
	Cards          []Card   `json:"cards"`
}

type UpdateCutData struct {
	WithCount bool      `json:"withCount"`
	Declared  int       `json:"declared"`
	Player    PlayerID  `json:"player"`
	Players   []*Player `json:"players"`
	Hands     [][]Card  `json:"hands"`
}

type UpdateErrorData struct {
	Message string `json:"message"`
}

type UpdateEndGameData struct {
	Scores [][]PlayerScore `json:"scores"`
}
