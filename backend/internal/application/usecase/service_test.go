package usecase

import (
	"context"
	"errors"
	"football/internal/application/ports"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	predports "football/internal/prediction/ports"
	rec "football/internal/recommendation/domain"
	"testing"
	"time"
)

type memoryStore struct{ state ports.State }

func (m *memoryStore) View(context.Context) (ports.State, error) { return m.state, nil }
func (m *memoryStore) ViewDate(context.Context, time.Time, time.Time) (ports.State, error) {
	return m.state, nil
}
func (m *memoryStore) Update(_ context.Context, fn func(*ports.State) error) error {
	return fn(&m.state)
}
func (m *memoryStore) Health(context.Context) error { return nil }

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type ids struct{}

func (ids) New() string { return "test-id" }

type unavailableEngine struct{}

func (unavailableEngine) Version() string { return "test-model" }
func (unavailableEngine) Estimate(context.Context, float64, float64, odds.Selection) (predports.Estimate, error) {
	return predports.Estimate{}, errors.New("missing expected goals")
}

type provider struct{ state ports.State }

func (p provider) Fetch(context.Context, time.Time, string) (ports.State, error) { return p.state, nil }

type summary struct{}

func (summary) Reasons(context.Context, string) ([]string, error) {
	return []string{"ข้อมูลไม่ครบ"}, nil
}
func TestUnavailableInputsCreatePassWithNullMetrics(t *testing.T) {
	now := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	m := match.Match{ID: "m", Status: "scheduled", Kickoff: now.Add(time.Hour)}
	state := ports.State{Matches: []match.Match{m}, Odds: []odds.Snapshot{{ID: "o", MatchID: "m", Selection: odds.Selection{Market: "1X2", Side: "home"}, Odds: 2, CapturedAt: now, Phase: "manual", SourceKey: "s"}}}
	store := &memoryStore{}
	s := New(store, fixedClock{now}, ids{}, unavailableEngine{}, provider{state}, summary{}, rec.Rules{PlayEV: .05, MaxAge: 15 * time.Minute}, "Asia/Bangkok")
	if err := s.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	r := store.state.Recommendations[0]
	if r.Status != "PASS" || r.Prediction != nil || r.Odds != nil || r.PredictionID != nil || r.ReasonCode != "MISSING_INPUTS" {
		t.Fatal(r)
	}
	if len(store.state.Picks) != 0 {
		t.Fatal("auto-selected recommendation")
	}
}
func TestCurrentEligibilityDoesNotMutateHistory(t *testing.T) {
	now := time.Now()
	old := rec.Recommendation{Status: "PLAY", Rules: rec.Rules{PlayEV: .05, MaxAge: 15 * time.Minute}, Odds: &odds.Snapshot{CapturedAt: now.Add(-time.Hour)}, Prediction: &prediction.Prediction{EV: .1}}
	svc := &Service{clock: fixedClock{now}}
	c := svc.currentCard(Card{Match: match.Match{Status: "scheduled", Kickoff: now.Add(time.Hour)}, Recommendation: &old})
	if c.Recommendation.Status != "PASS" || old.Status != "PLAY" {
		t.Fatal("historical state changed")
	}
}
