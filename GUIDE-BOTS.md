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

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/sim"
	"github.com/manuelpepe/tincho/internal/tincho"
)

// 1.Create strategy implementing bots.Strategy interface
type MyBotStrategy struct {
	bots.BaseStrategy // embed to avoid reimplementing all methods
}

// 2. Implement methods you need from bots.Strategy
func (s *MyBotStrategy) Turn(player *tincho.Connection, data tincho.UpdateTurnData) (tincho.Action, error) {
	if data.Player != player.ID {
		return tincho.Action{}, nil
	}

	updateData, err := json.Marshal(tincho.ActionCutData{
		WithCount: false,
		Declared:  0,
	})
	if err != nil {
		return tincho.Action{}, fmt.Errorf("json.Marshal: %w", err)
	}
	return tincho.Action{Type: tincho.ActionCut, Data: updateData}, nil
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
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// 3.2. Run
	res, err := sim.Compete(context.Background(), logger, newMyBot, newEasy, 100)
	if err != nil {
		panic(err)
	}

	// 3.3. Summarize and sum
	sum := sim.Summarize(res)
	fmt.Printf("%+v\n", sum)
}
```

3. Run simulation:

```bash
$ go run ./main.go
```
