package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type Strategy interface {
	PlayersChanged(player tincho.Player, data tincho.UpdatePlayersChangedData) (tincho.Action, error)
	GameStart(player tincho.Player) (tincho.Action, error)
	PlayerFirstPeeked(player tincho.Player, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error)
	Turn(player tincho.Player, data tincho.UpdateTurnData) (tincho.Action, error)
	Draw(player tincho.Player, data tincho.UpdateDrawData) (tincho.Action, error)
	PeekCard(player tincho.Player, data tincho.UpdatePeekCardData) (tincho.Action, error)
	SwapCards(player tincho.Player, data tincho.UpdateSwapCardsData) (tincho.Action, error)
	Discard(player tincho.Player, data tincho.UpdateDiscardData) (tincho.Action, error)
	FailedDoubleDiscard(player tincho.Player) (tincho.Action, error)
	Cut(player tincho.Player, data tincho.UpdateCutData) (tincho.Action, error)
	Error(player tincho.Player, data tincho.UpdateErrorData) (tincho.Action, error)
	StartNextRound(player tincho.Player, data tincho.UpdateStartNextRoundData) (tincho.Action, error)
	EndGame(player tincho.Player, data tincho.UpdateEndGameData) (tincho.Action, error)
}

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
			action, err := b.RespondToUpdate(*b.player, update)
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

func (b *Bot) RespondToUpdate(player tincho.Player, update tincho.Update) (tincho.Action, error) {
	switch update.Type {
	case tincho.UpdateTypeGameStart:
		return b.strategy.GameStart(player)
	case tincho.UpdateTypePlayersChanged:
		var data tincho.UpdatePlayersChangedData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.PlayersChanged(player, data)
	case tincho.UpdateTypePlayerFirstPeeked:
		var data tincho.UpdatePlayerFirstPeekedData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.PlayerFirstPeeked(player, data)
	case tincho.UpdateTypeTurn:
		var data tincho.UpdateTurnData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.Turn(player, data)
	case tincho.UpdateTypeDraw:
		var data tincho.UpdateDrawData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.Draw(player, data)
	case tincho.UpdateTypePeekCard:
		var data tincho.UpdatePeekCardData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.PeekCard(player, data)
	case tincho.UpdateTypeSwapCards:
		var data tincho.UpdateSwapCardsData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.SwapCards(player, data)
	case tincho.UpdateTypeDiscard:
		var data tincho.UpdateDiscardData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.Discard(player, data)
	case tincho.UpdateTypeFailedDoubleDiscard:
		return b.strategy.FailedDoubleDiscard(player)
	case tincho.UpdateTypeCut:
		var data tincho.UpdateCutData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.Cut(player, data)
	case tincho.UpdateTypeError:
		var data tincho.UpdateErrorData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.Error(player, data)
	case tincho.UpdateTypeStartNextRound:
		var data tincho.UpdateStartNextRoundData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.StartNextRound(player, data)
	case tincho.UpdateTypeEndGame:
		var data tincho.UpdateEndGameData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.EndGame(player, data)
	}
	return tincho.Action{}, nil
}
