package domain

import "time"

type Team struct {
	Starting    []string `json:"starting_xi"`
	Substitutes []string `json:"substitutes"`
	Injuries    []string `json:"injuries"`
	Suspensions []string `json:"suspensions"`
}
type Snapshot struct {
	ID         string    `json:"id"`
	MatchID    string    `json:"match_id"`
	Home       Team      `json:"home"`
	Away       Team      `json:"away"`
	CapturedAt time.Time `json:"captured_at"`
	Phase      string    `json:"phase"`
	SourceKey  string    `json:"source_key"`
}
