package bots

import (
	"fmt"
	"math/rand"

	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

type EasyStrategy struct {
	BaseStrategy // embedded to avoid implementing all the methods

	firstTurn bool
}

func NewEasyStrategy() *EasyStrategy {
	return &EasyStrategy{}
}

func (s *EasyStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return s.StartNextRound(player, data)
}

func (s *EasyStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	s.firstTurn = true
	return &tincho.Action[tincho.ActionWithoutData]{Type: tincho.ActionFirstPeek}, nil
}

func (s *EasyStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return nil, nil
	}
	// TODO: prevent cutting in the first N rounds
	triggerCut := rand.Float32() < 0.05
	if triggerCut {
		return &tincho.Action[tincho.ActionCutData]{
			Type: tincho.ActionCut,
			Data: tincho.ActionCutData{
				WithCount: false,
				Declared:  0,
			}}, nil
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

func (s *EasyStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return nil, nil
	}
	return &tincho.Action[tincho.ActionDiscardData]{
		Type: tincho.ActionDiscard,
		Data: tincho.ActionDiscardData{
			CardPosition: rand.Intn(len(player.Hand)),
		},
	}, nil
}

func (s *EasyStrategy) Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.TypedAction, error) {
	return nil, fmt.Errorf("recieved error update: %s", data.Message)
}

func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}
