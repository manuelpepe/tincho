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
type MinMaxMean struct {
	Min  int
	Max  int
	Mean int
}

// Summary of multiple games for a single strategy
type StratSummary struct {
	Wins   int
	Rounds MinMaxMean
	Turns  MinMaxMean

	TotalRounds int
	TotalTurns  int
}

// Summary of multiple games for two strategies
type Summary struct {
	Strats []StratSummary

	TotalRounds int
	TotalTurns  int
}

func (s Summary) AsText() string {
	res := ""
	for i, strat := range s.Strats {
		res += fmt.Sprintf("%d: %+v\n", i, strat)
	}
	res += fmt.Sprintf("TotalRounds: %d\n", s.TotalRounds)
	res += fmt.Sprintf("TotalTurns: %d\n", s.TotalTurns)
	return res
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
		logger.Error("Simulation timed out after 60 seconds", "total_rounds", room.TotalRounds()) // RACE: on total rounds
		return Result{}, fmt.Errorf("error on room %s: %w", roomID, ErrSimTimeout)
	}
}

func Compete(ctx context.Context, logger *slog.Logger, rounds int, strats ...func() bots.Strategy) (Summary, error) {
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

		summary := Summary{}
		for i := 0; i < len(strats); i++ {
			summary.Strats = append(summary.Strats, StratSummary{
				Rounds: MinMaxMean{Min: 9999},
				Turns:  MinMaxMean{Min: 9999},
			})
		}

		for i := 0; i < rounds; i++ {
			select {
			case result := <-outs:
				winnerSummary := summary.Strats[result.Winner]

				winnerSummary.Wins++
				winnerSummary.TotalRounds += result.TotalRounds
				winnerSummary.TotalTurns += result.TotalTurns

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

				summary.Strats[result.Winner] = winnerSummary
			case err := <-errs:
				finalErrChan <- err
				cancelPendingGames()
				return
			}
		}

		for _, strat := range summary.Strats {
			if strat.Wins > 0 {
				strat.Rounds.Mean = strat.TotalRounds / strat.Wins
				strat.Turns.Mean = strat.TotalTurns / strat.Wins
			}
			summary.TotalRounds += strat.TotalRounds
			summary.TotalTurns += strat.TotalTurns
		}

		finalResChan <- summary
	}()

	select {
	case res := <-finalResChan:
		return res, nil
	case err := <-finalErrChan:
		return Summary{}, err
	}
}
