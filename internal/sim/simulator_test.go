package sim

import (
	"context"
	"testing"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/stretchr/testify/assert"
)

func TestEasyVsMedium(t *testing.T) {
	ctx := context.Background()

	winsForMedium := 0
	for i := 0; i < 5; i++ {
		res, err := compete(ctx, &bots.EasyStrategy{}, &bots.MediumStrategy{})
		assert.NoError(t, err)
		winsForMedium += res.Winner
	}

	// medium should win 80% of the time at least
	assert.GreaterOrEqual(t, winsForMedium, 4)
}
