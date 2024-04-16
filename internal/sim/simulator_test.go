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
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	sum, err := Compete(ctx, logger, easy, easy, 2000)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)

}

func TestEvM(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	sum, err := Compete(ctx, logger, easy, medium, 2000)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)
}

func TestEvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	easy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, easy, hard, 2000)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)
}

func TestMvM(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	sum, err := Compete(ctx, logger, medium, medium, 2000)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)
}

func TestMvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	medium := func() bots.Strategy {
		return bots.NewMediumStrategy()
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, medium, hard, 2000)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)
}

func TestHvH(t *testing.T) {
	ctx := context.Background()

	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	hard := func() bots.Strategy {
		return bots.NewHardStrategy()
	}

	sum, err := Compete(ctx, logger, hard, hard, 10)
	assert.NoError(t, err)

	fmt.Printf("Summary: %+v\n", sum)
}
