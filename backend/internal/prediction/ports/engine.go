package ports

import (
	"context"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
)

type Estimate struct {
	Distribution prediction.Distribution
	OneXTwo      [3]float64
}
type Engine interface {
	Estimate(context.Context, float64, float64, odds.Selection) (Estimate, error)
	Version() string
}
