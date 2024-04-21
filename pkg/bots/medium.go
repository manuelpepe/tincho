package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"

	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

type MediumStrategy struct {
	BaseStrategy // embedded to avoid implementing all the methods

	hand      KnownHand
	firstTurn bool
}

func NewMediumStrategy() *MediumStrategy {
	return &MediumStrategy{}
}

func (s *MediumStrategy) ResetHand(self *tincho.Connection, players []tincho.MarshalledPlayer) {
	for _, p := range players {
		if p.ID == self.ID {
			s.hand = make(KnownHand, p.CardsInHand)
			return
		}
	}
}

func (s *MediumStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.ResetHand(player, data.Players)
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	s.ResetHand(player, data.Players)
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) PlayerFirstPeeked(player *tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	if data.Player == player.ID {
		s.hand.Replace(0, data.Cards[0])
		s.hand.Replace(1, data.Cards[1])
	}
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error) {
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
		var choices []game.DrawSource
		if s.firstTurn {
			choices = []game.DrawSource{game.DrawSourcePile}
		} else {
			choices = []game.DrawSource{game.DrawSourcePile, game.DrawSourceDiscard}
		}
		data, err := json.Marshal(tincho.ActionDrawData{Source: RandChoice(choices)})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.firstTurn = false
		return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
	}
}

func (s *MediumStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	unkownCard, hasUnkownCard := s.hand.GetUnkownCard()
	if hasUnkownCard {
		if data.Source == game.DrawSourcePile && data.Card.GetEffect() == game.CardEffectPeekOwnCard {
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

func (s *MediumStrategy) PeekCard(player *tincho.Connection, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	s.hand.Replace(data.CardPosition, data.Card)
	return tincho.Action{}, nil
}

func (s *MediumStrategy) SwapCards(player *tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
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

func (s *MediumStrategy) Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, fmt.Errorf("recieved error update: %s", data.Message)
}
