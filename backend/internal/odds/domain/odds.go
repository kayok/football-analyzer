package domain

import (
	"errors"
	"math"
	"time"
)

type Selection struct {
	Market string   `json:"market"`
	Side   string   `json:"selection"`
	Line   *float64 `json:"line"`
}

func (s Selection) Validate() error {
	switch s.Market {
	case "1X2":
		if s.Line != nil || (s.Side != "home" && s.Side != "draw" && s.Side != "away") {
			return errors.New("invalid 1X2 selection")
		}
	case "AH", "OU":
		if s.Line == nil || math.IsNaN(*s.Line) || math.IsInf(*s.Line, 0) || math.Abs(*s.Line*4-math.Round(*s.Line*4)) > 1e-9 {
			return errors.New("invalid quarter line")
		}
		if s.Market == "AH" && s.Side != "home" && s.Side != "away" {
			return errors.New("invalid AH selection")
		}
		if s.Market == "OU" && (s.Side != "over" && s.Side != "under" || *s.Line < 0) {
			return errors.New("invalid OU selection")
		}
	default:
		return errors.New("unsupported market")
	}
	return nil
}

type Snapshot struct {
	ID      string `json:"id"`
	MatchID string `json:"match_id"`
	Selection
	Bookmaker  string    `json:"bookmaker"`
	Odds       float64   `json:"odds"`
	CapturedAt time.Time `json:"captured_at"`
	Phase      string    `json:"phase"`
	SourceKey  string    `json:"source_key"`
}

func Implied(odds float64) (float64, error) {
	if math.IsNaN(odds) || math.IsInf(odds, 0) || odds <= 1 {
		return 0, errors.New("invalid decimal odds")
	}
	return 1 / odds, nil
}
func RemoveMargin(odds [3]float64) ([3]float64, error) {
	var p [3]float64
	sum := 0.0
	for i, o := range odds {
		v, err := Implied(o)
		if err != nil {
			return p, err
		}
		p[i] = v
		sum += v
	}
	for i := range p {
		p[i] /= sum
	}
	return p, nil
}
