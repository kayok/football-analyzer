package footballapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixtureJSON(id int, status, date string) string {
	return fmt.Sprintf(`{"fixture":{"id":%d,"date":%q,"status":{"short":%q}},"league":{"id":39,"season":2026,"name":"Premier League"},"teams":{"home":{"id":1,"name":"Home"},"away":{"id":2,"name":"Away"}},"score":{"fulltime":{"home":2,"away":1}}}`, id, date, status)
}
func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestAPIFootballRealMapping(t *testing.T) {
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	p := APIFootball{Key: "test-key", Leagues: []int{39}, Client: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("x-apisports-key") != "test-key" {
			t.Fatal("missing auth header")
		}
		q := r.URL.Query()
		switch r.URL.Path {
		case "/fixtures":
			if q.Get("date") != "" {
				return response(`{"errors":[],"paging":{"total":1},"response":[` + fixtureJSON(10, "NS", "2026-10-06T11:00:00Z") + `,` + fixtureJSON(11, "AET", "2026-10-06T07:00:00Z") + `]}`), nil
			}
			if q.Get("to") != "2026-10-05" || q.Get("status") != "FT" {
				t.Fatal("unsafe history range", q)
			}
			return response(`{"errors":[],"response":[` + fixtureJSON(1, "FT", "2026-10-01T11:00:00Z") + `]}`), nil
		case "/odds":
			if q.Get("page") == "2" {
				return response(`{"errors":[],"paging":{"total":2},"response":[]}`), nil
			}
			return response(`{"errors":[],"paging":{"total":2},"response":[{"fixture":{"id":10},"update":"2026-10-06T08:00:00Z","bookmakers":[{"id":1,"name":"RealBook","bets":[{"id":1,"name":"Match Winner","values":[{"value":"Home","odd":"2.1"},{"value":"Draw","odd":"NaN"}]},{"id":5,"name":"Goals Over/Under","values":[{"value":"Over 2.25","odd":"1.9"}]},{"id":4,"name":"Asian Handicap","values":[{"value":"Away +0.25","odd":"1.8"}]},{"id":99,"name":"Goals Over/Under First Half","values":[{"value":"Over 1.5","odd":"2.0"}]}]}]}]}`), nil
		case "/fixtures/lineups":
			return response(`{"errors":[],"response":[{"team":{"id":1},"startXI":[{"player":{"name":"Player A"}}],"substitutes":[]}]}`), nil
		case "/injuries":
			return response(`{"errors":[],"response":[{"team":{"id":2},"player":{"name":"Player B","reason":"Injury"}}]}`), nil
		default:
			t.Fatal("unexpected path", r.URL.Path)
			return nil, nil
		}
	})}}
	st, err := p.Fetch(context.Background(), now, "Asia/Bangkok")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Matches) != 3 || len(st.Odds) != 3 || len(st.Lineups) != 1 || len(st.Results) != 2 {
		t.Fatal("mapping", st)
	}
	for _, m := range st.Matches {
		if m.Provider != APIName || m.ExpectedHome != 0 {
			t.Fatal("mock data inserted", m)
		}
	}
	if st.Odds[0].CapturedAt.Equal(now) || st.Odds[0].CapturedAt.Hour() != 8 {
		t.Fatal("old source time refreshed")
	}
	if st.Results[1].Home != 2 || st.Results[1].Away != 1 {
		t.Fatal("regulation score lost")
	}
	if st.Lineups[0].Home.Starting[0] != "Player A" || len(st.Lineups[0].Away.Injuries) != 1 {
		t.Fatal("lineup lost")
	}
	again, err := p.Fetch(context.Background(), now, "Asia/Bangkok")
	if err != nil || again.Odds[0].ID != st.Odds[0].ID || again.Matches[0].ID != st.Matches[0].ID {
		t.Fatal("unstable identifiers", err)
	}
}
func TestAPIFootballErrorsAndBudget(t *testing.T) {
	now := time.Now()
	for _, body := range []string{`{"errors":{"token":"secret-test-key"},"response":[]}`, `bad-json`, `{"errors":[],"response":{}}`} {
		p := APIFootball{Key: "secret-test-key", Leagues: []int{39}, Client: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return response(body), nil })}}
		if _, err := p.Fetch(context.Background(), now, "UTC"); err == nil || strings.Contains(err.Error(), p.Key) {
			t.Fatal("missing or unsafe error", err)
		}
	}
	p := APIFootball{Key: "key", Leagues: []int{39}, MaxRequests: 1, Client: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return response(`{"errors":[],"response":[` + fixtureJSON(10, "NS", now.Add(time.Hour).UTC().Format(time.RFC3339)) + `]}`), nil
	})}}
	if _, err := p.Fetch(context.Background(), now, "UTC"); err == nil {
		t.Fatal("budget not enforced")
	}
	if _, err := (APIFootball{}).Fetch(context.Background(), now, "UTC"); err == nil {
		t.Fatal("missing key accepted")
	}
}
func TestExactPreMatchMarketMapping(t *testing.T) {
	for _, tc := range []struct {
		market, value string
		ok            bool
	}{
		{"Match Winner", "Home", true}, {"Asian Handicap", "Home -0.25", true}, {"Goals Over/Under", "Under 2", true},
		{"Asian Handicap", "Home -0.3", false}, {"European Handicap", "Home -1", false}, {"Match Winner First Half", "Home", false}, {"Goals Over/Under", "Over NaN", false}, {"Goals Over/Under", "Over 2,2.5", false},
	} {
		if _, ok := apiSelection(tc.market, tc.value); ok != tc.ok {
			t.Fatal(tc)
		}
	}
}
