package bots

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type MediumStrategy struct {
}

func (s *MediumStrategy) PlayersChanged(player tincho.Player, data tincho.UpdatePlayersChangedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) GameStart(player tincho.Player) (tincho.Action, error) {
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) PlayerFirstPeeked(player tincho.Player, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Turn(player tincho.Player, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player == player.ID {
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
				Source: RandChoice([]tincho.DrawSource{tincho.DrawSourcePile, tincho.DrawSourceDiscard}),
			})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			return tincho.Action{Type: tincho.ActionDraw, Data: data}, nil
		}
	}
	return tincho.Action{}, nil
}

func (s *MediumStrategy) Draw(player tincho.Player, data tincho.UpdateDrawData) (tincho.Action, error) {
	if data.Player == player.ID {
		data, err := json.Marshal(tincho.ActionDiscardData{
			CardPosition: rand.Intn(len(player.Hand)),
		})
		if err != nil {
			return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
		}
		return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(data)}, nil
	}
	return tincho.Action{}, nil
}

func (s *MediumStrategy) PeekCard(player tincho.Player, data tincho.UpdatePeekCardData) (tincho.Action, error) {
	return tincho.Action{}, nil
}

func (s *MediumStrategy) SwapCards(player tincho.Player, data tincho.UpdateSwapCardsData) (tincho.Action, error) {
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
	return tincho.Action{Type: tincho.ActionFirstPeek}, nil
}

func (s *MediumStrategy) EndGame(player tincho.Player, data tincho.UpdateEndGameData) (tincho.Action, error) {
	return tincho.Action{}, nil
}
