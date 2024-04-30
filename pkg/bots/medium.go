package bots

import (
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

func (s *MediumStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return s.StartNextRound(player, data)
}

func (s *MediumStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	s.firstTurn = true
	s.ResetHand(player, data.Players)
	return &tincho.Action[tincho.ActionWithoutData]{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) PlayerFirstPeeked(player *tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.TypedAction, error) {
	if data.Player == player.ID {
		s.hand.Replace(0, data.Cards[0])
		s.hand.Replace(1, data.Cards[1])
	}
	return nil, nil
}

func (s *MediumStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return nil, nil
	}

	forceCut := rand.Float32() < 0.05
	triggerCut := rand.Float32() < 0.75
	pointsInHand, knowFullHand := s.hand.KnownPoints()
	if forceCut || (knowFullHand && triggerCut && pointsInHand <= 10) {
		return &tincho.Action[tincho.ActionCutData]{
			Type: tincho.ActionCut,
			Data: tincho.ActionCutData{
				WithCount: false,
				Declared:  0,
			},
		}, nil
	} else {
		choices := []game.DrawSource{game.DrawSourcePile, game.DrawSourceDiscard}
		if s.firstTurn {
			choices = []game.DrawSource{game.DrawSourcePile}
			s.firstTurn = false
		}
		return &tincho.Action[tincho.ActionDrawData]{
			Type: tincho.ActionDraw,
			Data: tincho.ActionDrawData{Source: RandChoice(choices)},
		}, nil
	}
}

func (s *MediumStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return nil, nil
	}
	unkownCard, hasUnkownCard := s.hand.GetUnkownCard()
	if hasUnkownCard {
		if data.Source == game.DrawSourcePile && data.Card.GetEffect() == game.CardEffectPeekOwnCard {
			s.hand.Replace(unkownCard, data.Card)
			return &tincho.Action[tincho.ActionPeekOwnCardData]{
				Type: tincho.ActionPeekOwnCard,
				Data: tincho.ActionPeekOwnCardData{CardPosition: unkownCard},
			}, nil
		} else {
			s.hand.Replace(unkownCard, data.Card)
			return &tincho.Action[tincho.ActionDiscardData]{
				Type: tincho.ActionDiscard,
				Data: tincho.ActionDiscardData{CardPosition: unkownCard},
			}, nil
		}
	}

	// chance of discarding a random card
	if makesMistake := rand.Float32() < 0.20; makesMistake {
		discardIx := rand.Intn(len(s.hand))
		s.hand.Replace(discardIx, data.Card)
		return &tincho.Action[tincho.ActionDiscardData]{
			Type: tincho.ActionDiscard,
			Data: tincho.ActionDiscardData{CardPosition: discardIx},
		}, nil
	}

	// discard highest value card
	discardIx := s.hand.GetHighestValueCardOrRandom()
	s.hand.Replace(discardIx, data.Card)
	return &tincho.Action[tincho.ActionDiscardData]{
		Type: tincho.ActionDiscard,
		Data: tincho.ActionDiscardData{CardPosition: discardIx},
	}, nil
}

func (s *MediumStrategy) PeekCard(player *tincho.Connection, data tincho.UpdatePeekCardData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return nil, nil
	}
	s.hand.Replace(data.CardPosition, data.Card)
	return nil, nil
}

func (s *MediumStrategy) SwapCards(player *tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.TypedAction, error) {
	myIX := slices.Index(data.Players, player.ID)
	if myIX == -1 {
		return nil, nil
	}
	cardPos := data.CardsPositions[myIX]
	if err := s.hand.Forget(cardPos); err != nil {
		return nil, fmt.Errorf("s.hand.Forget: %w", err)
	}
	return nil, nil
}

func (s *MediumStrategy) Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.TypedAction, error) {
	return nil, fmt.Errorf("recieved error update: %s", data.Message)
}
