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

func TestEasyVsMediumCompeteSummary(t *testing.T) {
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

	res, err := Compete(ctx, logger, easy, medium, 1000)
	assert.NoError(t, err)

	summary := Summarize(res)

	fmt.Printf("Summary: %+v\n", summary)
}

func TestEasyVsHardCompeteSummary(t *testing.T) {
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

	res, err := Compete(ctx, logger, easy, hard, 1000)
	assert.NoError(t, err)

	summary := Summarize(res)

	fmt.Printf("Summary: %+v\n", summary)
}

func TestMediumVsHardCompeteSummary(t *testing.T) {
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

	res, err := Compete(ctx, logger, medium, hard, 1000)
	assert.NoError(t, err)

	summary := Summarize(res)

	fmt.Printf("Summary: %+v\n", summary)
}
