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
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, easy, easy)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestEvM(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, easy, medium)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestEvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, easy, hard)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestMvM(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, medium, medium)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestMvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, medium, hard)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestHvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, 200, hard, hard)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}

func TestEvMvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, 2000, easy, medium, hard)
	assert.NoError(t, err)

	fmt.Printf(sum.AsText())
}
