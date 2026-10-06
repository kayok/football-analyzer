package footballapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"football/internal/application/ports"
	lineup "football/internal/lineup/domain"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const APIName = "api-football"

// APIFootball keeps credentials server-side. There is no automatic mock fallback.
type APIFootball struct {
	Key         string
	BaseURL     string
	Client      *http.Client
	Leagues     []int
	Store       ports.Store
	MaxRequests int
	ReportUsage func(Usage)
}
type apiEnvelope struct {
	Errors   json.RawMessage              `json:"errors"`
	Response json.RawMessage              `json:"response"`
	Paging   struct{ Current, Total int } `json:"paging"`
}
type apiFixture struct {
	Fixture struct {
		ID     int
		Date   time.Time
		Status struct{ Short string }
	}
	League struct {
		ID, Season int
		Name       string
	}
	Teams struct {
		Home, Away struct {
			ID   int
			Name string
		}
	}
	Score struct{ Fulltime struct{ Home, Away *int } }
}
type apiOdds struct {
	Fixture    struct{ ID int }
	Update     time.Time
	Bookmakers []struct {
		ID   int
		Name string
		Bets []struct {
			ID     int
			Name   string
			Values []struct{ Value, Odd string }
		}
	}
}
type apiLineup struct {
	Team        struct{ ID int }
	StartXI     []struct{ Player struct{ Name string } }
	Substitutes []struct{ Player struct{ Name string } }
}
type apiInjury struct {
	Team   struct{ ID int }
	Player struct{ Name, Type, Reason string }
}
type apiRun struct {
	provider APIFootball
	calls    int
	usage    Usage
}

