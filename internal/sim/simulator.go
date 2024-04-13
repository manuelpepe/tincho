package sim

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/game"
	"github.com/manuelpepe/tincho/internal/tincho"
)

/*
Simulator would:

1. take two bots
2. face them against each other N times
3. return the results

This would be useful for testing the bots against each other
and running competetions between them.
*/

var ErrSimTimeout = errors.New("simulation timed out")

type Result struct {
	Winner      int
	TotalRounds int
	TotalTurns  int
}

func compete(ctx context.Context, strat bots.Strategy, strat2 bots.Strategy) (Result, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(ctx)

	deck := game.NewDeck()
	deck.Shuffle()
	room := tincho.NewRoomWithDeck(logger, ctx, cancel, "sim-room", deck, 2)
	bot := bots.NewBotFromStrategy(logger, ctx, tincho.NewConnection("strat-1"), strat)
	bot2 := bots.NewBotFromStrategy(logger, ctx, tincho.NewConnection("strat-2"), strat2)

	go room.Start()
	go bot.Start()
	go bot2.Start()

	room.AddPlayer(bot.Player())
	room.AddPlayer(bot2.Player())

	bot.Player().QueueAction(tincho.Action{Type: tincho.ActionStart})

	select {
	case <-ctx.Done():
		winner, err := room.Winner()
		if err != nil {
			return Result{}, err
		}

		var winnerIx int
		if winner.ID == bot.Player().ID {
			winnerIx = 0
		} else {
			winnerIx = 1
		}

		return Result{
			Winner:      winnerIx,
			TotalRounds: room.TotalRounds(),
			TotalTurns:  room.TotalTurns(),
		}, nil
	case <-time.After(60 * time.Second):
		logger.Error("Simulation timed out after 10 seconds", "total_rounds", room.TotalRounds())
		return Result{}, ErrSimTimeout
	}
}
