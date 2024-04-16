package sim

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/stretchr/testify/assert"
)

func easy() bots.Strategy {
	return bots.NewEasyStrategy()
}

func medium() bots.Strategy {
	return bots.NewMediumStrategy()
}

func hard() bots.Strategy {
	return bots.NewHardStrategy()
}

func run(iters int, showLogs bool, strats ...func() bots.Strategy) error {
	var logger *slog.Logger
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	ctx := context.Background()
	sum, err := Compete(ctx, logger, iters, strats...)
	if err != nil {
		return err
	}

	fmt.Printf(sum.AsText())
	return nil
}

func TestEasyVsMedium(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	winsForMedium := 0
	for i := 0; i < 100; i++ {
		res, err := Play(ctx, logger, &bots.EasyStrategy{}, &bots.MediumStrategy{})
		assert.NoError(t, err)
		winsForMedium += res.Winner
	}

	// medium should win 80% of the time at least
	assert.GreaterOrEqual(t, winsForMedium, 80)
	fmt.Printf("Medium won %d times\n", winsForMedium)
}

func TestEvE(t *testing.T) {
	assert.NoError(t, run(2000, false, easy, easy))
}

func TestEvM(t *testing.T) {
	assert.NoError(t, run(2000, false, easy, medium))
}

func TestEvH(t *testing.T) {
	assert.NoError(t, run(2000, false, easy, hard))
}

func TestMvM(t *testing.T) {
	assert.NoError(t, run(2000, false, medium, medium))
}

func TestMvH(t *testing.T) {
	assert.NoError(t, run(2000, false, medium, hard))
}

func TestHvH(t *testing.T) {
	assert.NoError(t, run(10, false, hard, hard))
}

func TestEvMvH(t *testing.T) {
	assert.NoError(t, run(2000, false, easy, medium, hard))
}