func (r *apiRun) get(ctx context.Context, path string, values url.Values, out any) (int, error) {
	if r.calls >= r.provider.MaxRequests {
		return 0, errors.New("API-Football sync request budget exceeded; reduce leagues or increase API_FOOTBALL_MAX_REQUESTS")
	}
	if r.usage.DailyRemaining != nil && *r.usage.DailyRemaining == 0 {
		return 0, errors.New("API-Football daily quota exhausted; wait for the provider quota reset")
	}
	if r.usage.MinuteRemaining != nil && *r.usage.MinuteRemaining == 0 {
		return 0, errors.New("API-Football minute quota exhausted; wait before running worker again")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", r.provider.BaseURL+path+"?"+values.Encode(), nil)
	if err != nil {
		return 0, errors.New("invalid API-Football request")
	}
	req.Header.Set("x-apisports-key", r.provider.Key)
	r.calls++
	r.usage.RequestsAttempted = r.calls
	r.usage.ByEndpoint[path]++
	res, err := r.provider.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("API-Football %s request failed (check network or timeout)", path)
	}
	defer res.Body.Close()
	r.usage.Responses++
	r.usage.DailyLimit = quotaHeader(res.Header, "x-ratelimit-requests-limit")
	r.usage.DailyRemaining = quotaHeader(res.Header, "x-ratelimit-requests-remaining")
	r.usage.MinuteLimit = quotaHeader(res.Header, "x-ratelimit-limit")
	r.usage.MinuteRemaining = quotaHeader(res.Header, "x-ratelimit-remaining")
	if res.StatusCode != 200 {
		return 0, fmt.Errorf("API-Football %s returned HTTP %d", path, res.StatusCode)
	}
	var env apiEnvelope
	if err = json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&env); err != nil {
		return 0, fmt.Errorf("API-Football %s returned invalid JSON", path)
	}
	raw := strings.TrimSpace(string(env.Errors))
	if raw != "" && raw != "null" && raw != "[]" && raw != "{}" {
		return 0, fmt.Errorf("API-Football %s reported an error; check your plan, quota and parameters in the provider dashboard", path)
	}
	if err = json.Unmarshal(env.Response, out); err != nil {
		return 0, fmt.Errorf("API-Football %s returned an unexpected response", path)
	}
	if env.Paging.Total > 20 {
		return 0, errors.New("API-Football response exceeds 20 pages; reduce query scope")
	}
	return env.Paging.Total, nil
}
func (r *apiRun) fixtures(ctx context.Context, query url.Values) ([]apiFixture, error) {
	var out []apiFixture
	for page := 1; ; page++ {
		if page > 1 {
			query.Set("page", strconv.Itoa(page))
		}
		var batch []apiFixture
		total, err := r.get(ctx, "/fixtures", query, &batch)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if page >= total {
			return out, nil
		}
	}
}
func (p APIFootball) Fetch(ctx context.Context, now time.Time, tz string) (ports.State, error) {
	r := apiRun{provider: p, usage: Usage{ByEndpoint: map[string]int{}}}
	defer func() {
		if p.ReportUsage != nil {
			p.ReportUsage(r.usage)
		}
	}()
	if strings.TrimSpace(p.Key) == "" {
		return ports.State{}, errors.New("API_FOOTBALL_KEY is required for real data")
	}
	if len(p.Leagues) == 0 {
		return ports.State{}, errors.New("API_FOOTBALL_LEAGUES must contain league IDs")
	}
	if p.BaseURL == "" {
		p.BaseURL = "https://v3.football.api-sports.io"
	}
	if p.Client == nil {
		p.Client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if p.MaxRequests <= 0 {
		p.MaxRequests = 50
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return ports.State{}, err
	}
	local := now.In(loc)
	date := local.Format("2006-01-02")
	r.provider = p
	selected := map[int]bool{}
	for _, id := range p.Leagues {
		selected[id] = true
	}
	today, err := r.fixtures(ctx, url.Values{"date": {date}, "timezone": {tz}})
	if err != nil {
		return ports.State{}, err
	}
	fixtures := map[int]apiFixture{}
	seasons := map[int]int{}
	for _, f := range today {
		if selected[f.League.ID] {
			fixtures[f.Fixture.ID] = f
			seasons[f.League.ID] = f.League.Season
		}
	}
	// Refresh persisted unfinished fixtures even after their original calendar date.
	if p.Store != nil {
		state, err := p.Store.View(ctx)
		if err != nil {
			return ports.State{}, err
		}
		pending := []string{}
		for _, m := range state.Matches {
			if m.Provider == APIName && m.Status != "finished" && m.Status != "cancelled" && m.Status != "abandoned" && !m.Kickoff.After(now) {
				id, err := strconv.Atoi(m.ExternalID)
				if err != nil {
					return ports.State{}, errors.New("invalid persisted API-Football fixture ID")
				}
				if _, ok := fixtures[id]; !ok {
					pending = append(pending, strconv.Itoa(id))
				}
			}
		}
		sort.Strings(pending)
		for len(pending) > 0 {
			n := len(pending)
			if n > 20 {
				n = 20
			}
			batch, err := r.fixtures(ctx, url.Values{"ids": {strings.Join(pending[:n], "-")}})
			if err != nil {
				return ports.State{}, err
			}
			for _, f := range batch {
				fixtures[f.Fixture.ID] = f
			}
			pending = pending[n:]
		}
	}
	st := ports.State{}
	// Real completed league games supply the baseline model; no bookmaker-derived xG.
	for _, league := range p.Leagues {
		season, ok := seasons[league]
		if !ok {
			continue
		}
		past, err := r.fixtures(ctx, url.Values{"league": {strconv.Itoa(league)}, "season": {strconv.Itoa(season)}, "from": {local.AddDate(0, 0, -180).Format("2006-01-02")}, "to": {local.AddDate(0, 0, -1).Format("2006-01-02")}, "status": {"FT"}, "timezone": {tz}})
		if err != nil {
			return ports.State{}, err
		}
		for _, f := range past {
			fixtures[f.Fixture.ID] = f
		}
	}
	ids := []int{}
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		f := fixtures[id]
		if f.Fixture.ID <= 0 || f.Teams.Home.ID <= 0 || f.Teams.Away.ID <= 0 || f.League.ID <= 0 || f.Fixture.Date.IsZero() {
			return st, errors.New("API-Football fixture is missing required IDs or kickoff")
		}
		m := match.Match{ID: ID(APIName + "/fixture/" + strconv.Itoa(id)), CompetitionID: ID(APIName + "/league/" + strconv.Itoa(f.League.ID)), Competition: f.League.Name, HomeID: ID(APIName + "/team/" + strconv.Itoa(f.Teams.Home.ID)), AwayID: ID(APIName + "/team/" + strconv.Itoa(f.Teams.Away.ID)), Home: f.Teams.Home.Name, Away: f.Teams.Away.Name, Kickoff: f.Fixture.Date.UTC(), Status: fixtureStatus(f.Fixture.Status.Short), Provider: APIName, ExternalID: strconv.Itoa(id)}
		st.Matches = append(st.Matches, m)
		if m.Status == "finished" && f.Score.Fulltime.Home != nil && f.Score.Fulltime.Away != nil && *f.Score.Fulltime.Home >= 0 && *f.Score.Fulltime.Away >= 0 {
			st.Results = append(st.Results, match.Result{MatchID: m.ID, Home: *f.Score.Fulltime.Home, Away: *f.Score.Fulltime.Away, Status: "finished", RecordedAt: now.UTC()})
		} else if m.Status == "cancelled" {
			st.Results = append(st.Results, match.Result{MatchID: m.ID, Status: "cancelled", RecordedAt: now.UTC()})
		}
		// Fetch prices only for pre-match fixtures on today's requested local date.
		if m.Status != "scheduled" || !m.Kickoff.After(now) || m.Kickoff.In(loc).Format("2006-01-02") != date {
			continue
		}
		for page := 1; ; page++ {
			var batches []apiOdds
			total, err := r.get(ctx, "/odds", url.Values{"fixture": {strconv.Itoa(id)}, "page": {strconv.Itoa(page)}}, &batches)
			if err != nil {
				return st, err
			}
			for _, batch := range batches {
				if batch.Fixture.ID != id || batch.Update.IsZero() || batch.Update.After(now.Add(time.Minute)) {
					continue
				}
				for _, book := range batch.Bookmakers {
					for _, bet := range book.Bets {
						for _, value := range bet.Values {
							sel, ok := apiSelection(bet.Name, value.Value)
							if !ok {
								continue
							}
							price, err := strconv.ParseFloat(value.Odd, 64)
							if err != nil {
								continue
							}
							if _, err = odds.Implied(price); err != nil {
								continue
							}
							source := fmt.Sprintf("%s/odds/%d/%d/%s/%s/%s", m.ID, book.ID, bet.ID, value.Value, value.Odd, batch.Update.UTC().Format(time.RFC3339Nano))
							st.Odds = append(st.Odds, odds.Snapshot{ID: ID(source), MatchID: m.ID, Selection: sel, Bookmaker: book.Name, Odds: price, CapturedAt: batch.Update.UTC(), Phase: "manual", SourceKey: source})
						}
					}
				}
			}
			if page >= total {
				break
			}
		}
		// Missing lineup data stays missing. Fetch within the pre-match window only.
		if m.Kickoff.Sub(now) > 90*time.Minute {
			continue
		}
		var lines []apiLineup
		if _, err := r.get(ctx, "/fixtures/lineups", url.Values{"fixture": {strconv.Itoa(id)}}, &lines); err != nil {
			return st, err
		}
		var injuries []apiInjury
		if _, err := r.get(ctx, "/injuries", url.Values{"fixture": {strconv.Itoa(id)}}, &injuries); err != nil {
			return st, err
		}
		if len(lines) == 0 && len(injuries) == 0 {
			continue
		}
		home, away := emptyTeam(), emptyTeam()
		for _, l := range lines {
			var t *lineup.Team
			if l.Team.ID == f.Teams.Home.ID {
				t = &home
			} else if l.Team.ID == f.Teams.Away.ID {
				t = &away
			} else {
				continue
			}
			for _, player := range l.StartXI {
				if player.Player.Name != "" {
					t.Starting = append(t.Starting, player.Player.Name)
				}
			}
			for _, player := range l.Substitutes {
				if player.Player.Name != "" {
					t.Substitutes = append(t.Substitutes, player.Player.Name)
				}
			}
		}
		for _, inj := range injuries {
			var t *lineup.Team
			if inj.Team.ID == f.Teams.Home.ID {
				t = &home
			} else if inj.Team.ID == f.Teams.Away.ID {
				t = &away
			} else {
				continue
			}
			if inj.Player.Name != "" {
				t.Injuries = append(t.Injuries, inj.Player.Name+" · "+inj.Player.Reason)
			}
		}
		source := m.ID + "/lineup/" + now.UTC().Format(time.RFC3339Nano)
		st.Lineups = append(st.Lineups, lineup.Snapshot{ID: ID(source), MatchID: m.ID, Home: home, Away: away, CapturedAt: now.UTC(), Phase: "manual", SourceKey: source})
	}
	return st, nil
}
func emptyTeam() lineup.Team {
	return lineup.Team{Starting: []string{}, Substitutes: []string{}, Injuries: []string{}, Suspensions: []string{}}
}
func fixtureStatus(code string) string {
	switch code {
	case "NS":
		return "scheduled"
	case "FT", "AET", "PEN":
		return "finished"
	case "CANC":
		return "cancelled"
	case "PST":
		return "postponed"
	case "ABD", "SUSP":
		return "suspended"
	default:
		return "live"
	}
}

// Exact full-regulation market names only; never infer European or half-time handicaps.
func apiSelection(market, value string) (odds.Selection, bool) {
	var sel odds.Selection
	switch market {
	case "Match Winner":
		sel.Market = "1X2"
		switch value {
		case "Home":
			sel.Side = "home"
		case "Draw":
			sel.Side = "draw"
		case "Away":
			sel.Side = "away"
		default:
			return sel, false
		}
	case "Goals Over/Under", "Asian Handicap":
		fields := strings.Fields(value)
		if len(fields) != 2 {
			return sel, false
		}
		line, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return sel, false
		}
		sel.Line = &line
		if market == "Goals Over/Under" {
			sel.Market = "OU"
			switch fields[0] {
			case "Over":
				sel.Side = "over"
			case "Under":
				sel.Side = "under"
			default:
				return sel, false
			}
		} else {
			sel.Market = "AH"
			switch fields[0] {
			case "Home":
				sel.Side = "home"
			case "Away":
				sel.Side = "away"
			default:
				return sel, false
			}
		}
	default:
		return sel, false
	}
	return sel, sel.Validate() == nil
}
