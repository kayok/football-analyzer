package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"football/infrastructure/ai"
	"football/infrastructure/footballapi"
	"football/infrastructure/postgres"
	"football/internal/application/ports"
	"football/internal/application/usecase"
	"football/internal/prediction/engine"
	rec "football/internal/recommendation/domain"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

type IDs struct{}

func (IDs) New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func Open(ctx context.Context) (*postgres.Store, *usecase.Service, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is required; run from the root using make after copying .env.example to .env")
	}
	tz := Env("APP_TIMEZONE", "Asia/Bangkok")
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, nil, err
	}
	threshold, err := strconv.ParseFloat(Env("PLAY_EV_THRESHOLD", "0.05"), 64)
	if err != nil || threshold <= 0 || math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return nil, nil, fmt.Errorf("invalid PLAY_EV_THRESHOLD")
	}
	age, err := time.ParseDuration(Env("ODDS_MAX_AGE", "15m"))
	if err != nil || age <= 0 {
		return nil, nil, fmt.Errorf("invalid ODDS_MAX_AGE")
	}
	store, err := postgres.New(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	eng := engine.Poisson{}
	var provider ports.FootballProvider = footballapi.MockFootballProvider{Engine: eng}
	var summary ports.AISummaryProvider = ai.MockAISummaryProvider{}
	name := Env("FOOTBALL_PROVIDER", "mock")
	switch name {
	case "mock":
	case "api-football":
		leagues := []int{}
		for _, raw := range strings.Split(Env("API_FOOTBALL_LEAGUES", "39,140,135,78,61"), ",") {
			id, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil || id <= 0 {
				store.Close()
				return nil, nil, fmt.Errorf("invalid API_FOOTBALL_LEAGUES")
			}
			leagues = append(leagues, id)
		}
		budget, err := strconv.Atoi(Env("API_FOOTBALL_MAX_REQUESTS", "50"))
		if err != nil || budget < 1 || budget > 100 {
			store.Close()
			return nil, nil, fmt.Errorf("API_FOOTBALL_MAX_REQUESTS must be 1–100")
		}
		provider = footballapi.APIFootball{Key: os.Getenv("API_FOOTBALL_KEY"), Leagues: leagues, Store: store, MaxRequests: budget, ReportUsage: footballapi.LogUsage}
		eng.ModelVersion = "v1-recent-goals-poisson"
		summary = ai.RuleSummaryProvider{}
	default:
		store.Close()
		return nil, nil, fmt.Errorf("invalid FOOTBALL_PROVIDER")
	}
	svc := usecase.New(store, Clock{}, IDs{}, eng, provider, summary, rec.Rules{PlayEV: threshold, MaxAge: age}, tz).WithProvider(name)
	if name != "mock" {
		svc = svc.WithRealData()
	}
	return store, svc, nil
}
