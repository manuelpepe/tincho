package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/pprof"
	"slices"
	"time"

	"github.com/manuelpepe/tincho/pkg/bots"
	"github.com/manuelpepe/tincho/pkg/sim"
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
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	}

	ctx := context.Background()
	sum, err := sim.Compete(ctx, logger, iters, strats...)
	if err != nil {
		fmt.Printf("=== RUN: %s\n%s\n--- ERR (%s)\n\n", name, err, time.Since(start).String())
		return err
	}

	fmt.Printf("=== RUN: %s\n%+v--- OK (%s)\n\n", name, sum.AsText(), time.Since(start).String())
	return nil
}

func main() {
	var showLogs, all, eve, evm, evh, mvm, mvh, hvh, evmvh bool
	var pp string
	var iters int

	flag.BoolVar(&showLogs, "logs", false, "Show logs")
	flag.StringVar(&pp, "pp", "", "Run pprof")
	flag.IntVar(&iters, "iters", 10000, "Number of iterations")

	flag.BoolVar(&all, "all", false, "Run all")
	flag.BoolVar(&eve, "ee", false, "Run Easy vs Easy")
	flag.BoolVar(&evm, "em", false, "Run Easy vs Medium")
	flag.BoolVar(&evh, "eh", false, "Run Easy vs Hard")
	flag.BoolVar(&mvm, "mm", false, "Run Medium vs Medium")
	flag.BoolVar(&mvh, "mh", false, "Run Medium vs Hard")
	flag.BoolVar(&hvh, "hh", false, "Run Hard vs Hard")
	flag.BoolVar(&evmvh, "emh", false, "Run Easy vs Medium vs Hard")

	flag.Parse()

	allFlags := []bool{eve, evm, evh, mvm, mvh, hvh, evmvh}
	if all && slices.Contains(allFlags, true) {
		fmt.Println("Cannot use -all with other sim specific flags")
		return
	}

	if pp != "" {
		f, err := os.Create(pp)
		if err != nil {
			fmt.Println(err)
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if all || eve {
		run("EvE", iters, showLogs, easy, easy)
	}

	if all || evm {
		run("EvM", iters, showLogs, easy, medium)
	}

	if all || evh {
		run("EvH", iters, showLogs, easy, hard)
	}

	if all || mvm {
		run("MvM", iters, showLogs, medium, medium)
	}

	if all || mvh {
		run("MvH", iters, showLogs, medium, hard)
	}

	if hvh {
		if all {
			fmt.Println("[W] HvH simulation disable in -all mode.")
			fmt.Println("    (HvH simulations can take a long time going back and forth in points)")
		} else {
			run("HvH", iters, showLogs, hard, hard)
		}
	}

	if all || evmvh {
		run("EvMvH", iters, showLogs, easy, medium, hard)
	}
}
