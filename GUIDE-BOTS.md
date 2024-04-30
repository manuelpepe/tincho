# Create custom Bots and simulate matches

1. Create and initialize a new repository

```bash
$ mkdir mybot
$ cd mybot
$ go mod init github.com/me/mybot
$ go get github.com/manuelpepe/tincho
```

2. Create strategy:

```go
// example main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/manuelpepe/tincho/pkg/bots"
	"github.com/manuelpepe/tincho/pkg/sim"
	"github.com/manuelpepe/tincho/pkg/tincho"
)

// 1.Create strategy implementing bots.Strategy interface
type MyBotStrategy struct {
	bots.BaseStrategy // embed to avoid reimplementing all methods
}

// 2. Implement methods you need from bots.Strategy
func (s *MyBotStrategy) GameStart(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return tincho.Action[tincho.ActionWithoutData]{Type: tincho.ActionFirstPeek}, nil
}

func (s *MyBotStrategy) StartNextRound(player *tincho.Connection, data tincho.UpdateStartNextRoundData) (tincho.TypedAction, error) {
	return tincho.Action[tincho.ActionWithoutData]{Type: tincho.ActionFirstPeek}, nil
}

func (s *MyBotStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.TypedAction, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}
	return tincho.Action[tincho.ActionCutData]{
		Type: tincho.ActionCut, 
		Data: tincho.ActionCutData{
			WithCount: false,
			Declared:  0,
		},
	}, nil
}

func main() {
	// 3. Run simulation

	// 3.1. Lambdas for creating bots
	newEasy := func() bots.Strategy {
		return bots.NewEasyStrategy()
	}

	newMyBot := func() bots.Strategy {
		return &MyBotStrategy{}
	}

	// 3.1. Create logger as preferred
	var logger *slog.Logger
	const showLogs = false
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	}

	// 3.2. Run
	sum, err := sim.Compete(context.Background(), logger, 100, newMyBot, newEasy)
	if err != nil {
		panic(err)
	}

	fmt.Printf(sum.AsText())
}
```

3. Run simulation:

```bash
$ go run ./main.go
```
