package sim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/game"
	"github.com/manuelpepe/tincho/internal/tincho"
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
		room.AddPlayer(bot.Player())
		go bot.Start()
		players[name] = b{Ix: ix, Bot: &bot}
	}

	players["strat-0"].Bot.Player().QueueAction(tincho.Action{Type: tincho.ActionStart})

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
		logger.Error("Simulation timed out after 60 seconds", "total_rounds", room.TotalRounds(), "total_turns", room.TotalTurns()) // RACE: on total rounds and turns
		return Result{}, fmt.Errorf("error on room %s: %w", roomID, ErrSimTimeout)
	}
}

func Compete(ctx context.Context, logger *slog.Logger, rounds int, strats ...func() bots.Strategy) (Summary, error) {
	if rounds < 1 {
		return Summary{}, fmt.Errorf("invalid number of rounds: %d", rounds)
	}

	ctx, cancelPendingGames := context.WithCancel(ctx)

	outs := make(chan Result)
	errs := make(chan error)
	for i := 0; i < rounds; i++ {
		go func() {
			select {
			case <-ctx.Done():
				return // early exit if done
			default:
			}

			bots := make([]bots.Strategy, 0, len(strats))
			for _, strat := range strats {
				bots = append(bots, strat())
			}
			result, err := Play(ctx, logger, bots...)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				errs <- fmt.Errorf("error on round: %w", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
			outs <- result

		}()
	}

	var finalResChan = make(chan Summary)
	var finalErrChan = make(chan error)
	go func() {
		defer close(outs)
		defer close(errs)

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
				summary.TotalGames++

				summary.Rounds.Sum += result.TotalRounds
				summary.Rounds.Min = min(result.TotalRounds, summary.Rounds.Min)
				summary.Rounds.Max = max(result.TotalRounds, summary.Rounds.Max)

				summary.Turns.Sum += result.TotalTurns
				summary.Turns.Min = min(result.TotalTurns, summary.Turns.Min)
				summary.Turns.Max = max(result.TotalTurns, summary.Turns.Max)

				winnerSummary := summary.Strats[result.Winner]
				winnerSummary.Wins++

				winnerSummary.Rounds.Sum += result.TotalRounds
				winnerSummary.Rounds.Min = min(result.TotalRounds, winnerSummary.Rounds.Min)
				winnerSummary.Rounds.Max = max(result.TotalRounds, winnerSummary.Rounds.Max)

				winnerSummary.Turns.Sum += result.TotalTurns
				winnerSummary.Turns.Min = min(result.TotalTurns, winnerSummary.Turns.Min)
				winnerSummary.Turns.Max = max(result.TotalTurns, winnerSummary.Turns.Max)

				summary.Strats[result.Winner] = winnerSummary
			case err := <-errs:
				finalErrChan <- err
				cancelPendingGames()
				return
			}
		}

		for ix, strat := range summary.Strats {
			if strat.Wins > 0 {
				strat.Rounds.Mean = strat.Rounds.Sum / strat.Wins
				strat.Turns.Mean = strat.Turns.Sum / strat.Wins
			}
			summary.Strats[ix] = strat
		}

		summary.Rounds.Mean = summary.Rounds.Sum / summary.TotalGames
		summary.Turns.Mean = summary.Turns.Sum / summary.TotalGames

		finalResChan <- summary
	}()

	select {
	case res := <-finalResChan:
		return res, nil
	case err := <-finalErrChan:
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
