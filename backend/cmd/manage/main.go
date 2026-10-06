package main

import (
	"context"
	"fmt"
	"football/infrastructure/bootstrap"
	"football/infrastructure/security"
	"football/internal/application/usecase"
	"io"
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
			slog.Error("use: manage seed|create-owner|migrate-up|migrate-down|migrate-status")
			os.Exit(1)
		}
		switch os.Args[1] {
		case "create-owner":
			var members *usecase.Membership
			members, err = usecase.NewMembership(store, security.Passwords{}, security.Tokens{}, bootstrap.Clock{}, bootstrap.IDs{})
			if err == nil {
				var password []byte
				password, err = io.ReadAll(io.LimitReader(os.Stdin, 73))
				if err == nil && os.Getenv("OWNER_EMAIL") == "" {
					err = fmt.Errorf("OWNER_EMAIL is required")
				}
				if err == nil {
					var token string
					_, token, err = members.Register(ctx, bootstrap.Env("OWNER_NAME", "Owner"), os.Getenv("OWNER_EMAIL"), string(password))
					if err == nil {
						err = members.Logout(ctx, token)
					}
				}
			}
		case "seed":
			if bootstrap.Env("FOOTBALL_PROVIDER", "mock") == "api-football" {
				slog.Info("mock seed skipped in real-data mode; run make worker to sync")
			} else {
				err = svc.Run(ctx, true)
			}
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
