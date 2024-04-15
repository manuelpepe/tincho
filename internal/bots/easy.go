package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/manuelpepe/tincho/internal/game"
	"github.com/manuelpepe/tincho/internal/tincho"
)

type EasyStrategy struct {
	BaseStrategy // embedded to avoid implementing all the methods

	firstTurn bool
}

func NewEasyStrategy() *EasyStrategy {
	return &EasyStrategy{}
}

func (s *EasyStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *EasyStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	s.firstTurn = true
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *EasyStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	// TODO: prevent cutting in the first N rounds
	triggerCut := rand.Float32() < 0.05
	if triggerCut {
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
		data, err := json.Marshal(
			tincho.ActionDrawData{Source: RandChoice(choices)},
		)
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		s.firstTurn = false
		return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
	}
}

func (s *EasyStrategy) Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	res, err := json.Marshal(tincho.ActionDiscardData{
		CardPosition: rand.Intn(len(player.Hand)),
	})
	if err != nil {
		return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
	}
	return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(res)}, nil
}

func (s *EasyStrategy) Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, fmt.Errorf("recieved error update: %s", data.Message)
}

func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}
