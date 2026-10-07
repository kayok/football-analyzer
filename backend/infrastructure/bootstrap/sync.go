package bootstrap

import (
	"context"
	"fmt"
	"football/internal/application/ports"
	"football/internal/application/usecase"
	"log/slog"
	"strconv"
	"time"
)

func Sync(ctx context.Context, store ports.SyncStore, runner usecase.SyncRunner, log *slog.Logger) (*usecase.SyncManager, error) {
	enabled, err := strconv.ParseBool(Env("SYNC_AUTO_ENABLED", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid SYNC_AUTO_ENABLED")
	}
	interval, err := time.ParseDuration(Env("SYNC_INTERVAL", "24h"))
	if err != nil || interval < 15*time.Minute {
		return nil, fmt.Errorf("SYNC_INTERVAL must be at least 15m")
	}
	cooldown, err := time.ParseDuration(Env("SYNC_COOLDOWN", "30m"))
	if err != nil || cooldown < time.Minute {
		return nil, fmt.Errorf("SYNC_COOLDOWN must be at least 1m")
	}
	if !enabled {
		interval = 0
	}
	return usecase.NewSyncManager(ctx, store, runner, Clock{}, cooldown, interval, log), nil
}
