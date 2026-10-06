package footballapi

import (
	"log/slog"
	"net/http"
	"strconv"
)

// Usage contains only counts, never credentials, URLs or upstream response bodies.
// Remaining quotas are the last response's authoritative values, not local estimates.
type Usage struct {
	RequestsAttempted int            `json:"requests_attempted"`
	Responses         int            `json:"responses"`
	DailyLimit        *int           `json:"daily_limit"`
	DailyRemaining    *int           `json:"daily_remaining"`
	MinuteLimit       *int           `json:"minute_limit"`
	MinuteRemaining   *int           `json:"minute_remaining"`
	ByEndpoint        map[string]int `json:"by_endpoint"`
}

func quotaHeader(headers http.Header, name string) *int {
	n, err := strconv.Atoi(headers.Get(name))
	if err != nil || n < 0 {
		return nil
	}
	return &n
}
func (u Usage) DailyUsed() *int {
	if u.DailyLimit == nil || u.DailyRemaining == nil || *u.DailyLimit <= 0 || *u.DailyRemaining > *u.DailyLimit {
		return nil
	}
	n := *u.DailyLimit - *u.DailyRemaining
	return &n
}
func (u Usage) Level() string {
	used := u.DailyUsed()
	if used == nil {
		return "unknown"
	}
	if *u.DailyRemaining == 0 {
		return "exhausted"
	}
	if float64(*used)/float64(*u.DailyLimit) >= 0.8 {
		return "high"
	}
	if float64(*used)/float64(*u.DailyLimit) >= 0.5 {
		return "moderate"
	}
	return "low"
}
func LogUsage(u Usage) {
	slog.Info("API-Football usage", "requests_attempted_this_run", u.RequestsAttempted, "responses_this_run", u.Responses,
		"daily_limit", quotaValue(u.DailyLimit), "daily_used", quotaValue(u.DailyUsed()), "daily_remaining", quotaValue(u.DailyRemaining),
		"minute_limit", quotaValue(u.MinuteLimit), "minute_remaining", quotaValue(u.MinuteRemaining), "usage_level", u.Level(), "by_endpoint", u.ByEndpoint)
}

func quotaValue(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
