package domain

import (
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	pred "football/internal/prediction/domain"
	rec "football/internal/recommendation/domain"
	"testing"
	"time"
)

func TestSnapshotAndCancellation(t *testing.T) {
	now := time.Now()
	m := match.Match{ID: "m", Status: "scheduled", Kickoff: now.Add(time.Hour)}
	r := rec.Recommendation{ID: "r", Status: "PLAY", Rules: rec.Rules{PlayEV: .05, MaxAge: 15 * time.Minute}, Odds: &odds.Snapshot{Selection: odds.Selection{Market: "1X2", Side: "home"}, Odds: 2, CapturedAt: now}, Prediction: &pred.Prediction{EV: .12, Probability: .56, Distribution: pred.Distribution{Win: .56, Loss: .44}}}
	p, err := Select("p", now, m, r)
	if err != nil {
		t.Fatal(err)
	}
	r.Odds.Odds = 3
	r.Prediction.EV = .8
	if p.Odds != 2 || p.EV != .12 {
		t.Fatal("snapshot mutated")
	}
	p, err = p.Cancel(now)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := p.Cancel(now.Add(time.Hour))
	if !again.CancelledAt.Equal(*p.CancelledAt) {
		t.Fatal("not idempotent")
	}
	settled, _ := p.Settle(now, match.Result{Status: "finished", Home: 1})
	if settled.SettledAt != nil {
		t.Fatal("cancelled pick settled")
	}
}
func TestHalfProfit(t *testing.T) {
	line := -.75
	now := time.Now()
	p := Pick{Selection: odds.Selection{Market: "AH", Side: "home", Line: &line}, Odds: 2.2, Stake: 1}
	p, err := p.Settle(now, match.Result{Status: "finished", Home: 1})
	if err != nil || *p.Result != "half-win" || *p.Profit < .5999 || *p.Profit > .6001 {
		t.Fatal(p, err)
	}
	if _, err = p.Cancel(now); err == nil {
		t.Fatal("settled cancellation")
	}
}
