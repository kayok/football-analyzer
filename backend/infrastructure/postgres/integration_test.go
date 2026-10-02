package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"football/infrastructure/ai"
	"football/infrastructure/bootstrap"
	"football/infrastructure/footballapi"
	"football/infrastructure/postgres"
	"football/internal/application/ports"
	"football/internal/application/usecase"
	delivery "football/internal/delivery/http"
	"football/internal/prediction/engine"
	rec "football/internal/recommendation/domain"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }
func TestPostgresVerticalSlice(t *testing.T) {
	rawURL := os.Getenv("TEST_DATABASE_URL")
	if rawURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the isolated PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	schema := "test_" + time.Now().Format("20060102150405") + "_" + bootstrap.IDs{}.New()[:8]
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = conn.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	// Remove only the uniquely created test schema; never touch application data or volumes.
	defer func() {
		if _, err := conn.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	store, err := postgres.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Migrate(ctx, filepath.Join("..", "..", "migrations"), "up"); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx, filepath.Join("..", "..", "migrations"), "up"); err != nil {
		t.Fatal("migration must be idempotent", err)
	}
	clk := &clock{time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)}
	eng := engine.Poisson{}
	svc := usecase.New(store, clk, bootstrap.IDs{}, eng, footballapi.MockFootballProvider{Engine: eng}, ai.MockAISummaryProvider{}, rec.Rules{PlayEV: .05, MaxAge: 15 * time.Minute}, "Asia/Bangkok")
	if err = svc.Run(ctx, true); err != nil {
		t.Fatal(err)
	}
	today, err := svc.Today(ctx, "Asia/Bangkok")
	if err != nil || len(today) != 3 {
		t.Fatal(today, err)
	}
	statuses := map[string]bool{}
	var play, watch usecase.Card
	for _, c := range today {
		statuses[c.Recommendation.Status] = true
		if c.Recommendation.Status == "PLAY" {
			play = c
		}
		if c.Recommendation.Status == "WATCH" {
			watch = c
		}
	}
	for _, status := range []string{"PLAY", "WATCH", "PASS"} {
		if !statuses[status] {
			t.Fatal("missing", status)
		}
	}
	h, err := svc.History(ctx, "", "Asia/Bangkok")
	if err != nil || len(h.Picks) != 2 || h.Stats.ROI == nil {
		t.Fatal("historical demo picks missing", h, err)
	}
	handler := delivery.New(svc, "http://localhost:3000", slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tc := range []struct {
		method, path, body, origin string
		status                     int
	}{
		{"GET", "/health", "", "", 200},
		{"GET", "/api/v1/matches/today?timezone=Bad/Zone", "", "", 400},
		{"GET", "/api/v1/user-picks?limit=201", "", "", 400},
		{"GET", "/api/v1/matches/not-found", "", "", 404},
		{"POST", "/api/v1/user-picks", `{"recommendation_id":"x","odds":9}`, "", 400},
		{"POST", "/api/v1/user-picks", `{"recommendation_id":"x"} garbage`, "", 400},
		{"GET", "/api/v1/matches/today", "", "http://unauthorized.test", 403},
		{"GET", "/api/v1/matches/today?limit=1", "", "http://localhost:3000", 200},
	} {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		if tc.origin != "" {
			request.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		if w.Code != tc.status {
			t.Fatalf("%s %s: got %d want %d (%s)", tc.method, tc.path, w.Code, tc.status, w.Body.String())
		}
		if tc.status >= 400 {
			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["error"] == nil {
				t.Fatal("unstructured error", w.Body.String())
			}
		}
	}
	// Competing requests must create precisely one active pick.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := svc.Pick(ctx, play.Recommendation.ID); errs <- e }()
	}
	wg.Wait()
	close(errs)
	success := 0
	for e := range errs {
		if e == nil {
			success++
		} else {
			var app *usecase.Error
			if !errors.As(e, &app) || app.Code != "DUPLICATE_PICK" {
				t.Fatal(e)
			}
		}
	}
	if success != 1 {
		t.Fatal("duplicate transaction", success)
	}
	picks, err := svc.Picks(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	var selected usecase.PickView
	for _, p := range picks {
		if p.MatchID == play.ID {
			selected = p
		}
	}
	snapshot, _ := json.Marshal(selected.Pick)
	before, err := svc.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.Run(ctx, true); err != nil {
		t.Fatal(err)
	}
	after, err := svc.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Odds) != len(after.Odds) || len(after.Recommendations) <= len(before.Recommendations) {
		t.Fatal("repeat seed duplicated source or failed to append history")
	}
	again, err := svc.PickByID(ctx, selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	same, _ := json.Marshal(again.Pick)
	if string(snapshot) != string(same) {
		t.Fatal("pick snapshot mutated")
	}
	if _, err = svc.Pick(ctx, watch.Recommendation.ID); err == nil {
		t.Fatal("superseded recommendation accepted")
	}
	today, _ = svc.Today(ctx, "Asia/Bangkok")
	for _, c := range today {
		if c.ID == watch.ID {
			watch = c
		}
	}
	wp, err := svc.Pick(ctx, watch.Recommendation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.Cancel(ctx, wp.ID); err != nil {
		t.Fatal(err)
	}
	if err = svc.Cancel(ctx, wp.ID); err != nil {
		t.Fatal(err)
	}
	cancelled, err := svc.PickByID(ctx, wp.ID)
	if err != nil || cancelled.CancelledAt == nil {
		t.Fatal(cancelled, err)
	}
	if err = svc.FinishDemo(ctx, play.ID, 2, 0); err != nil {
		t.Fatal(err)
	}
	settled, err := svc.PickByID(ctx, selected.ID)
	if err != nil || settled.Result == nil || *settled.Result != "win" || settled.SettledAt == nil {
		t.Fatal(settled, err)
	}
	if err = svc.Cancel(ctx, selected.ID); err == nil {
		t.Fatal("settled pick cancelled")
	}
	// Failed updates roll back all writes.
	if err = store.Update(ctx, func(st *ports.State) error { st.Matches = nil; return errors.New("rollback") }); err == nil {
		t.Fatal("rollback error lost")
	}
	state, err := svc.State(ctx)
	if err != nil || len(state.Matches) != 5 {
		t.Fatal("rollback failed", err)
	}
	clk.now = clk.now.Add(16 * time.Minute)
	if _, err = svc.Pick(ctx, watch.Recommendation.ID); err == nil {
		t.Fatal("stale odds accepted")
	}
	if _, err = svc.Today(ctx, "Bad/Timezone"); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	h, err = svc.History(ctx, "2026-10-02", "Asia/Bangkok")
	if err != nil || len(h.Picks) != 2 || h.Stats.Settled != 1 {
		t.Fatal("history eligibility", h, err)
	}
}
