package main

import (
	"context"
	"flag"
	"football/infrastructure/bootstrap"
	"log/slog"
	"os"
	"time"
)

func main() {
	finish := flag.String("finish-match", "", "mock match UUID to finish (optional)")
	home := flag.Int("home", 1, "mock final home score")
	away := flag.Int("away", 0, "mock final away score")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, svc, err := bootstrap.Open(ctx)
	if err == nil {
		defer store.Close()
		if *finish != "" {
			err = svc.FinishDemo(ctx, *finish, *home, *away)
		} else {
			err = svc.Run(ctx, false)
		}
	}
	if err != nil {
		slog.Error("worker failed", "error", err)
		os.Exit(1)
	}
	slog.Info("worker complete")
}
