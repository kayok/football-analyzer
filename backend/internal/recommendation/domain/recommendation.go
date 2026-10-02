package domain

import (
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	"time"
)

type Rules struct {
	PlayEV float64       `json:"play_ev_threshold"`
	MaxAge time.Duration `json:"odds_max_age_ns"`
}

func (r Rules) Status(now time.Time, m match.Match, o odds.Snapshot, ev float64) (string, string) {
	if m.Status != "scheduled" || !now.Before(m.Kickoff) {
		return "PASS", "MATCH_STARTED"
	}
	if o.CapturedAt.After(now) || now.Sub(o.CapturedAt) > r.MaxAge {
		return "PASS", "STALE_ODDS"
	}
	if ev >= r.PlayEV {
		return "PLAY", "VALUE"
	}
	if ev > 0 {
		return "WATCH", "SMALL_VALUE"
	}
	return "PASS", "NO_VALUE"
}

type Recommendation struct {
	Generation   int                    `json:"generation"`
	ID           string                 `json:"id"`
	MatchID      string                 `json:"match_id"`
	OddsID       *string                `json:"odds_snapshot_id"`
	PredictionID *string                `json:"prediction_id"`
	Status       string                 `json:"status"`
	ReasonCode   string                 `json:"reason_code"`
	Reasons      []string               `json:"reasons"`
	ModelVersion string                 `json:"model_version"`
	RuleVersion  string                 `json:"rule_version"`
	Rules        Rules                  `json:"rules"`
	GeneratedAt  time.Time              `json:"generated_at"`
	Odds         *odds.Snapshot         `json:"odds"`
	Prediction   *prediction.Prediction `json:"prediction"`
}
