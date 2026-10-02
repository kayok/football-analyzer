package domain

import (
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	"testing"
	"time"
)

func TestBoundaries(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := match.Match{Status: "scheduled", Kickoff: now.Add(time.Hour)}
	o := odds.Snapshot{CapturedAt: now}
	r := Rules{.05, 15 * time.Minute}
	for _, tc := range []struct {
		ev   float64
		want string
	}{{.05, "PLAY"}, {.0499, "WATCH"}, {0, "PASS"}, {-.1, "PASS"}} {
		v, _ := r.Status(now, m, o, tc.ev)
		if v != tc.want {
			t.Fatal(v)
		}
	}
	o.CapturedAt = now.Add(-15 * time.Minute)
	v, _ := r.Status(now, m, o, .1)
	if v != "PLAY" {
		t.Fatal("age boundary")
	}
	o.CapturedAt = o.CapturedAt.Add(-time.Nanosecond)
	v, _ = r.Status(now, m, o, .1)
	if v != "PASS" {
		t.Fatal("stale odds")
	}
	o.CapturedAt = now
	m.Kickoff = now
	v, _ = r.Status(now, m, o, .1)
	if v != "PASS" {
		t.Fatal("kickoff boundary")
	}
}
