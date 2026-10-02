package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"football/internal/application/ports"
	pick "football/internal/userpick/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"strings"
	"time"
)

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool}, nil
}
func (s *Store) Close()                           { s.pool.Close() }
func (s *Store) Health(ctx context.Context) error { return s.pool.Ping(ctx) }
func read[T any](ctx context.Context, tx pgx.Tx, query string, args ...interface{}) ([]T, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func load(ctx context.Context, tx pgx.Tx, bounds ...interface{}) (ports.State, error) {
	query := func(sql string, isMatch bool) string {
		if len(bounds) == 0 {
			return sql
		}
		condition := "match_id IN (SELECT id FROM matches WHERE kickoff >= $1 AND kickoff < $2)"
		if isMatch {
			condition = "kickoff >= $1 AND kickoff < $2"
		}
		return strings.Replace(sql, " ORDER BY", " WHERE "+condition+" ORDER BY", 1)
	}
	var st ports.State
	var err error
	st.Matches, err = read[matchType](ctx, tx, query("SELECT data FROM matches ORDER BY kickoff,id", true), bounds...)
	if err != nil {
		return st, err
	}
	st.Odds, err = read[oddsType](ctx, tx, query("SELECT data FROM odds_snapshots ORDER BY captured_at,id", false), bounds...)
	if err != nil {
		return st, err
	}
	st.Lineups, err = read[lineupType](ctx, tx, query("SELECT data FROM lineup_snapshots ORDER BY captured_at,id", false), bounds...)
	if err != nil {
		return st, err
	}
	st.Predictions, err = read[predType](ctx, tx, query("SELECT data FROM predictions ORDER BY generated_at,id", false), bounds...)
	if err != nil {
		return st, err
	}
	st.Recommendations, err = read[recType](ctx, tx, query("SELECT data FROM recommendations ORDER BY generated_at,id", false), bounds...)
	if err != nil {
		return st, err
	}
	st.Picks, err = read[pick.Pick](ctx, tx, query(`SELECT data || jsonb_build_object('cancelled_at',cancelled_at,'settled_at',settled_at,'result',result,'net_profit_units',net_profit_units) FROM user_picks ORDER BY picked_at,id`, false), bounds...)
	if err != nil {
		return st, err
	}
	st.Results, err = read[resultType](ctx, tx, query("SELECT data FROM match_results ORDER BY recorded_at,match_id", false), bounds...)
	return st, err
}
func (s *Store) View(ctx context.Context) (ports.State, error) { return s.view(ctx) }
func (s *Store) ViewDate(ctx context.Context, start, end time.Time) (ports.State, error) {
	return s.view(ctx, start, end)
}
func (s *Store) view(ctx context.Context, bounds ...interface{}) (ports.State, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ports.State{}, err
	}
	defer tx.Rollback(ctx)
	st, err := load(ctx, tx, bounds...)
	if err != nil {
		return st, err
	}
	return st, tx.Commit(ctx)
}
func (s *Store) Update(ctx context.Context, fn func(*ports.State) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// One local application: serialize worker and manual selection consistently across processes.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(7102401)"); err != nil {
		return err
	}
	st, err := load(ctx, tx)
	if err != nil {
		return err
	}
	var before ports.State
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &before); err != nil {
		return err
	}
	if err = fn(&st); err != nil {
		return err
	}
	if err = save(ctx, tx, before, st); err != nil {
		return fmt.Errorf("persist state: %w", err)
	}
	return tx.Commit(ctx)
}
func encoded(v interface{}) ([]byte, error) { return json.Marshal(v) }
func save(ctx context.Context, tx pgx.Tx, before, st ports.State) error {
	for _, m := range st.Matches {
		changed := true
		for _, old := range before.Matches {
			if old.ID == m.ID && reflect.DeepEqual(old, m) {
				changed = false
			}
		}
		if !changed {
			continue
		}
		if _, err := tx.Exec(ctx, "INSERT INTO competitions(id,name) VALUES($1,$2) ON CONFLICT DO NOTHING", m.CompetitionID, m.Competition); err != nil {
			return err
		}
		for _, team := range [][2]string{{m.HomeID, m.Home}, {m.AwayID, m.Away}} {
			if _, err := tx.Exec(ctx, "INSERT INTO teams(id,name) VALUES($1,$2) ON CONFLICT DO NOTHING", team[0], team[1]); err != nil {
				return err
			}
		}
		raw, err := encoded(m)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO matches(id,competition_id,home_id,away_id,kickoff,provider_name,external_id,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET kickoff=excluded.kickoff,data=excluded.data`, m.ID, m.CompetitionID, m.HomeID, m.AwayID, m.Kickoff, m.Provider, m.ExternalID, raw); err != nil {
			return err
		}
	}
	existing := map[string]bool{}
	for _, o := range before.Odds {
		existing[o.ID] = true
	}
	for _, o := range st.Odds {
		if existing[o.ID] {
			continue
		}
		raw, err := encoded(o)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO odds_snapshots(id,match_id,captured_at,source_key,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", o.ID, o.MatchID, o.CapturedAt, o.SourceKey, raw); err != nil {
			return err
		}
	}
	existing = map[string]bool{}
	for _, l := range before.Lineups {
		existing[l.ID] = true
	}
	for _, l := range st.Lineups {
		if existing[l.ID] {
			continue
		}
		raw, err := encoded(l)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO lineup_snapshots(id,match_id,captured_at,source_key,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", l.ID, l.MatchID, l.CapturedAt, l.SourceKey, raw); err != nil {
			return err
		}
	}
	existing = map[string]bool{}
	for _, p := range before.Predictions {
		existing[p.ID] = true
	}
	for _, p := range st.Predictions {
		if existing[p.ID] {
			continue
		}
		raw, err := encoded(p)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO predictions(id,match_id,odds_snapshot_id,model_version,generated_at,data) VALUES($1,$2,$3,$4,$5,$6)", p.ID, p.MatchID, p.OddsID, p.ModelVersion, p.GeneratedAt, raw); err != nil {
			return err
		}
	}
	existing = map[string]bool{}
	for _, r := range before.Recommendations {
		existing[r.ID] = true
	}
	for _, r := range st.Recommendations {
		if existing[r.ID] {
			continue
		}
		raw, err := encoded(r)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO recommendations(id,match_id,odds_snapshot_id,prediction_id,model_version,generated_at,data) VALUES($1,$2,$3,$4,$5,$6,$7)", r.ID, r.MatchID, r.OddsID, r.PredictionID, r.ModelVersion, r.GeneratedAt, raw); err != nil {
			return err
		}
	}
	oldPicks := map[string]pick.Pick{}
	for _, p := range before.Picks {
		oldPicks[p.ID] = p
	}
	for _, p := range st.Picks {
		old, ok := oldPicks[p.ID]
		if ok {
			if reflect.DeepEqual(old, p) {
				continue
			}
			if _, err := tx.Exec(ctx, "UPDATE user_picks SET cancelled_at=$2,settled_at=$3,result=$4,net_profit_units=$5 WHERE id=$1", p.ID, p.CancelledAt, p.SettledAt, p.Result, p.Profit); err != nil {
				return err
			}
			continue
		}
		raw, err := encoded(p)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO user_picks(id,recommendation_id,match_id,market,selection,line,picked_at,cancelled_at,settled_at,result,net_profit_units,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, p.ID, p.RecommendationID, p.MatchID, p.Market, p.Side, p.Line, p.PickedAt, p.CancelledAt, p.SettledAt, p.Result, p.Profit, raw); err != nil {
			return err
		}
	}
	existing = map[string]bool{}
	for _, r := range before.Results {
		existing[r.MatchID] = true
	}
	for _, r := range st.Results {
		if existing[r.MatchID] {
			continue
		}
		raw, err := encoded(r)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO match_results(match_id,recorded_at,data) VALUES($1,$2,$3)", r.MatchID, r.RecordedAt, raw); err != nil {
			return err
		}
	}
	return nil
}
