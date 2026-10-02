package postgres

import (
	lineup "football/internal/lineup/domain"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	pred "football/internal/prediction/domain"
	rec "football/internal/recommendation/domain"
)

type matchType = match.Match
type resultType = match.Result
type oddsType = odds.Snapshot
type lineupType = lineup.Snapshot
type predType = pred.Prediction
type recType = rec.Recommendation
