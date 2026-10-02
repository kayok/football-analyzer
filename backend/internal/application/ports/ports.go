package ports

import (
	"context"
	lineup "football/internal/lineup/domain"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	rec "football/internal/recommendation/domain"
	pick "football/internal/userpick/domain"
	"time"
)

type State struct {
	Matches         []match.Match           `json:"matches"`
	Odds            []odds.Snapshot         `json:"odds"`
	Lineups         []lineup.Snapshot       `json:"lineups"`
	Predictions     []prediction.Prediction `json:"predictions"`
	Recommendations []rec.Recommendation    `json:"recommendations"`
	Picks           []pick.Pick             `json:"picks"`
	Results         []match.Result          `json:"results"`
}

// Update serializes state transitions in a database transaction. Views use a consistent snapshot.
type Store interface {
	View(context.Context) (State, error)
	ViewDate(context.Context, time.Time, time.Time) (State, error)
	Update(context.Context, func(*State) error) error
	Health(context.Context) error
}
type Clock interface{ Now() time.Time }
type IDs interface{ New() string }
type FootballProvider interface {
	Fetch(context.Context, time.Time, string) (State, error)
}
type AISummaryProvider interface {
	Reasons(context.Context, string) ([]string, error)
}
