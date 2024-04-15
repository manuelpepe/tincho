package sim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/game"
	"github.com/manuelpepe/tincho/internal/tincho"
)

var ErrSimTimeout = errors.New("simulation timed out")

type Result struct {
	Winner      int
	TotalRounds int
	TotalTurns  int
}

func Play(ctx context.Context, logger *slog.Logger, strat bots.Strategy, strat2 bots.Strategy) (Result, error) {
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
		logger.Error("Simulation timed out after 60 seconds", "total_rounds", room.TotalRounds())
		return Result{}, ErrSimTimeout
	}
}

func Compete(ctx context.Context, logger *slog.Logger, strat func() bots.Strategy, strat2 func() bots.Strategy, rounds int) ([]Result, error) {
	results := make([]Result, rounds)

	for i := 0; i < rounds; i++ {
		result, err := Play(ctx, logger, strat(), strat2())
		if err != nil {
			return nil, fmt.Errorf("error on round %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

type MinMaxMean struct {
	Min  int
	Max  int
	Mean int
}

type StratSummary struct {
	Wins   int
	Rounds MinMaxMean
	Turns  MinMaxMean
}

type Summary struct {
	Strat1Summary StratSummary
	Strat2Summary StratSummary

	TotalRounds int
	TotalTurns  int
}

func Summarize(results []Result) Summary {
	var strat1TotalRounds, strat2TotalRounds int
	var strat1TotalTurns, strat2TotalTurns int

	summary := Summary{
		Strat1Summary: StratSummary{
			Rounds: MinMaxMean{Min: 9999},
			Turns:  MinMaxMean{Min: 9999},
		},
		Strat2Summary: StratSummary{
			Rounds: MinMaxMean{Min: 9999},
			Turns:  MinMaxMean{Min: 9999},
		},
	}
	var winnerSummary *StratSummary

	for _, result := range results {
		if result.Winner == 0 {
			winnerSummary = &summary.Strat1Summary
			strat1TotalRounds += result.TotalRounds
			strat1TotalTurns += result.TotalTurns
		} else {
			winnerSummary = &summary.Strat2Summary
			strat2TotalRounds += result.TotalRounds
			strat2TotalTurns += result.TotalTurns
		}

		winnerSummary.Wins++
		if result.TotalRounds < winnerSummary.Rounds.Min {
			winnerSummary.Rounds.Min = result.TotalRounds
		}
		if result.TotalRounds > winnerSummary.Rounds.Max {
			winnerSummary.Rounds.Max = result.TotalRounds
		}
		if result.TotalTurns < winnerSummary.Turns.Min {
			winnerSummary.Turns.Min = result.TotalTurns
		}
		if result.TotalTurns > winnerSummary.Turns.Max {
			winnerSummary.Turns.Max = result.TotalTurns
		}
	}

	if summary.Strat1Summary.Wins > 0 {
		summary.Strat1Summary.Rounds.Mean = strat1TotalRounds / summary.Strat1Summary.Wins
		summary.Strat1Summary.Turns.Mean = strat1TotalTurns / summary.Strat1Summary.Wins
	}
	if summary.Strat2Summary.Wins > 0 {
		summary.Strat2Summary.Rounds.Mean = strat2TotalRounds / summary.Strat2Summary.Wins
		summary.Strat2Summary.Turns.Mean = strat2TotalTurns / summary.Strat2Summary.Wins
	}

	summary.TotalRounds = strat1TotalRounds + strat2TotalRounds
	summary.TotalTurns = strat1TotalTurns + strat2TotalTurns

	return summary
}
