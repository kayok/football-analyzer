package domain

import (
	odds "football/internal/odds/domain"
	"math"
	"testing"
)

func TestValues(t *testing.T) {
	for _, tc := range []struct {
		d        Distribution
		ev, fair float64
	}{{Distribution{Win: .56, Loss: .44}, .12, 1 / .56}, {Distribution{Win: .5, Push: .1, Loss: .4}, .1, 1.8}, {Distribution{Win: .3, HalfWin: .2, Push: .1, HalfLoss: .2, Loss: .2}, .1, 1.75}} {
		ev, fair, min, err := tc.d.Value(2, .05)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(ev-tc.ev) > 1e-12 || math.Abs(*fair-tc.fair) > 1e-12 {
			t.Fatalf("wrong EV/fair: %v %v", ev, *fair)
		}
		atMin, _, _, _ := tc.d.Value(*min, .05)
		if math.Abs(atMin-.05) > 1e-12 {
			t.Fatal("wrong minimum odds")
		}
	}
	_, fair, _, _ := (Distribution{Loss: 1}).Value(2, .05)
	if fair != nil {
		t.Fatal("infinite fair odds must be null")
	}
}
func TestQuarterSettlement(t *testing.T) {
	for _, tc := range []struct {
		market, side string
		line         float64
		h, a         int
		want         float64
	}{{"AH", "home", -.25, 0, 0, -.5}, {"AH", "home", .25, 0, 0, .5}, {"AH", "away", -.75, 0, 1, .5}, {"AH", "home", -1, 1, 0, 0}, {"OU", "over", 2.25, 1, 1, -.5}, {"OU", "under", 2.25, 1, 1, .5}, {"OU", "over", 2.75, 2, 1, .5}, {"OU", "under", 2.75, 2, 1, -.5}} {
		v, err := Settlement(odds.Selection{Market: tc.market, Side: tc.side, Line: &tc.line}, tc.h, tc.a)
		if err != nil || v != tc.want {
			t.Fatalf("%+v got %v %v", tc, v, err)
		}
	}
}
func TestInvalidDistribution(t *testing.T) {
	for _, d := range []Distribution{{Win: .5}, {Win: math.NaN()}, {Win: 1.1, Loss: -.1}} {
		if _, _, _, err := d.Value(2, .05); err == nil {
			t.Fatal("invalid probabilities accepted")
		}
	}
	if _, _, _, err := (Distribution{Win: 1}).Value(math.Inf(1), .05); err == nil {
		t.Fatal("invalid odds accepted")
	}
}
