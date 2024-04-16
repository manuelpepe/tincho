package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/manuelpepe/tincho/internal/bots"
	"github.com/manuelpepe/tincho/internal/sim"
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

func run(name string, iters int, showLogs bool, strats ...func() bots.Strategy) error {
	start := time.Now()

	var logger *slog.Logger
	if showLogs {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	ctx := context.Background()
	sum, err := sim.Compete(ctx, logger, iters, strats...)
	if err != nil {
		return err
	}

	end := time.Now().Sub(start)
	fmt.Printf("=== RUN: %s\n%+v--- OK (%s)\n\n", name, sum.AsText(), end.String())
	return nil
}

func main() {
	go func() {
		fmt.Println("Starting pprof")
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	run("EvE", 10000, false, easy, easy)
	run("EvM", 10000, false, easy, medium)
	run("EvH", 10000, false, easy, hard)
	run("MvM", 10000, false, medium, medium)
	run("MvH", 10000, false, medium, hard)
	run("HvH", 10, false, hard, hard)
	run("EvMvH", 10000, false, easy, medium, hard)
}
