package engine

import (
	"context"
	odds "football/internal/odds/domain"
	"math"
	"testing"
)

func TestPoissonSymmetryAndNormalization(t *testing.T) {
	p, err := (Poisson{}).Estimate(context.Background(), 1.5, 1.5, odds.Selection{Market: "1X2", Side: "home"})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(p.OneXTwo[0]-p.OneXTwo[2]) > 1e-12 {
		t.Fatal("symmetric strengths must be symmetric")
	}
	if math.Abs(p.OneXTwo[0]+p.OneXTwo[1]+p.OneXTwo[2]-1) > 1e-12 {
		t.Fatal("not normalized")
	}
	if p.OneXTwo[1] < .2 || p.OneXTwo[1] > .3 {
		t.Fatal("unexpected draw probability")
	}
}
func TestMassTruncation(t *testing.T) {
	for _, lambda := range []float64{.01, 1.5, 8, 100} {
		ps, err := masses(lambda)
		if err != nil {
			t.Fatal(err)
		}
		sum := 0.0
		for _, p := range ps {
			sum += p
		}
		if 1-sum > 2.01e-10 {
			t.Fatal("tail exceeds bound")
		}
	}
}
func TestInvalidInputs(t *testing.T) {
	for _, v := range []float64{0, -1, math.NaN(), math.Inf(1), 101} {
		if _, err := (Poisson{}).Estimate(context.Background(), v, 1, odds.Selection{Market: "1X2", Side: "home"}); err == nil {
			t.Fatalf("accepted %v", v)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Poisson{}).Estimate(ctx, 1, 1, odds.Selection{Market: "1X2", Side: "home"}); err == nil {
		t.Fatal("ignored cancelled context")
	}
}
