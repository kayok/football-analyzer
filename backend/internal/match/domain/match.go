package domain

import "time"

type Match struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	Competition   string    `json:"competition"`
	HomeID        string    `json:"home_id"`
	AwayID        string    `json:"away_id"`
	Home          string    `json:"home"`
	Away          string    `json:"away"`
	Kickoff       time.Time `json:"kickoff"`
	Status        string    `json:"status"`
	Provider      string    `json:"provider_name"`
	ExternalID    string    `json:"external_id"`
	ExpectedHome  float64   `json:"expected_goals_home"`
	ExpectedAway  float64   `json:"expected_goals_away"`
}
type Result struct {
	MatchID    string    `json:"match_id"`
	Home       int       `json:"home"`
	Away       int       `json:"away"`
	Status     string    `json:"status"`
	RecordedAt time.Time `json:"recorded_at"`
}
