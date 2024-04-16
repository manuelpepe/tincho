package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/manuelpepe/tincho/internal/tincho"
)

type Strategy interface {
	PlayersChanged(player *tincho.Connection, data tincho.UpdatePlayersChangedData) (tincho.Action, error)
	GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error)
	StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.Action, error)
	PlayerFirstPeeked(player *tincho.Connection, data tincho.UpdatePlayerFirstPeekedData) (tincho.Action, error)
	Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error)
	Draw(player *tincho.Connection, data tincho.UpdateDrawData) (tincho.Action, error)
	PeekCard(player *tincho.Connection, data tincho.UpdatePeekCardData) (tincho.Action, error)
	SwapCards(player *tincho.Connection, data tincho.UpdateSwapCardsData) (tincho.Action, error)
	Discard(player *tincho.Connection, data tincho.UpdateDiscardData) (tincho.Action, error)
	FailedDoubleDiscard(player *tincho.Connection, data tincho.UpdateTypeFailedDoubleDiscardData) (tincho.Action, error)
	Cut(player *tincho.Connection, data tincho.UpdateCutData) (tincho.Action, error)
	Error(player *tincho.Connection, data tincho.UpdateErrorData) (tincho.Action, error)
	EndGame(player *tincho.Connection, data tincho.UpdateEndGameData) (tincho.Action, error)
}

type Bot struct {
	ctx      context.Context
	player   *tincho.Connection
	strategy Strategy
	logger   *slog.Logger
}

func NewBot(logger *slog.Logger, ctx context.Context, player *tincho.Connection, difficulty string) (Bot, error) {
	var strategy Strategy
	switch difficulty {
	case "easy":
		strategy = NewEasyStrategy()
	case "medium":
		strategy = NewMediumStrategy()
	case "hard":
		strategy = NewHardStrategy()
	// case "expert":
	default:
		return Bot{}, fmt.Errorf("invalid difficulty: %s", difficulty)
	}
	return Bot{
		ctx:      ctx,
		player:   player,
		strategy: strategy,
		logger:   logger,
	}, nil

}

func NewBotFromStrategy(logger *slog.Logger, ctx context.Context, player *tincho.Connection, strategy Strategy) Bot {
	return Bot{
		ctx:      ctx,
		player:   player,
		strategy: strategy,
		logger:   logger,
	}
}

func (b *Bot) Player() *tincho.Connection {
	return b.player
}

func (b *Bot) Strategy() Strategy {
	return b.strategy
}

func (b *Bot) Start() error {
	b.logger.Info(fmt.Sprintf("Bot %s started", b.player.ID))
	for {
		select {
		case update := <-b.player.Updates:
			action, err := b.RespondToUpdate(b.player, update)
			if err != nil {
				return fmt.Errorf("error responding to update: %w", err)
			}
			if action.Type != "" {
				b.player.QueueAction(action)
			}
		case <-b.ctx.Done():
			b.logger.Info(fmt.Sprintf("Bot %s finished", b.player.ID))
			return nil
		}
	}
}

func (b *Bot) RespondToUpdate(player *tincho.Connection, update tincho.Update) (tincho.Action, error) {
	b.logger.Debug(fmt.Sprintf("Bot %s received update: %s", player.ID, update.Type), "update", update)
	switch update.Type {
	case tincho.UpdateTypeGameStart:
		var data tincho.UpdateStartNextRoundData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.GameStart(player, data)
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
		var data tincho.UpdateTypeFailedDoubleDiscardData
		if err := json.Unmarshal(update.Data, &data); err != nil {
			return tincho.Action{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
		return b.strategy.FailedDoubleDiscard(player, data)
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
