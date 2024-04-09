package bots

import "github.com/manuelpepe/tincho/internal/tincho"

// BaseStrategy just implements non-op methods for all the Strategy interface.
// It's useful for creating a new strategy by embedding it and overriding only the methods you need.
type BaseStrategy struct{}

func (s *BaseStrategy) PlayersChanged(player tincho.Connection, data tincho.UpdatePlayersChangedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) GameStart(player tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) PlayerFirstPeeked(player tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Turn(player tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Draw(player tincho.Connection, data tincho.UpdateDrawData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) PeekCard(player tincho.Connection, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) SwapCards(player tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Discard(player tincho.Connection, data tincho.UpdateDiscardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) FailedDoubleDiscard(player tincho.Connection) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Cut(player tincho.Connection, data tincho.UpdateCutData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Error(player tincho.Connection, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) StartNextRound(player tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) EndGame(player tincho.Connection, data tincho.UpdateEndGameData) (tincho.Action, error) {
	return tincho.Action{}, nil
}
