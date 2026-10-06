package footballapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUsageReportedOnProviderFailureAndQuotaStop(t *testing.T) {
	for _, remaining := range []string{"0", "20"} {
		calls := 0
		reports := 0
		var usage Usage
		p := APIFootball{Key: "secret-key", Leagues: []int{39}, ReportUsage: func(u Usage) { reports++; usage = u }, Client: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			calls++
			res := response(`{"errors":[],"response":[]}`)
			res.Header.Set("x-ratelimit-requests-limit", "100")
			res.Header.Set("x-ratelimit-requests-remaining", remaining)
			return res, nil
		})}}
		_, err := p.Fetch(context.Background(), time.Now(), "UTC")
		if err != nil || reports != 1 || usage.RequestsAttempted != 1 || usage.Responses != 1 || usage.ByEndpoint["/fixtures"] != 1 || usage.DailyUsed() == nil {
			t.Fatal(usage, err)
		}
		r := apiRun{provider: p, usage: usage, calls: calls}
		if remaining == "0" {
			if _, err = r.get(context.Background(), "/fixtures", nil, &[]apiFixture{}); err == nil || calls != 1 {
				t.Fatal("daily quota did not stop request", err)
			}
		}
	}
	reports := 0
	p := APIFootball{Key: "secret-key", Leagues: []int{39}, ReportUsage: func(u Usage) {
		reports++
		if u.Responses != 1 || u.RequestsAttempted != 1 || u.Level() != "unknown" {
			t.Fatal(u)
		}
	}, Client: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return response(`{"errors":{"token":"secret-key"},"response":[]}`), nil
	})}}
	if _, err := p.Fetch(context.Background(), time.Now(), "UTC"); err == nil || strings.Contains(err.Error(), p.Key) || reports != 1 {
		t.Fatal("failed request not reported safely", err)
	}
}
func TestUnknownQuotaAndUsageLevels(t *testing.T) {
	for _, raw := range []string{"", "-1", "secret-key"} {
		h := make(http.Header)
		h.Set("x-ratelimit-requests-limit", raw)
		if quotaHeader(h, "x-ratelimit-requests-limit") != nil {
			t.Fatal("invalid header accepted")
		}
	}
	if (Usage{}).Level() != "unknown" {
		t.Fatal("missing headers became zero")
	}
	limit := 100
	for _, tc := range []struct {
		remaining int
		level     string
	}{{95, "low"}, {50, "moderate"}, {20, "high"}, {0, "exhausted"}, {101, "unknown"}} {
		if (Usage{DailyLimit: &limit, DailyRemaining: &tc.remaining}).Level() != tc.level {
			t.Fatal(tc)
		}
	}
	minute := 0
	calls := 0
	r := apiRun{provider: APIFootball{MaxRequests: 50, Client: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { calls++; return response("{}"), nil })}}, usage: Usage{MinuteRemaining: &minute, ByEndpoint: map[string]int{}}}
	if _, err := r.get(context.Background(), "/fixtures", nil, &[]apiFixture{}); err == nil || calls != 0 {
		t.Fatal("minute quota not enforced")
	}
}
