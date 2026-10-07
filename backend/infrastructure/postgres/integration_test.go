package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"football/infrastructure/ai"
	"football/infrastructure/bootstrap"
	"football/infrastructure/footballapi"
	"football/infrastructure/postgres"
	"football/infrastructure/security"
	"football/internal/application/ports"
	"football/internal/application/usecase"
	delivery "football/internal/delivery/http"
	"football/internal/prediction/engine"
	rec "football/internal/recommendation/domain"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http"
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
	members, err := usecase.NewMembership(store, security.Passwords{}, security.Tokens{}, clk, bootstrap.IDs{})
	if err != nil {
		t.Fatal(err)
	}
	owner, token, err := members.Register(ctx, "Owner", "OWNER@example.test", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	other, otherToken, err := members.Register(ctx, "Other", "other@example.test", "test-password-456")
	if err != nil {
		t.Fatal(err)
	}
	if owner.Email != "owner@example.test" {
		t.Fatal("email normalization")
	}
	if _, _, err = members.Register(ctx, "Duplicate", "Owner@example.test", "test-password-123"); err == nil {
		t.Fatal("duplicate email accepted")
	}
	if _, _, err = members.Login(ctx, owner.Email, "wrong-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if _, _, err = members.Login(ctx, "missing@example.test", "test-password-123"); err == nil {
		t.Fatal("unknown member accepted")
	}
	loggedIn, loginToken, err := members.Login(ctx, " OWNER@example.test ", "test-password-123")
	if err != nil || loggedIn.ID != owner.ID {
		t.Fatal("login failed", err)
	}
	if err = members.Logout(ctx, loginToken); err != nil {
		t.Fatal(err)
	}
	if _, err = members.Current(ctx, loginToken); err == nil {
		t.Fatal("logged out session accepted")
	}
	personal, err := svc.ForUser(owner.ID).History(ctx, "", "Asia/Bangkok")
	if err != nil || len(personal.Picks) != 4 {
		t.Fatal("legacy picks not claimed", len(personal.Picks), err)
	}
	empty, err := svc.ForUser(other.ID).History(ctx, "", "Asia/Bangkok")
	if err != nil || len(empty.Picks) != 0 || empty.Stats.ROI != nil {
		t.Fatal("other member history leaked", empty, err)
	}
	if _, err = svc.ForUser(other.ID).PickByID(ctx, selected.ID); err == nil {
		t.Fatal("foreign pick visible")
	}
	if err = svc.ForUser(other.ID).Cancel(ctx, selected.ID); err == nil {
		t.Fatal("foreign cancellation accepted")
	}
	clk.now = time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	currentCards, err := svc.ForUser(owner.ID).Today(ctx, "Asia/Bangkok")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range currentCards {
		if c.ID == watch.ID {
			watch = c
		}
	}
	ownedPick, err := svc.ForUser(owner.ID).Pick(ctx, watch.Recommendation.ID)
	if err != nil {
		t.Fatal(err)
	}
	otherPick, err := svc.ForUser(other.ID).Pick(ctx, watch.Recommendation.ID)
	if err != nil || ownedPick.ID == otherPick.ID {
		t.Fatal("per-member uniqueness failed", err)
	}
	if err = svc.ForUser(other.ID).Cancel(ctx, ownedPick.ID); err == nil {
		t.Fatal("foreign cancellation accepted")
	}
	if err = svc.ForUser(other.ID).Cancel(ctx, otherPick.ID); err != nil {
		t.Fatal(err)
	}
	stillActive, err := svc.ForUser(owner.ID).PickByID(ctx, ownedPick.ID)
	if err != nil || stillActive.CancelledAt != nil {
		t.Fatal("other cancellation affected owner", err)
	}
	handler := delivery.New(svc, members, "http://localhost:3000", slog.New(slog.NewTextHandler(io.Discard, nil)), clk)
	privateMembers, err := usecase.NewPrivateMembership(store, security.Passwords{}, security.Tokens{}, clk, bootstrap.IDs{}, owner.Email)
	if err != nil {
		t.Fatal(err)
	}
	lock, acquired, err := store.TrySyncLock(ctx)
	if err != nil || !acquired {
		t.Fatal("sync lock", err)
	}
	_, acquired, err = store.TrySyncLock(ctx)
	if err != nil || acquired {
		t.Fatal("overlapping sync accepted", err)
	}
	if err = lock.Save(ctx, ports.SyncStatus{State: "succeeded", LastSuccessAt: &clk.now}); err != nil {
		t.Fatal(err)
	}
	lock.Release()
	persisted, err := store.SyncStatus(ctx)
	if err != nil || persisted.LastSuccessAt == nil {
		t.Fatal("sync status not persisted", err)
	}
	syncer := usecase.NewSyncManager(ctx, store, svc, clk, 30*time.Minute, 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	privateHandler := delivery.New(svc, privateMembers, "http://localhost:3000", slog.New(slog.NewTextHandler(io.Discard, nil)), clk, syncer)
	for _, tc := range []struct {
		method, token, origin string
		status                int
	}{
		{"GET", "", "", 401}, {"GET", otherToken, "", 401}, {"GET", token, "", 200},
		{"POST", token, "", 403}, {"POST", otherToken, "http://localhost:3000", 401},
	} {
		r := httptest.NewRequest(tc.method, "/api/v1/sync", nil)
		r.Header.Set("Origin", tc.origin)
		if tc.token != "" {
			r.AddCookie(&http.Cookie{Name: "football_session", Value: tc.token})
		}
		w := httptest.NewRecorder()
		privateHandler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("sync authorization: got %d want %d", w.Code, tc.status)
		}
	}
	for _, tc := range []struct {
		method, path, body, token string
		status                    int
	}{
		{"POST", "/api/v1/auth/register", `{"name":"New","email":"new@example.test","password":"new-password"}`, "", 403},
		{"POST", "/api/v1/auth/login", `{"email":"other@example.test","password":"test-password-456"}`, "", 401},
		{"POST", "/api/v1/auth/login", `{"email":"owner@example.test","password":"test-password-123"}`, "", 200},
		{"GET", "/api/v1/auth/me", "", token, 200},
		{"GET", "/api/v1/user-picks", "", otherToken, 401},
	} {
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.Header.Set("Origin", "http://localhost:3000")
		r.Header.Set("Content-Type", "application/json")
		if tc.token != "" {
			r.AddCookie(&http.Cookie{Name: "football_session", Value: tc.token})
		}
		w := httptest.NewRecorder()
		privateHandler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("private %s: got %d want %d: %s", tc.path, w.Code, tc.status, w.Body.String())
		}
	}
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
			request.Header.Set("Origin", "http://localhost:3000")
		}
		if tc.origin != "" {
			request.Header.Set("Origin", tc.origin)
		}
		request.AddCookie(&http.Cookie{Name: "football_session", Value: token})
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

	// Authentication and CSRF fail closed, independently of frontend visibility.
	for _, tc := range []struct {
		method, path, token, origin string
		want                        int
	}{
		{"GET", "/api/v1/user-picks", "", "", 401},
		{"GET", "/api/v1/auth/me", "invalid", "", 401},
		{"DELETE", "/api/v1/user-picks/" + ownedPick.ID, otherToken, "http://localhost:3000", 404},
		{"POST", "/api/v1/auth/logout", token, "", 403},
		{"GET", "/api/v1/auth/me", token, "", 200},
		{"POST", "/api/v1/auth/logout", token, "http://localhost:3000", 204},
		{"GET", "/api/v1/auth/me", token, "", 401},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.token != "" {
			r.AddCookie(&http.Cookie{Name: "football_session", Value: tc.token})
		}
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("auth %s %s: %d want %d (%s)", tc.method, tc.path, w.Code, tc.want, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "PasswordHash") || strings.Contains(w.Body.String(), "password_hash") {
			t.Fatal("password hash exposed")
		}
	}
	// Login HTTP cookies are opaque and inaccessible to JavaScript.
	request := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"owner@example.test","password":"test-password-123"}`))
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal("HTTP login", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge != int(usecase.SessionLifetime.Seconds()) {
		t.Fatal("unsafe session cookie", cookies)
	}
	if strings.Contains(response.Body.String(), "test-password") || strings.Contains(response.Body.String(), "password_hash") {
		t.Fatal("secret exposed")
	}
	limited := delivery.New(svc, members, "http://localhost:3000", slog.New(slog.NewTextHandler(io.Discard, nil)), clk)
	for i := 0; i < 11; i++ {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("invalid JSON"))
		r.Header.Set("Origin", "http://localhost:3000")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		limited.ServeHTTP(w, r)
		want := 400
		if i == 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("rate limit attempt %d: %d want %d", i, w.Code, want)
		}
	}
	clk.now = clk.now.Add(time.Minute)
	retry := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("invalid JSON"))
	retry.Header.Set("Origin", "http://localhost:3000")
	retry.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	limited.ServeHTTP(response, retry)
	if response.Code != 400 {
		t.Fatal("rate limit did not expire", response.Code)
	}

	clk.now = clk.now.Add(usecase.SessionLifetime + time.Hour)
	if _, err = members.Current(ctx, otherToken); err == nil {
		t.Fatal("expired session accepted")
	}

}
