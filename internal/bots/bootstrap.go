package bots

import "github.com/manuelpepe/tincho/internal/tincho"

// BaseStrategy just implements non-op methods for all the Strategy interface.
// It's useful for creating a new strategy by embedding it and overriding only the methods you need.
type BaseStrategy struct{}

func (s *BaseStrategy) PlayersChanged(player tincho.Player, data tincho.UpdatePlayersChangedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) GameStart(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) PlayerFirstPeeked(player tincho.Player, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Turn(player tincho.Player, data tincho.UpdateTurnData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Draw(player tincho.Player, data tincho.UpdateDrawData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) PeekCard(player tincho.Player, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) SwapCards(player tincho.Player, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Discard(player tincho.Player, data tincho.UpdateDiscardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) FailedDoubleDiscard(player tincho.Player) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Cut(player tincho.Player, data tincho.UpdateCutData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) Error(player tincho.Player, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) StartNextRound(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *BaseStrategy) EndGame(player tincho.Player, data tincho.UpdateEndGameData) (tincho.Action, error) {
	return tincho.Action{}, nil
}
