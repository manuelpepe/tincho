package sim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/manuelpepe/tincho/pkg/bots"
	"github.com/manuelpepe/tincho/pkg/game"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

var ErrSimTimeout = errors.New("simulation timed out")

// Result of a single game
type Result struct {
	Winner      int
	TotalRounds int
	TotalTurns  int
}

// Three common values
type MinMaxMeanSum struct {
	Min  int
	Max  int
	Mean int
	Sum  int
}

// Summary of multiple games for a single strategy
type StratSummary struct {
	Wins   int
	Rounds MinMaxMeanSum
	Turns  MinMaxMeanSum
}

// Summary of multiple games for two strategies
type Summary struct {
	Strats []StratSummary

	TotalGames int
	Rounds     MinMaxMeanSum
	Turns      MinMaxMeanSum
}

func (s *Summary) record(result Result) error {
	if s == nil {
		return errors.New("nil summary")
	}
	s.TotalGames++

	s.Rounds.Sum += result.TotalRounds
	s.Rounds.Min = min(result.TotalRounds, s.Rounds.Min)
	s.Rounds.Max = max(result.TotalRounds, s.Rounds.Max)

	s.Turns.Sum += result.TotalTurns
	s.Turns.Min = min(result.TotalTurns, s.Turns.Min)
	s.Turns.Max = max(result.TotalTurns, s.Turns.Max)

	winnerSummary := s.Strats[result.Winner]
	winnerSummary.Wins++

	winnerSummary.Rounds.Sum += result.TotalRounds
	winnerSummary.Rounds.Min = min(result.TotalRounds, winnerSummary.Rounds.Min)
	winnerSummary.Rounds.Max = max(result.TotalRounds, winnerSummary.Rounds.Max)
	winnerSummary.Rounds.Mean = winnerSummary.Rounds.Sum / winnerSummary.Wins

	winnerSummary.Turns.Sum += result.TotalTurns
	winnerSummary.Turns.Min = min(result.TotalTurns, winnerSummary.Turns.Min)
	winnerSummary.Turns.Max = max(result.TotalTurns, winnerSummary.Turns.Max)
	winnerSummary.Turns.Mean = winnerSummary.Turns.Sum / winnerSummary.Wins

	s.Strats[result.Winner] = winnerSummary

	s.Rounds.Mean = s.Rounds.Sum / s.TotalGames
	s.Turns.Mean = s.Turns.Sum / s.TotalGames

	return nil
}

func (s Summary) AsText() string {
	res := ""
	for i, strat := range s.Strats {
		res += fmt.Sprintf("%d: %+v\n", i, strat)
	}
	res += fmt.Sprintf("Total Games: %d\n", s.TotalGames)
	res += fmt.Sprintf("Total Rounds: %+v\n", s.Rounds)
	res += fmt.Sprintf("Total Turns: %+v\n", s.Turns)
	return res
}

func Play(ctx context.Context, logger *slog.Logger, strats ...bots.Strategy) (Result, error) {
	ctx, cancel := context.WithCancel(ctx)

	deck := game.NewDeck()
	deck.Shuffle()

	roomID := generateRandomString(6)
	logger = logger.With("room", roomID)
	room := tincho.NewRoomWithDeck(logger, ctx, cancel, roomID, deck, len(strats))
	go room.Start()

	type b struct {
		Ix  int
		Bot *bots.Bot
	}

	players := make(map[game.PlayerID]b)
	for ix, strat := range strats {
		name := game.PlayerID(fmt.Sprintf("strat-%d", ix))
		bot := bots.NewBotFromStrategy(logger, ctx, tincho.NewConnection(name), strat)
		room.AddConnection(bot.Connection())
		go func() {
			if err := bot.Start(); err != nil {
				logger.Error("Bot failed with error", "error", err)
			}
		}()
		players[name] = b{Ix: ix, Bot: &bot}
	}

	players["strat-0"].Bot.Connection().QueueAction(&tincho.Action[tincho.ActionWithoutData]{Type: tincho.ActionStart})

	select {
	case <-ctx.Done():
		winner, err := room.Winner()
		if err != nil {
			return Result{}, err
		}
		return Result{
			Winner:      players[winner.ID].Ix,
			TotalRounds: room.TotalRounds(),
			TotalTurns:  room.TotalTurns(),
		}, nil
	case <-time.After(60 * time.Second):
		logger.Error("Simulation timed out after 60 seconds", "total_rounds", room.TotalRounds(), "total_turns", room.TotalTurns())
		return Result{}, fmt.Errorf("error on room %s: %w", roomID, ErrSimTimeout)
	}
}

type StrategyFactory = func() bots.Strategy

// A worker consumes signals from a pending channel and starts a match with the given strategies,
// sending results to the outs channel and errors to the errs channel.
// The worker exits on ctx.Done().
func worker(ctx context.Context, logger *slog.Logger, pending <-chan struct{}, outs chan<- Result, errs chan<- error, strats ...StrategyFactory) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pending:
			bots := make([]bots.Strategy, 0, len(strats))
			for _, strat := range strats {
				bots = append(bots, strat())
			}

			result, err := Play(ctx, logger, bots...)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case errs <- fmt.Errorf("error on round: %w", err):
				}
			}

			select {
			case <-ctx.Done():
				return
			case outs <- result:
			}

		}
	}
}

func Compete(ctx context.Context, logger *slog.Logger, rounds int, strats ...StrategyFactory) (Summary, error) {
	if rounds < 1 {
		return Summary{}, fmt.Errorf("invalid number of rounds: %d", rounds)
	}

	ctx, cancelPendingGames := context.WithCancel(ctx)

	// start worker goroutines
	routines := min(rounds, 10000)
	pending := make(chan struct{})
	outs := make(chan Result)
	errs := make(chan error)
	for i := 0; i < routines; i++ {
		go worker(ctx, logger, pending, outs, errs, strats...)
	}

	// start summarizing goroutine
	var finalResChan = make(chan Summary)
	var finalErrChan = make(chan error)
	go func() {
		summary := Summary{
			Strats: make([]StratSummary, 0, len(strats)),
			Rounds: MinMaxMeanSum{Min: 9999},
			Turns:  MinMaxMeanSum{Min: 9999},
		}

		for i := 0; i < len(strats); i++ {
			summary.Strats = append(summary.Strats, StratSummary{
				Rounds: MinMaxMeanSum{Min: 9999},
				Turns:  MinMaxMeanSum{Min: 9999},
			})
		}

		for i := 0; i < rounds; i++ {
			select {
			case result := <-outs:
				summary.record(result)
			case err := <-errs:
				finalErrChan <- err
				return
			}
		}

		finalResChan <- summary
	}()

	// start queueing goroutine
	go func() {
		for i := 0; i < rounds; i++ {
			select {
			case <-ctx.Done():
				return
			case pending <- struct{}{}:
			}
		}
	}()

	// wait for either output
	select {
	case res := <-finalResChan:
		cancelPendingGames()
		return res, nil
	case err := <-finalErrChan:
		cancelPendingGames()
		return Summary{}, err
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateRandomString(length int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
