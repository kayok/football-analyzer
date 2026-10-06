package usecase

import (
	"context"
	"errors"
	"fmt"
	"football/internal/application/ports"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	"football/internal/prediction/engine"
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

func TestRealSyncPreservesSnapshotsAndAcceptsRescheduledFinalResult(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	real := match.Match{ID: "real", Provider: "real-source", ExternalID: "10", CompetitionID: "league", HomeID: "home", AwayID: "away", Status: "scheduled", Kickoff: now.Add(time.Hour)}
	incoming := ports.State{Matches: []match.Match{real}, Odds: []odds.Snapshot{{ID: "real-quote", MatchID: real.ID, Selection: odds.Selection{Market: "1X2", Side: "home"}, Odds: 3, CapturedAt: now, Phase: "manual", SourceKey: "real-quote"}}}
	for i := 0; i < 3; i++ {
		historical := real
		historical.ID = fmt.Sprintf("past-%d", i)
		historical.Status = "finished"
		historical.Kickoff = now.Add(-time.Duration(i+2) * 24 * time.Hour)
		incoming.Matches = append(incoming.Matches, historical)
		incoming.Results = append(incoming.Results, match.Result{MatchID: historical.ID, Home: 2, Away: 1, Status: "finished", RecordedAt: now})
	}
	mock := match.Match{ID: "mock", Provider: "mock", Kickoff: real.Kickoff, Status: "scheduled"}
	store := &memoryStore{state: ports.State{Matches: []match.Match{mock}}}
	svc := New(store, fixedClock{now}, ids{}, engine.Poisson{ModelVersion: "v1-recent-goals-poisson"}, provider{incoming}, summary{}, rec.Rules{PlayEV: .05, MaxAge: 15 * time.Minute}, "UTC").WithProvider("real-source").WithRealData()
	if err := svc.Run(context.Background(), true); err == nil {
		t.Fatal("real mode allowed mock seed")
	}
	if err := svc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	cards, err := svc.Today(context.Background(), "UTC")
	if err != nil || len(cards) != 1 || cards[0].ID != real.ID {
		t.Fatal("sources mixed", cards, err)
	}
	if len(store.state.Predictions) != 1 || len(store.state.Recommendations) != 1 || len(store.state.Picks) != 0 {
		t.Fatal("retroactive predictions or auto picks")
	}
	pred := store.state.Predictions[0]
	if pred.ExpectedHome != 2 || pred.ExpectedAway != 1 || pred.ModelVersion != "v1-recent-goals-poisson" {
		t.Fatal("real baseline missing", pred)
	}
	selected, err := svc.Pick(context.Background(), store.state.Recommendations[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	real.Status = "finished"
	real.Kickoff = now.Add(2 * time.Hour)
	svc.clock = fixedClock{now.Add(4 * time.Hour)}
	svc.football = provider{ports.State{Matches: []match.Match{real}, Results: []match.Result{{MatchID: real.ID, Status: "finished", Home: 2, Away: 1, RecordedAt: now.Add(4 * time.Hour)}}}}
	if err := svc.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(store.state.Predictions) != 1 || store.state.Predictions[0].ExpectedHome != pred.ExpectedHome {
		t.Fatal("historical snapshot overwritten")
	}
	if store.state.Picks[0].Odds != selected.Odds || store.state.Picks[0].SettledAt == nil {
		t.Fatal("real pick not settled from snapshot")
	}
	for _, m := range store.state.Matches {
		if m.ID == real.ID && (!m.Kickoff.Equal(real.Kickoff) || m.Status != "finished") {
			t.Fatal("real status or kickoff frozen", m)
		}
	}
}
