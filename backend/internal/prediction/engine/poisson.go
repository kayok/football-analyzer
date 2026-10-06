package engine

import (
	"context"
	"errors"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	"football/internal/prediction/ports"
	"math"
)

type Poisson struct{ ModelVersion string }

func (p Poisson) Version() string {
	if p.ModelVersion != "" {
		return p.ModelVersion
	}
	return "v1-mock-poisson"
}
func masses(lambda float64) ([]float64, error) {
	if math.IsNaN(lambda) || math.IsInf(lambda, 0) || lambda <= 0 {
		return nil, errors.New("invalid expected goals")
	}
	// Football goals outside this numerical range are rejected rather than hanging.
	if lambda > 100 {
		return nil, errors.New("expected goals exceeds supported range (100)")
	}
	p := math.Exp(-lambda)
	out := []float64{p}
	sum := p
	for n := 1; 1-sum > 2e-10; n++ {
		if n > 1000 {
			return nil, errors.New("Poisson did not converge")
		}
		p *= lambda / float64(n)
		out = append(out, p)
		sum += p
	}
	return out, nil
}
func (Poisson) Estimate(ctx context.Context, h, a float64, s odds.Selection) (ports.Estimate, error) {
	var out ports.Estimate
	if err := s.Validate(); err != nil {
		return out, err
	}
	hp, err := masses(h)
	if err != nil {
		return out, err
	}
	ap, err := masses(a)
	if err != nil {
		return out, err
	}
	total := 0.0
	for i, pi := range hp {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		for j, pj := range ap {
			p := pi * pj
			total += p
			k := 1
			if i > j {
				k = 0
			} else if j > i {
				k = 2
			}
			out.OneXTwo[k] += p
			result, _ := prediction.Settlement(s, i, j)
			switch result {
			case 1:
				out.Distribution.Win += p
			case .5:
				out.Distribution.HalfWin += p
			case 0:
				out.Distribution.Push += p
			case -.5:
				out.Distribution.HalfLoss += p
			case -1:
				out.Distribution.Loss += p
			}
		}
	}
	out.Distribution.Win /= total
	out.Distribution.HalfWin /= total
	out.Distribution.Push /= total
	out.Distribution.HalfLoss /= total
	out.Distribution.Loss /= total
	for i := range out.OneXTwo {
		out.OneXTwo[i] /= total
	}
	return out, out.Distribution.Validate()
}
