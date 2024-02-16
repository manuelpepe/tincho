package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type Bot struct {
	ctx      context.Context
	player   *tincho.Player
	strategy Strategy
}

func NewBot(ctx context.Context, player *tincho.Player, difficulty string) (Bot, error) {
	var strategy Strategy
	switch difficulty {
	case "easy":
		strategy = &EasyStrategy{}
	case "medium":
	case "hard":
	case "expert":
	default:
		return Bot{}, fmt.Errorf("invalid difficulty: %s", difficulty)
	}
	return Bot{
		ctx:      ctx,
		player:   player,
		strategy: strategy,
	}, nil

}

func (b *Bot) Start() error {
	for {
		time.Sleep(1 * time.Second)
		select {
		case update := <-b.player.Updates:
			log.Printf("Bot %s recieved update: {Type:%s, Data:\"%s\"}\n", b.player.ID, update.Type, update.Data)
			action, err := b.strategy.RespondToUpdate(*b.player, update)
			if err != nil {
				return fmt.Errorf("error responding to update: %w", err)
			}
			if action.Type != "" {
				log.Printf("Bot %s queued action: {Type:%s, Data:\"%s\"}\n", b.player.ID, action.Type, action.Data)
				b.player.QueueAction(action)
			}
		case <-b.ctx.Done():
			log.Printf("Bot %s finished\n", b.player.ID)
			return nil
		}
	}
}

type Strategy interface {
	RespondToUpdate(tincho.Player, tincho.Update) (tincho.Action, error)
}

type EasyStrategy struct{}

func (s EasyStrategy) RespondToUpdate(player tincho.Player, update tincho.Update) (tincho.Action, error) {
	switch update.Type {
	case tincho.UpdateTypeGameStart:
		return tincho.Action{Type: tincho.ActionFirstPeek}, nil
	case tincho.UpdateTypeTurn:
		var data tincho.UpdateTurnData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		if data.Player == player.ID {
			return s.decideTurn(player)
		}
	case tincho.UpdateTypeDraw:
		var data tincho.UpdateDrawData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		if data.Player == player.ID {
			data, err := json.Marshal(tincho.ActionDiscardData{
				CardPosition: rand.Intn(len(player.Hand)),
			})
			if err != nil {
				return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			return tincho.Action{Type: tincho.ActionDiscard, Data: json.RawMessage(data)}, nil
		}
	case tincho.UpdateTypeError:
		return tincho.Action{}, fmt.Errorf("recieved error update: %s", update.Data)
	case tincho.UpdateTypeStartNextRound:
		return tincho.Action{Type: tincho.ActionFirstPeek}, nil
	}
	return tincho.Action{}, nil
}

func (s EasyStrategy) decideTurn(player tincho.Player) (tincho.Action, error) {
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

func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}
