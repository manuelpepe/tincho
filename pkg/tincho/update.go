package tincho

import (
	"github.com/manuelpepe/tincho/pkg/game"
)

type UpdateType string

const (
	UpdateTypeGameConfig          UpdateType = "game_config"
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

type UpdateData interface {
	UpdatePlayersChangedData |
		UpdateGameConfig |
		UpdateStartNextRoundData |
		UpdatePlayerFirstPeekedData |
		UpdateTurnData |
		UpdateDrawData |
		UpdatePeekCardData |
		UpdateSwapCardsData |
		UpdateDiscardData |
		UpdateTypeFailedDoubleDiscardData |
		UpdateCutData |
		UpdateErrorData |
		UpdateEndGameData |
		UpdateTypeRejoinData
}

type Update[T UpdateData] struct {
	Type UpdateType `json:"type"`
	Data T          `json:"data"`
}

type TypedUpdate interface {
	GetType() UpdateType
}

func (u Update[T]) GetType() UpdateType {
	return u.Type
}

type UpdatePlayersChangedData struct {
	Players []MarshalledPlayer `json:"players"`
}

type UpdateGameConfig struct {
	CardsInDeck int `json:"cardsInDeck"`
	// Maybe:
	//  - has password
	//	- deck options
	// 	- room owner
}

type UpdateStartNextRoundData struct {
	Players    []MarshalledPlayer `json:"players"`
	TopDiscard game.Card          `json:"topDiscard"`
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
	Player         game.PlayerID    `json:"player"`
	CardsPositions []int            `json:"cardsPositions"`
	Cards          []game.Card      `json:"cards"`
	CycledPiles    game.CycledPiles `json:"cycledPiles"`
}

type UpdateTypeFailedDoubleDiscardData struct {
	Player         game.PlayerID    `json:"player"`
	CardsPositions []int            `json:"cardsPositions"`
	Cards          []game.Card      `json:"cards"`
	TopOfDiscard   game.Card        `json:"topOfDiscard"`
	CycledPiles    game.CycledPiles `json:"cycledPiles"`
}

type UpdateCutData struct {
	WithCount bool               `json:"withCount"`
	Declared  int                `json:"declared"`
	Player    game.PlayerID      `json:"player"`
	Players   []MarshalledPlayer `json:"players"`
	Hands     [][]game.Card      `json:"hands"`
}

type UpdateErrorData struct {
	Message string `json:"message"`
}

type UpdateEndGameData struct {
	Rounds []game.Round `json:"rounds"`
}

type UpdateTypeRejoinData struct {
	Players          []MarshalledPlayer `json:"players"`
	CurrentTurn      game.PlayerID      `json:"currentTurn"`
	CardInHand       bool               `json:"cardInHand"`
	CardInHandVal    *game.Card         `json:"cardInHandValue"`
	CardInHandSource *game.DrawSource   `json:"cardInHandSource"`
	LastDiscarded    *game.Card         `json:"lastDiscarded"`
	CardsInDeck      int                `json:"cardsInDeck"`
	CardsInDrawPile  int                `json:"cardsInDrawPile"`
}
