package domain

import (
	"math"
	"testing"
	"time"
)

func TestRecentGoalsRequiresHistoricalVenueSamples(t *testing.T) {
	now := time.Now()
	scores := []HistoricalScore{}
	for i := 0; i < 3; i++ {
		scores = append(scores, HistoricalScore{HomeID: "home", AwayID: "away", Home: 2, Away: 1, Kickoff: now.Add(-48 * time.Hour), ObservedAt: now.Add(-time.Hour)})
	}
	h, a, err := RecentGoals("home", "away", now, scores)
	if err != nil || math.Abs(h-2) > 1e-9 || math.Abs(a-1) > 1e-9 {
		t.Fatal(h, a, err)
	}
	future := HistoricalScore{HomeID: "home", AwayID: "away", Home: 99, Away: 99, Kickoff: now.Add(time.Hour), ObservedAt: now.Add(-time.Minute)}
	scores = append(scores, future)
	h, a, err = RecentGoals("home", "away", now, scores)
	if err != nil || h != 2 || a != 1 {
		t.Fatal("future result leaked", h, a, err)
	}
	scores[0].ObservedAt = now.Add(time.Hour)
	if _, _, err = RecentGoals("home", "away", now, scores); err == nil {
		t.Fatal("unobserved result included")
	}
	if _, _, err = RecentGoals("different-home", "away", now, scores); err == nil {
		t.Fatal("missing venue sample accepted")
	}
}
