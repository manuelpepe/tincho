package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type KnownHand tincho.Hand

func (h *KnownHand) Forget(pos int) error {
	return h.Replace(pos, tincho.Card{})
}

func (h *KnownHand) Replace(position int, card tincho.Card) error {
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
	knownHand := make(tincho.Hand, 0)
	for _, c := range *h {
		if c != (tincho.Card{}) {
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
		if c == (tincho.Card{}) {
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

type MediumStrategy struct {
	hand      KnownHand
	firstTurn bool
}

func (s *MediumStrategy) ResetHand(self tincho.Player, players []*tincho.Player) {
	for _, p := range players {
		if p.ID == self.ID {
			s.hand = make(KnownHand, len(p.Hand))
			return
		}
	}
}

func (s *MediumStrategy) PlayersChanged(player tincho.Player, data tincho.UpdatePlayersChangedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) GameStart(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.ResetHand(player, data.Players)
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) PlayerFirstPeeked(player tincho.Player, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	if data.Player == player.ID {
		s.hand.Replace(0, data.Cards[0])
		s.hand.Replace(1, data.Cards[1])
	}
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Turn(player tincho.Player, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	forceCut := rand.Float32() < 0.05
	triggerCut := rand.Float32() < 0.75
	pointsInHand, knowFullHand := s.hand.KnownPoints()
	if forceCut || (knowFullHand && triggerCut && pointsInHand <= 10) {
		data, err := json.Marshal(tincho.ActionCutData{
			WithCount: false,
			Declared:  0,
		})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		return tincho.Action{Type: tincho.ActionCut, Data: data}, nil
	} else {
		var choices []tincho.DrawSource
		if s.firstTurn {
			choices = []tincho.DrawSource{tincho.DrawSourcePile}
		} else {
			choices = []tincho.DrawSource{tincho.DrawSourcePile, tincho.DrawSourceDiscard}
		}
		data, err := json.Marshal(tincho.ActionDrawData{Source: RandChoice(choices)})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.firstTurn = false
		return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
	}
}

func (s *MediumStrategy) Draw(player tincho.Player, data tincho.UpdateDrawData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	unkownCard, hasUnkownCard := s.hand.GetUnkownCard()
	if hasUnkownCard {
		if data.Card.GetEffect() == tincho.CardEffectPeekOwnCard {
			res, err := json.Marshal(tincho.ActionPeekOwnCardData{CardPosition: unkownCard})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			s.hand.Replace(unkownCard, data.Card)
			return tincho.Action{Type: tincho.ActionPeekOwnCard, Data: json.RawMessage(res)}, nil
		} else {
			res, err := json.Marshal(tincho.ActionDiscardData{CardPosition: unkownCard})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			s.hand.Replace(unkownCard, data.Card)
			return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
		}
	}

	// chance of discarding a random card
	if makesMistake := rand.Float32() < 0.20; makesMistake {
		discardIx := rand.Intn(len(s.hand))
		res, err := json.Marshal(tincho.ActionDiscardData{CardPosition: discardIx})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.hand.Replace(discardIx, data.Card)
		return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
	}

	// discard highest value card
	discardIx := s.hand.GetHighestValueCardOrRandom()
	res, err := json.Marshal(tincho.ActionDiscardData{CardPosition: discardIx})
	if err != nil {
		return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
	}
	s.hand.Replace(discardIx, data.Card)
	return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
}

func (s *MediumStrategy) PeekCard(player tincho.Player, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	s.hand.Replace(data.CardPosition, data.Card)
	return tincho.Action{}, nil
}

func (s *MediumStrategy) SwapCards(player tincho.Player, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
	myIX := slices.Index(data.Players, player.ID)
	if myIX == -1 {
		return tincho.Action{}, nil
	}
	cardPos := data.CardsPositions[myIX]
	if err := s.hand.Forget(cardPos); err != nil {
		return tincho.Action{}, fmt.Errorf("s.hand.Forget: %w", err)
	}
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Discard(player tincho.Player, data tincho.UpdateDiscardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) FailedDoubleDiscard(player tincho.Player) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Cut(player tincho.Player, data tincho.UpdateCutData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Error(player tincho.Player, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, fmt.Errorf("recieved error update: %s", data.Message)
}

func (s *MediumStrategy) StartNextRound(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.ResetHand(player, data.Players)
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) EndGame(player tincho.Player, data tincho.UpdateEndGameData) (tincho.Action, error) {
	return tincho.Action{}, nil
}
