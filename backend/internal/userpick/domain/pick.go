package domain

import (
	"errors"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	rec "football/internal/recommendation/domain"
	"time"
)

type Pick struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	RecommendationID string `json:"recommendation_id"`
	MatchID          string `json:"match_id"`
	odds.Selection
	Odds         float64                 `json:"odds_at_pick"`
	Probability  float64                 `json:"probability_at_pick"`
	Distribution prediction.Distribution `json:"settlement_probabilities_at_pick"`
	EV           float64                 `json:"ev_at_pick"`
	Bookmaker    string                  `json:"bookmaker"`
	ModelVersion string                  `json:"model_version"`
	Stake        float64                 `json:"stake_units"`
	PickedAt     time.Time               `json:"picked_at"`
	CancelledAt  *time.Time              `json:"cancelled_at"`
	SettledAt    *time.Time              `json:"settled_at"`
	Result       *string                 `json:"result"`
	Profit       *float64                `json:"net_profit_units"`
	Demo         bool                    `json:"demo"`
}

func Select(id string, now time.Time, m match.Match, r rec.Recommendation) (Pick, error) {
	if (r.Status != "PLAY" && r.Status != "WATCH") || r.Odds == nil || r.Prediction == nil {
		return Pick{}, errors.New("recommendation is not selectable")
	}
	status, _ := r.Rules.Status(now, m, *r.Odds, r.Prediction.EV)
	if status == "PASS" {
		return Pick{}, errors.New("recommendation expired")
	}
	selection := r.Odds.Selection
	if selection.Line != nil {
		line := *selection.Line
		selection.Line = &line
	}
	return Pick{ID: id, RecommendationID: r.ID, MatchID: m.ID, Selection: selection, Odds: r.Odds.Odds, Probability: r.Prediction.Probability, Distribution: r.Prediction.Distribution, EV: r.Prediction.EV, Bookmaker: r.Odds.Bookmaker, ModelVersion: r.ModelVersion, Stake: 1, PickedAt: now}, nil
}
func (p Pick) Cancel(now time.Time) (Pick, error) {
	if p.SettledAt != nil {
		return p, errors.New("settled picks cannot be cancelled")
	}
	if p.CancelledAt == nil {
		p.CancelledAt = &now
	}
	return p, nil
}
func (p Pick) Settle(now time.Time, r match.Result) (Pick, error) {
	if p.CancelledAt != nil || p.SettledAt != nil {
		return p, nil
	}
	if r.Status != "finished" && r.Status != "cancelled" && r.Status != "abandoned" {
		return p, nil
	}
	result := "void"
	profit := 0.0
	if r.Status == "finished" {
		v, err := prediction.Settlement(p.Selection, r.Home, r.Away)
		if err != nil {
			return p, err
		}
		profit = prediction.Profit(v, p.Odds) * p.Stake
		switch v {
		case 1:
			result = "win"
		case .5:
			result = "half-win"
		case 0:
			result = "push"
		case -.5:
			result = "half-loss"
		case -1:
			result = "loss"
		}
	}
	p.Result = &result
	p.Profit = &profit
	p.SettledAt = &now
	return p, nil
}
