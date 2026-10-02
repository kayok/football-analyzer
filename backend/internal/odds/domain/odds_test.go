package domain

import (
	"math"
	"testing"
)

func TestMargin(t *testing.T) {
	q, err := RemoveMargin([3]float64{2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(q[0]-6.0/13) > 1e-12 || math.Abs(q[0]+q[1]+q[2]-1) > 1e-12 {
		t.Fatal(q)
	}
	if _, err := Implied(1); err == nil {
		t.Fatal("invalid odds")
	}
}
