package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type EasyStrategy struct{}

func (s EasyStrategy) PlayersChanged(player tincho.Player, data tincho.UpdatePlayersChangedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) GameStart(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s EasyStrategy) PlayerFirstPeeked(player tincho.Player, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) Turn(player tincho.Player, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
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
		data, err := json.Marshal(tincho.ActionDrawData{
			Source: RandChoice([]tincho.DrawSource{
				tincho.DrawSourcePile,
				tincho.DrawSourceDiscard,
			}),
		})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
	}
}

func (s EasyStrategy) Draw(player tincho.Player, data tincho.UpdateDrawData) (tincho.Action, error) {
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

func (s EasyStrategy) PeekCard(player tincho.Player, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) SwapCards(player tincho.Player, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) Discard(player tincho.Player, data tincho.UpdateDiscardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) FailedDoubleDiscard(player tincho.Player) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) Cut(player tincho.Player, data tincho.UpdateCutData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s EasyStrategy) Error(player tincho.Player, data tincho.UpdateErrorData) (tincho.Action, error) {
	return tincho.Action{}, fmt.Errorf("recieved error update: %s", data.Message)
}

func (s EasyStrategy) StartNextRound(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error) {
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s EasyStrategy) EndGame(player tincho.Player, data tincho.UpdateEndGameData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}
