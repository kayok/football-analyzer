package domain

import (
	match "football/internal/match/domain"
	prediction "football/internal/prediction/domain"
	pick "football/internal/userpick/domain"
	"time"
)

type Stats struct {
	Profit  float64            `json:"profit_units"`
	Stake   float64            `json:"settled_stake_units"`
	ROI     *float64           `json:"roi"`
	Settled int                `json:"settled_picks"`
	Brier   map[string]float64 `json:"brier_by_model"`
}

func Calculate(picks []pick.Pick, predictions []prediction.Prediction, matches []match.Match, results []match.Result) Stats {
	s := Stats{Brier: map[string]float64{}}
	for _, p := range picks {
		if p.CancelledAt != nil || p.SettledAt == nil || p.Result == nil || *p.Result == "void" {
			continue
		}
		s.Stake += p.Stake
		if p.Profit != nil {
			s.Profit += *p.Profit
		}
		s.Settled++
	}
	if s.Stake > 0 {
		roi := s.Profit / s.Stake
		s.ROI = &roi
	}
	ms := map[string]match.Match{}
	rs := map[string]match.Result{}
	for _, m := range matches {
		ms[m.ID] = m
	}
	for _, r := range results {
		rs[r.MatchID] = r
	}
	latest := map[string]prediction.Prediction{}
	for _, p := range predictions {
		m, ok := ms[p.MatchID]
		if !ok || !p.GeneratedAt.Before(m.Kickoff) {
			continue
		}
		r, ok := rs[p.MatchID]
		if !ok || r.Status != "finished" {
			continue
		}
		key := p.MatchID + "/" + p.ModelVersion
		old, ok := latest[key]
		if !ok || p.GeneratedAt.After(old.GeneratedAt) || (p.GeneratedAt.Equal(old.GeneratedAt) && p.ID > old.ID) {
			latest[key] = p
		}
	}
	counts := map[string]int{}
	for _, p := range latest {
		r := rs[p.MatchID]
		winner := 1
		if r.Home > r.Away {
			winner = 0
		} else if r.Away > r.Home {
			winner = 2
		}
		for i, v := range p.OneXTwo {
			y := 0.0
			if i == winner {
				y = 1
			}
			s.Brier[p.ModelVersion] += (v - y) * (v - y)
		}
		counts[p.ModelVersion]++
	}
	for model, n := range counts {
		s.Brier[model] /= float64(n)
	}
	return s
}
func DateBounds(date, timezone string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start.UTC(), start.AddDate(0, 0, 1).UTC(), nil
}
