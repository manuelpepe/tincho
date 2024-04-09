package tincho

import (
	"encoding/json"

	"github.com/manuelpepe/tincho/internal/game"
)

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
	UpdateTypeRejoin              UpdateType = "rejoin_state"
)

type Update struct {
	Type UpdateType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

type UpdatePlayersChangedData struct {
	Players []*game.Player `json:"players"`
}

type UpdateStartNextRoundData struct {
	Players    []*game.Player `json:"players"`
	TopDiscard game.Card      `json:"topDiscard"`
}

type UpdatePlayerFirstPeekedData struct {
	Player game.PlayerID `json:"player"`
	Cards  []game.Card   `json:"cards"`
}

type UpdateTurnData struct {
	Player game.PlayerID `json:"player"`
}

type UpdateDrawData struct {
	Player game.PlayerID   `json:"player"`
	Source game.DrawSource `json:"source"`
	Card   game.Card       `json:"card"`
	Effect game.CardEffect `json:"effect"`
}

type UpdatePeekCardData struct {
	CardPosition int           `json:"cardPosition"`
	Card         game.Card     `json:"card"`
	Player       game.PlayerID `json:"player"`
}

type UpdateSwapCardsData struct {
	CardsPositions []int           `json:"cardsPositions"`
	Players        []game.PlayerID `json:"players"`
}

type UpdateDiscardData struct {
	Player         game.PlayerID `json:"player"`
	CardsPositions []int         `json:"cardsPositions"`
	Cards          []game.Card   `json:"cards"`
}

type UpdateTypeFailedDoubleDiscardData struct {
	Player         game.PlayerID `json:"player"`
	CardsPositions []int         `json:"cardsPositions"`
	Cards          []game.Card   `json:"cards"`
	TopOfDiscard   game.Card     `json:"topOfDiscard"`
}

type UpdateCutData struct {
	WithCount bool           `json:"withCount"`
	Declared  int            `json:"declared"`
	Player    game.PlayerID  `json:"player"`
	Players   []*game.Player `json:"players"`
	Hands     [][]game.Card  `json:"hands"`
}

type UpdateErrorData struct {
	Message string `json:"message"`
}

type UpdateEndGameData struct {
	Rounds []game.Round `json:"rounds"`
}

type UpdateTypeRejoinData struct {
	Players       []*game.Player `json:"players"`
	CurrentTurn   game.PlayerID  `json:"currentTurn"`
	CardInHand    bool           `json:"cardInHand"`
	CardInHandVal *game.Card     `json:"cardInHandValue"`
	LastDiscarded *game.Card     `json:"lastDiscarded"`
}
