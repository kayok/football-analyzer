package main

import (
	"context"
	"errors"
	"football/infrastructure/bootstrap"
	"football/infrastructure/security"
	"football/internal/application/usecase"
	delivery "football/internal/delivery/http"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, svc, err := bootstrap.Open(ctx)
	if err != nil {
		log.Error("startup failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	syncer, err := bootstrap.Sync(ctx, store, svc, log)
	if err != nil {
		log.Error("sync startup failed", "error", err)
		os.Exit(1)
	}
	defer func() { stop(); syncer.Wait() }()
	members, err := usecase.NewPrivateMembership(store, security.Passwords{}, security.Tokens{}, bootstrap.Clock{}, bootstrap.IDs{}, os.Getenv("OWNER_EMAIL"))
	if err != nil {
		log.Error("membership startup failed", "error", err)
		os.Exit(1)
	}
	syncer.Serve()
	server := &http.Server{Addr: net.JoinHostPort(bootstrap.Env("API_HOST", "127.0.0.1"), bootstrap.Env("API_PORT", "8080")), Handler: delivery.New(svc, members, bootstrap.Env("FRONTEND_ORIGIN", "http://localhost:3000"), log, bootstrap.Clock{}, syncer), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan error, 1)
	go func() { log.Info("API listening", "address", server.Addr); done <- server.ListenAndServe() }()
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(shutdown); err != nil {
			log.Error("shutdown failed", "error", err)
		}
	}
}
