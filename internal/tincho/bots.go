package tincho

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Bot struct {
	ctx      context.Context
	player   *Player
	strategy Strategy
}

func NewBot(ctx context.Context, player *Player, difficulty string) (Bot, error) {
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
	RespondToUpdate(Player, Update) (Action, error)
}

type EasyStrategy struct{}

func (s *EasyStrategy) RespondToUpdate(player Player, update Update) (Action, error) {
	switch update.Type {
	case UpdateTypeGameStart:
		return Action{Type: ActionFirstPeek}, nil
	case UpdateTypeTurn:
		var data UpdateTurnData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		if data.Player == player.ID {
			data, err := json.Marshal(ActionDrawData{
				Source: RandChoice([]DrawSource{DrawSourcePile, DrawSourceDiscard}),
			})
			if err != nil {
				return Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			return Action{Type: ActionDraw, Data: data}, nil
		}
	case UpdateTypeDraw:
		var data UpdateDrawData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		if data.Player == player.ID {
			data, err := json.Marshal(ActionDiscardData{
				CardPosition: rand.Intn(len(player.Hand)),
			})
			if err != nil {
				return Action{}, fmt.Errorf("json.Marshal: %w", err)
			}
			return Action{Type: ActionDiscard, Data: json.RawMessage(data)}, nil
		}
	case UpdateTypeError:
		return Action{}, fmt.Errorf("recieved error update: %s", update.Data)
	case UpdateTypeStartNextRound:
		return Action{Type: ActionFirstPeek}, nil
	}
	return Action{}, nil
}

func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}
