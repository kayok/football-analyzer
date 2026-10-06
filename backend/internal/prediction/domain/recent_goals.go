package domain

import (
	"errors"
	"time"
)

// HistoricalScore is a completed regulation-time game, observed before prediction time.
type HistoricalScore struct {
	HomeID, AwayID      string
	Home, Away          int
	Kickoff, ObservedAt time.Time
}

// RecentGoals is a transparent baseline, not measured xG. It averages the selected
// team's venue-specific attack and the opponent's defence, shrunk toward the league
// average by two games. Three venue-specific games per team are required.
func RecentGoals(homeID, awayID string, cutoff time.Time, scores []HistoricalScore) (float64, float64, error) {
	var n, hN, aN int
	var leagueH, leagueA, hFor, hAgainst, aFor, aAgainst float64
	for _, s := range scores {
		if !s.Kickoff.Before(cutoff) || s.ObservedAt.After(cutoff) || s.Home < 0 || s.Away < 0 {
			continue
		}
		n++
		leagueH += float64(s.Home)
		leagueA += float64(s.Away)
		if s.HomeID == homeID {
			hN++
			hFor += float64(s.Home)
			hAgainst += float64(s.Away)
		}
		if s.AwayID == awayID {
			aN++
			aFor += float64(s.Away)
			aAgainst += float64(s.Home)
		}
	}
	if n == 0 || hN < 3 || aN < 3 {
		return 0, 0, errors.New("not enough historical home/away results")
	}
	lh, la := leagueH/float64(n), leagueA/float64(n)
	hAttack := (hFor + 2*lh) / float64(hN+2)
	aDefence := (aAgainst + 2*lh) / float64(aN+2)
	aAttack := (aFor + 2*la) / float64(aN+2)
	hDefence := (hAgainst + 2*la) / float64(hN+2)
	h, a := (hAttack+aDefence)/2, (aAttack+hDefence)/2
	if h <= 0 || a <= 0 {
		return 0, 0, errors.New("historical expected goals unavailable")
	}
	return h, a, nil
}
