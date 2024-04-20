package bots

import (
	"context"
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

func (b *Bot) RespondToUpdate(player *tincho.Connection, update tincho.Typed) (tincho.Action, error) {
	typed, ok := update.(tincho.Typed)
	if !ok {
		return tincho.Action{}, fmt.Errorf("update is not Typed")
	}

	b.logger.Debug(fmt.Sprintf("Bot %s received update: %s", player.ID, typed.GetType()), "update", update)

	switch typed.GetType() {
	case tincho.UpdateTypeGameStart:
		up, ok := update.(tincho.Update[tincho.UpdateStartNextRoundData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.GameStart(player, up.Data)
	case tincho.UpdateTypePlayersChanged:
		up, ok := update.(tincho.Update[tincho.UpdatePlayersChangedData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.PlayersChanged(player, up.Data)
	case tincho.UpdateTypePlayerFirstPeeked:
		up, ok := update.(tincho.Update[tincho.UpdatePlayerFirstPeekedData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.PlayerFirstPeeked(player, up.Data)
	case tincho.UpdateTypeTurn:
		up, ok := update.(tincho.Update[tincho.UpdateTurnData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.Turn(player, up.Data)
	case tincho.UpdateTypeDraw:
		up, ok := update.(tincho.Update[tincho.UpdateDrawData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.Draw(player, up.Data)
	case tincho.UpdateTypePeekCard:
		up, ok := update.(tincho.Update[tincho.UpdatePeekCardData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.PeekCard(player, up.Data)
	case tincho.UpdateTypeSwapCards:
		up, ok := update.(tincho.Update[tincho.UpdateSwapCardsData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.SwapCards(player, up.Data)
	case tincho.UpdateTypeDiscard:
		up, ok := update.(tincho.Update[tincho.UpdateDiscardData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.Discard(player, up.Data)
	case tincho.UpdateTypeFailedDoubleDiscard:
		up, ok := update.(tincho.Update[tincho.UpdateTypeFailedDoubleDiscardData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.FailedDoubleDiscard(player, up.Data)
	case tincho.UpdateTypeCut:
		up, ok := update.(tincho.Update[tincho.UpdateCutData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.Cut(player, up.Data)
	case tincho.UpdateTypeError:
		up, ok := update.(tincho.Update[tincho.UpdateErrorData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.Error(player, up.Data)
	case tincho.UpdateTypeStartNextRound:
		up, ok := update.(tincho.Update[tincho.UpdateStartNextRoundData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.StartNextRound(player, up.Data)
	case tincho.UpdateTypeEndGame:
		up, ok := update.(tincho.Update[tincho.UpdateEndGameData])
		if !ok {
			return tincho.Action{}, fmt.Errorf("update data is not UpdateStartNextRoundData")
		}
		return b.strategy.EndGame(player, up.Data)
	}
	return tincho.Action{}, nil
}
