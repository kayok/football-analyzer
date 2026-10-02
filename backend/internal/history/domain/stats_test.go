package domain

import (
	match "football/internal/match/domain"
	pred "football/internal/prediction/domain"
	pick "football/internal/userpick/domain"
	"math"
	"testing"
	"time"
)

func TestDateBoundsDST(t *testing.T) {
	a, b, err := DateBounds("2026-03-08", "America/New_York")
	if err != nil || b.Sub(a) != 23*time.Hour {
		t.Fatal(a, b, err)
	}
	a, b, err = DateBounds("2026-01-01", "Asia/Bangkok")
	if err != nil || a.Hour() != 17 || b.Sub(a) != 24*time.Hour {
		t.Fatal(a, b, err)
	}
}
func TestROIAndBrier(t *testing.T) {
	now := time.Now()
	win, push, void := "win", "push", "void"
	profit, zero := 1.0, 0.0
	ps := []pick.Pick{{Stake: 1, SettledAt: &now, Result: &win, Profit: &profit}, {Stake: 1, SettledAt: &now, Result: &push, Profit: &zero}, {Stake: 1, SettledAt: &now, Result: &void, Profit: &zero}, {Stake: 1, CancelledAt: &now, SettledAt: &now, Result: &win, Profit: &profit}, {Stake: 1}}
	ms := []match.Match{{ID: "m", Kickoff: now}}
	rs := []match.Result{{MatchID: "m", Home: 1, Status: "finished"}}
	preds := []pred.Prediction{{ID: "a", MatchID: "m", GeneratedAt: now.Add(-time.Hour), ModelVersion: "v1", OneXTwo: [3]float64{1, 0, 0}}, {ID: "b", MatchID: "m", GeneratedAt: now.Add(-time.Minute), ModelVersion: "v1", OneXTwo: [3]float64{.5, .3, .2}}, {ID: "c", MatchID: "m", GeneratedAt: now.Add(time.Minute), ModelVersion: "v1", OneXTwo: [3]float64{1, 0, 0}}}
	s := Calculate(ps, preds, ms, rs)
	if s.ROI == nil || *s.ROI != .5 || s.Settled != 2 || math.Abs(s.Brier["v1"]-.38) > 1e-12 {
		t.Fatal(s)
	}
	if Calculate(nil, nil, nil, nil).ROI != nil {
		t.Fatal("empty ROI")
	}
}
