package main

import (
	"context"
	"football/infrastructure/bootstrap"
	"log/slog"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, svc, err := bootstrap.Open(ctx)
	if err == nil {
		defer store.Close()
		if len(os.Args) < 2 {
			slog.Error("use: manage seed|migrate-up|migrate-down|migrate-status")
			os.Exit(1)
		}
		switch os.Args[1] {
		case "seed":
			err = svc.Run(ctx, true)
		case "migrate-up":
			err = store.Migrate(ctx, "migrations", "up")
		case "migrate-down":
			err = store.Migrate(ctx, "migrations", "down")
		case "migrate-status":
			err = store.Migrate(ctx, "migrations", "status")
		default:
			slog.Error("unknown command")
			os.Exit(1)
		}
	}
	if err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
	slog.Info("command complete")
}
