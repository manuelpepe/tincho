package bots

import (
	"fmt"
	"math/rand"

	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

// BaseStrategy just implements non-op methods for all the Strategy interface.
// It's useful for creating a new strategy by embedding it and overriding only the methods you need.
type BaseStrategy struct{}

func (s *BaseStrategy) PlayersChanged(player *tincho.Connection, data tincho.UpdatePlayersChangedData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) PlayerFirstPeeked(player *tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) PeekCard(player *tincho.Connection, data tincho.UpdatePeekCardData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) SwapCards(player *tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) Discard(player *tincho.Connection, data tincho.UpdateDiscardData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) FailedDoubleDiscard(player *tincho.Connection, data tincho.UpdateTypeFailedDoubleDiscardData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) Cut(player *tincho.Connection, data tincho.UpdateCutData) (tincho.TypedAction, error) {
	return nil, nil
}

func (s *BaseStrategy) Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.TypedAction, error) {
	return nil, fmt.Errorf("recieved error update: %s", data.Message)
}

func (s *BaseStrategy) EndGame(player *tincho.Connection, data tincho.UpdateEndGameData) (tincho.TypedAction, error) {
	return nil, nil
}

type KnownHand game.Hand

func (h *KnownHand) Remove(pos int) {
	if h == nil {
		panic("discarding from empty hand") // TODO: maybe change to error instead of panic
	}
	*h = append((*h)[:pos], (*h)[pos+1:]...)
}

func (h *KnownHand) Forget(pos int) error {
	return h.Replace(pos, game.Card{})
}

func (h *KnownHand) Replace(position int, card game.Card) error {
	if h == nil {
		panic("nil KnownHand")
	}
	if position < 0 || position >= len(*h) {
		return fmt.Errorf("invalid position: %d", position)
	}
	(*h)[position] = card
	return nil
}

func (h *KnownHand) KnownPoints() (int, bool) {
	if h == nil {
		panic("nil KnownHand")
	}
	knownHand := make(game.Hand, 0)
	for _, c := range *h {
		if c != (game.Card{}) {
			knownHand = append(knownHand, c)
		}
	}
	return knownHand.Sum(), len(knownHand) == len(*h)
}

func (h *KnownHand) GetUnkownCard() (int, bool) {
	if h == nil {
		panic("nil KnownHand")
	}
	for ix, c := range *h {
		if c == (game.Card{}) {
			return ix, true
		}
	}
	return 0, false
}

func (h *KnownHand) GetHighestValueCard() (int, bool) {
	if h == nil {
		panic("nil KnownHand")
	}
	highestValue := -1
	highestValuePosition := 0
	for ix, c := range *h {
		if c.IsJoker() {
			continue
		}
		if c.IsTwelveOfDiamonds() {
			continue
		}
		if c.Value > highestValue {
			highestValue = c.Value
			highestValuePosition = ix
		}
	}
	if highestValue == -1 {
		return 0, false
	}
	return highestValuePosition, true
}

func (h *KnownHand) GetHighestValueCardOrRandom() int {
	if h == nil {
		panic("nil KnownHand")
	}
	position, ok := h.GetHighestValueCard()
	if ok {
		return position
	}
	return rand.Intn(len(*h))
}
