package footballapi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"football/internal/application/ports"
	lineup "football/internal/lineup/domain"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	engineports "football/internal/prediction/ports"
	"time"
)

func ID(key string) string {
	b := sha256.Sum256([]byte(key))
	b[6] = (b[6] & 15) | 80
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type MockFootballProvider struct{ Engine engineports.Engine }

func (p MockFootballProvider) Fetch(ctx context.Context, now time.Time, tz string) (ports.State, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return ports.State{}, err
	}
	local := now.In(loc)
	date := local.Format("2006-01-02")
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	st := ports.State{}
	fixtures := []struct {
		home, away, league, market, side string
		line                             *float64
		ev, h, a                         float64
		done                             bool
		scoreH, scoreA                   int
	}{
		{"Liverpool", "Arsenal", "Premier League", "1X2", "home", nil, .12, 1.8, 1.1, false, 0, 0},
		{"Barcelona", "Atlético Madrid", "La Liga", "AH", "home", ptr(-.25), .025, 1.7, 1.2, false, 0, 0},
		{"Inter", "Juventus", "Serie A", "OU", "over", ptr(2.25), -.04, 1.4, 1.1, false, 0, 0},
		{"Bayern Munich", "Dortmund", "Bundesliga", "AH", "home", ptr(-.75), .12, 2, 1.1, true, 1, 0},
		{"PSG", "Lyon", "Ligue 1", "OU", "over", ptr(2), .08, 1.9, 1.2, true, 1, 1},
	}
	for i, f := range fixtures {
		fixtureDate := date
		kick := midnight.Add(time.Duration(18+i) * time.Hour)
		if f.done {
			kick = midnight.AddDate(0, 0, -1).Add(time.Duration(18+i-3) * time.Hour)
			fixtureDate = kick.Format("2006-01-02")
		} else if !kick.After(now.Add(15 * time.Minute)) {
			kick = now.Add(time.Duration(i+1) * 20 * time.Minute)
			end := midnight.AddDate(0, 0, 1)
			if !kick.Before(end) {
				kick = end.Add(-time.Duration(3-i) * time.Second)
			}
		}
		external := fixtureDate + "/" + f.home + "/" + f.away
		m := match.Match{ID: ID("match/" + external), CompetitionID: ID("league/" + f.league), Competition: f.league, HomeID: ID("team/" + f.home), AwayID: ID("team/" + f.away), Home: f.home, Away: f.away, Kickoff: kick.UTC(), Status: "scheduled", Provider: "mock", ExternalID: external, ExpectedHome: f.h, ExpectedAway: f.a}
		if f.done {
			m.Status = "finished"
		}
		st.Matches = append(st.Matches, m)
		selections := []odds.Selection{{Market: "1X2", Side: "home"}, {Market: "1X2", Side: "draw"}, {Market: "1X2", Side: "away"}, {Market: "AH", Side: "home", Line: ptr(-.25)}, {Market: "AH", Side: "home", Line: ptr(-.75)}, {Market: "AH", Side: "away", Line: ptr(.25)}, {Market: "OU", Side: "over", Line: ptr(2)}, {Market: "OU", Side: "over", Line: ptr(2.25)}, {Market: "OU", Side: "under", Line: ptr(2.5)}}
		phases := []string{"T-60", "T-30", "T-10", "manual"}
		if f.done {
			phases = []string{"T-60", "T-30", "T-10", "closing"}
		}
		for phaseIndex, phase := range phases {
			captured := now.UTC().Truncate(time.Minute)
			if phase != "manual" {
				minutes := []int{60, 30, 10, 1}[phaseIndex]
				captured = m.Kickoff.Add(-time.Duration(minutes) * time.Minute)
				if !f.done && captured.After(now) {
					continue
				}
			}
			for _, sel := range selections {
				estimate, err := p.Engine.Estimate(ctx, f.h, f.a, sel)
				if err != nil {
					return st, err
				}
				d := estimate.Distribution
				a, b := d.Win+d.HalfWin/2, d.Loss+d.HalfLoss/2
				target := -.07
				if sel.Market == f.market && sel.Side == f.side && same(sel.Line, f.line) {
					target = f.ev
				}
				price := 1 + (b+target)/a
				if price <= 1 {
					price = 1.01
				}
				if phase == "T-60" {
					price -= .01
				}
				if price <= 1 {
					price = 1.01
				}
				source := fmt.Sprintf("%s/%s/%s/%v/%s/%s", m.ID, sel.Market, sel.Side, lineString(sel.Line), phase, captured.Format(time.RFC3339))
				st.Odds = append(st.Odds, odds.Snapshot{ID: ID(source), MatchID: m.ID, Selection: sel, Bookmaker: "MockBook", Odds: price, CapturedAt: captured, Phase: phase, SourceKey: source})
			}
		}
		captured := now.UTC().Truncate(time.Minute)
		phase := "manual"
		if f.done {
			captured = m.Kickoff.Add(-10 * time.Minute)
			phase = "T-10"
		}
		source := m.ID + "/lineup/" + captured.Format(time.RFC3339)
		st.Lineups = append(st.Lineups, lineup.Snapshot{ID: ID(source), MatchID: m.ID, Home: team(f.home), Away: team(f.away), CapturedAt: captured, Phase: phase, SourceKey: source})
		if f.done {
			st.Results = append(st.Results, match.Result{MatchID: m.ID, Home: f.scoreH, Away: f.scoreA, Status: "finished", RecordedAt: m.Kickoff.Add(2 * time.Hour)})
		}
	}
	return st, nil
}
func ptr(v float64) *float64 { return &v }
func same(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func lineString(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *v)
}
func team(name string) lineup.Team {
	t := lineup.Team{Starting: []string{}, Substitutes: []string{}, Injuries: []string{"ผู้เล่นจำลอง — บาดเจ็บกล้ามเนื้อ"}, Suspensions: []string{}}
	for i := 1; i <= 11; i++ {
		t.Starting = append(t.Starting, fmt.Sprintf("%s · ผู้เล่น %02d", name, i))
	}
	for i := 12; i <= 16; i++ {
		t.Substitutes = append(t.Substitutes, fmt.Sprintf("%s · ผู้เล่น %02d", name, i))
	}
	return t
}
