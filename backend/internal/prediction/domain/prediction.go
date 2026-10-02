package domain

import (
	"errors"
	odds "football/internal/odds/domain"
	"math"
	"time"
)

type Distribution struct {
	Win      float64 `json:"p_win"`
	HalfWin  float64 `json:"p_half_win"`
	Push     float64 `json:"p_push"`
	HalfLoss float64 `json:"p_half_loss"`
	Loss     float64 `json:"p_loss"`
}

func (d Distribution) Validate() error {
	sum := 0.0
	for _, p := range []float64{d.Win, d.HalfWin, d.Push, d.HalfLoss, d.Loss} {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
			return errors.New("invalid probability")
		}
		sum += p
	}
	if math.Abs(sum-1) > 1e-9 {
		return errors.New("probabilities must sum to one")
	}
	return nil
}
func (d Distribution) Value(o, threshold float64) (float64, *float64, *float64, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, nil, err
	}
	if _, err := odds.Implied(o); err != nil {
		return 0, nil, nil, err
	}
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 {
		return 0, nil, nil, errors.New("invalid threshold")
	}
	a := d.Win + d.HalfWin/2
	b := d.Loss + d.HalfLoss/2
	ev := a*(o-1) - b
	if math.IsInf(ev, 0) || math.IsNaN(ev) {
		return 0, nil, nil, errors.New("non-finite EV")
	}
	if a == 0 {
		return ev, nil, nil, nil
	}
	fair := 1 + b/a
	min := 1 + (b+threshold)/a
	if math.IsInf(fair, 0) || math.IsInf(min, 0) {
		return ev, nil, nil, nil
	}
	return ev, &fair, &min, nil
}

// Settlement returns -1, -.5, 0, .5 or 1: the stake fraction lost or won.
func Settlement(s odds.Selection, home, away int) (float64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if home < 0 || away < 0 {
		return 0, errors.New("negative score")
	}
	if s.Market == "1X2" {
		win := s.Side == "home" && home > away || s.Side == "draw" && home == away || s.Side == "away" && away > home
		if win {
			return 1, nil
		}
		return -1, nil
	}
	line := *s.Line
	if math.Abs(line*2-math.Round(line*2)) > 1e-9 {
		low, high := math.Floor(line*2)/2, math.Ceil(line*2)/2
		s.Line = &low
		a, _ := Settlement(s, home, away)
		s.Line = &high
		b, _ := Settlement(s, home, away)
		return (a + b) / 2, nil
	}
	v := float64(home-away) + line
	if s.Market == "AH" && s.Side == "away" {
		v = float64(away-home) + line
	}
	if s.Market == "OU" {
		v = float64(home+away) - line
		if s.Side == "under" {
			v = -v
		}
	}
	if v > 0 {
		return 1, nil
	}
	if v < 0 {
		return -1, nil
	}
	return 0, nil
}
func Profit(result, decimalOdds float64) float64 {
	if result > 0 {
		return result * (decimalOdds - 1)
	}
	return result
}

type Prediction struct {
	ID      string `json:"id"`
	MatchID string `json:"match_id"`
	OddsID  string `json:"odds_snapshot_id"`
	odds.Selection
	Distribution Distribution `json:"distribution"`
	Probability  float64      `json:"probability"`
	EV           float64      `json:"ev"`
	FairOdds     *float64     `json:"fair_odds"`
	MinimumOdds  *float64     `json:"minimum_acceptable_odds"`
	Implied      float64      `json:"market_implied_probability"`
	Normalized   *float64     `json:"margin_removed_probability"`
	OneXTwo      [3]float64   `json:"one_x_two"`
	ExpectedHome float64      `json:"expected_goals_home"`
	ExpectedAway float64      `json:"expected_goals_away"`
	ModelVersion string       `json:"model_version"`
	GeneratedAt  time.Time    `json:"generated_at"`
}
