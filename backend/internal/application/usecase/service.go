package usecase

import (
	"context"
	"fmt"
	"football/internal/application/ports"
	history "football/internal/history/domain"
	lineup "football/internal/lineup/domain"
	match "football/internal/match/domain"
	odds "football/internal/odds/domain"
	prediction "football/internal/prediction/domain"
	engineports "football/internal/prediction/ports"
	rec "football/internal/recommendation/domain"
	pick "football/internal/userpick/domain"
	"math"
	"sort"
	"strconv"
	"time"
)

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string                { return e.Message }
func fail(code, msg string, status int) error { return &Error{code, msg, status} }

type Service struct {
	store    ports.Store
	clock    ports.Clock
	ids      ports.IDs
	engine   engineports.Engine
	football ports.FootballProvider
	ai       ports.AISummaryProvider
	rules    rec.Rules
	timezone string
}

func New(store ports.Store, clock ports.Clock, ids ports.IDs, engine engineports.Engine, football ports.FootballProvider, ai ports.AISummaryProvider, rules rec.Rules, timezone string) *Service {
	return &Service{store, clock, ids, engine, football, ai, rules, timezone}
}
func (s *Service) Health(ctx context.Context) error               { return s.store.Health(ctx) }
func (s *Service) State(ctx context.Context) (ports.State, error) { return s.store.View(ctx) }

type Card struct {
	match.Match
	Recommendation *rec.Recommendation `json:"recommendation"`
	Pick           *pick.Pick          `json:"pick"`
	Result         *match.Result       `json:"result"`
}
type Detail struct {
	LatestOdds       []odds.Snapshot        `json:"current_odds"`
	LatestLineup     *lineup.Snapshot       `json:"current_lineup"`
	LatestPrediction *prediction.Prediction `json:"current_prediction"`
	Card
	Odds            []odds.Snapshot         `json:"odds_snapshots"`
	Predictions     []prediction.Prediction `json:"prediction_snapshots"`
	Recommendations []rec.Recommendation    `json:"recommendation_snapshots"`
	Lineups         []lineup.Snapshot       `json:"lineup_snapshots"`
}
type PickView struct {
	pick.Pick
	Match match.Match `json:"match"`
}
type History struct {
	Cards           []Card               `json:"matches"`
	Recommendations []rec.Recommendation `json:"recommendations"`
	Picks           []PickView           `json:"picks"`
	Results         []match.Result       `json:"results"`
	Stats           history.Stats        `json:"stats"`
}

func latest(st ports.State, id string) *rec.Recommendation {
	var best *rec.Recommendation
	for _, r := range st.Recommendations {
		if r.MatchID == id && (best == nil || r.Generation > best.Generation) {
			v := r
			best = &v
		}
	}
	return best
}
func card(st ports.State, m match.Match) Card {
	c := Card{Match: m, Recommendation: latest(st, m.ID)}
	for _, p := range st.Picks {
		if p.MatchID == m.ID && p.CancelledAt == nil && c.Recommendation != nil && c.Recommendation.Odds != nil && key(p.Selection) == key(c.Recommendation.Odds.Selection) {
			v := p
			c.Pick = &v
		}
	}
	for _, r := range st.Results {
		if r.MatchID == m.ID {
			v := r
			c.Result = &v
		}
	}
	return c
}
func (s *Service) bounds(date, tz string) (time.Time, time.Time, error) {
	if tz == "" {
		tz = s.timezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, time.Time{}, fail("INVALID_TIMEZONE", "เขตเวลาไม่ถูกต้อง", 400)
	}
	if date == "" {
		date = s.clock.Now().In(loc).Format("2006-01-02")
	}
	a, b, err := history.DateBounds(date, tz)
	if err != nil {
		return a, b, fail("INVALID_DATE", "วันที่ไม่ถูกต้อง", 400)
	}
	return a, b, nil
}
func (s *Service) Today(ctx context.Context, tz string) ([]Card, error) {
	a, b, err := s.bounds("", tz)
	if err != nil {
		return nil, err
	}
	st, err := s.store.ViewDate(ctx, a, b)
	if err != nil {
		return nil, err
	}
	cards := []Card{}
	for _, m := range st.Matches {
		if !m.Kickoff.Before(a) && m.Kickoff.Before(b) {
			cards = append(cards, s.currentCard(card(st, m)))
		}
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].Kickoff.Before(cards[j].Kickoff) })
	return cards, nil
}
func (s *Service) Detail(ctx context.Context, id string) (Detail, error) {
	st, err := s.store.View(ctx)
	if err != nil {
		return Detail{}, err
	}
	var m *match.Match
	for _, v := range st.Matches {
		if v.ID == id {
			t := v
			m = &t
		}
	}
	if m == nil {
		return Detail{}, fail("NOT_FOUND", "ไม่พบคู่แข่งขัน", 404)
	}
	d := Detail{Card: s.currentCard(card(st, *m)), Odds: []odds.Snapshot{}, Predictions: []prediction.Prediction{}, Recommendations: []rec.Recommendation{}}
	for _, o := range st.Odds {
		if o.MatchID == id {
			d.Odds = append(d.Odds, o)
		}
	}
	for _, p := range st.Predictions {
		if p.MatchID == id {
			d.Predictions = append(d.Predictions, p)
		}
	}
	for _, r := range st.Recommendations {
		if r.MatchID == id {
			d.Recommendations = append(d.Recommendations, r)
		}
	}
	lineups := st.Lineups[:0:0]
	for _, l := range st.Lineups {
		if l.MatchID == id {
			lineups = append(lineups, l)
		}
	}
	d.Lineups = lineups
	currentOdds := map[string]odds.Snapshot{}
	for _, o := range d.Odds {
		currentOdds[oddsKey(o)] = o
	}
	d.LatestOdds = []odds.Snapshot{}
	for _, o := range currentOdds {
		d.LatestOdds = append(d.LatestOdds, o)
	}
	sort.Slice(d.LatestOdds, func(i, j int) bool { return oddsKey(d.LatestOdds[i]) < oddsKey(d.LatestOdds[j]) })
	if len(lineups) > 0 {
		l := lineups[len(lineups)-1]
		d.LatestLineup = &l
	}
	if d.Recommendation != nil {
		d.LatestPrediction = d.Recommendation.Prediction
	}
	return d, nil
}
func (s *Service) Pick(ctx context.Context, recommendationID string) (pick.Pick, error) {
	var out pick.Pick
	err := s.store.Update(ctx, func(st *ports.State) error {
		var r *rec.Recommendation
		for _, v := range st.Recommendations {
			if v.ID == recommendationID {
				t := v
				r = &t
			}
		}
		if r == nil {
			return fail("NOT_FOUND", "ไม่พบคำแนะนำ", 404)
		}
		var m match.Match
		for _, v := range st.Matches {
			if v.ID == r.MatchID {
				m = v
			}
		}
		current := latest(*st, m.ID)
		if current == nil || current.ID != r.ID {
			return fail("STALE_RECOMMENDATION", "คำแนะนำเปลี่ยนแล้ว กรุณาโหลดข้อมูลใหม่", 409)
		}
		p, err := pick.Select(s.ids.New(), s.clock.Now(), m, *r)
		if err != nil {
			return fail("NOT_SELECTABLE", "คำแนะนำหมดอายุหรือเลือกไม่ได้ กรุณาโหลดข้อมูลใหม่", 409)
		}
		for _, old := range st.Picks {
			if old.CancelledAt == nil && old.MatchID == p.MatchID && key(old.Selection) == key(p.Selection) {
				return fail("DUPLICATE_PICK", "เลือกคู่นี้และตลาดนี้ไว้แล้ว", 409)
			}
		}
		st.Picks = append(st.Picks, p)
		out = p
		return nil
	})
	return out, err
}
func (s *Service) Cancel(ctx context.Context, id string) error {
	return s.store.Update(ctx, func(st *ports.State) error {
		for i, p := range st.Picks {
			if p.ID == id {
				v, err := p.Cancel(s.clock.Now())
				if err != nil {
					return fail("SETTLED_PICK", "ยกเลิกรายการที่สรุปผลแล้วไม่ได้", 409)
				}
				st.Picks[i] = v
				return nil
			}
		}
		return fail("NOT_FOUND", "ไม่พบรายการ", 404)
	})
}
func (s *Service) Picks(ctx context.Context, includeCancelled bool) ([]PickView, error) {
	st, err := s.store.View(ctx)
	if err != nil {
		return nil, err
	}
	out := []PickView{}
	for _, p := range st.Picks {
		if !includeCancelled && p.CancelledAt != nil {
			continue
		}
		for _, m := range st.Matches {
			if m.ID == p.MatchID {
				out = append(out, PickView{p, m})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PickedAt.After(out[j].PickedAt) })
	return out, nil
}
func (s *Service) PickByID(ctx context.Context, id string) (PickView, error) {
	ps, err := s.Picks(ctx, true)
	if err != nil {
		return PickView{}, err
	}
	for _, p := range ps {
		if p.ID == id {
			return p, nil
		}
	}
	return PickView{}, fail("NOT_FOUND", "ไม่พบรายการ", 404)
}
func (s *Service) History(ctx context.Context, date, tz string) (History, error) {
	var a, b time.Time
	var err error
	if date != "" {
		a, b, err = s.bounds(date, tz)
	} else {
		_, _, err = s.bounds("", tz)
	}
	if err != nil {
		return History{}, err
	}
	var st ports.State
	if date != "" {
		st, err = s.store.ViewDate(ctx, a, b)
	} else {
		st, err = s.store.View(ctx)
	}
	if err != nil {
		return History{}, err
	}
	h := History{Cards: []Card{}, Recommendations: []rec.Recommendation{}, Picks: []PickView{}, Results: []match.Result{}}
	selected := map[string]match.Match{}
	for _, m := range st.Matches {
		if date == "" || (!m.Kickoff.Before(a) && m.Kickoff.Before(b)) {
			selected[m.ID] = m
			h.Cards = append(h.Cards, card(st, m))
		}
	}
	ps := []pick.Pick{}
	preds := []prediction.Prediction{}
	ms := []match.Match{}
	for _, m := range selected {
		ms = append(ms, m)
	}
	for _, p := range st.Picks {
		if m, ok := selected[p.MatchID]; ok {
			ps = append(ps, p)
			h.Picks = append(h.Picks, PickView{p, m})
		}
	}
	for _, p := range st.Predictions {
		if _, ok := selected[p.MatchID]; ok {
			preds = append(preds, p)
		}
	}
	for _, r := range st.Recommendations {
		if _, ok := selected[r.MatchID]; ok {
			h.Recommendations = append(h.Recommendations, r)
		}
	}
	for _, r := range st.Results {
		if _, ok := selected[r.MatchID]; ok {
			h.Results = append(h.Results, r)
		}
	}
	h.Stats = history.Calculate(ps, preds, ms, h.Results)
	sort.Slice(h.Cards, func(i, j int) bool { return h.Cards[i].Kickoff.After(h.Cards[j].Kickoff) })
	return h, nil
}
func key(s odds.Selection) string {
	line := ""
	if s.Line != nil {
		line = strconv.FormatFloat(*s.Line, 'f', 2, 64)
	}
	return s.Market + "/" + s.Side + "/" + line
}
func oddsKey(o odds.Snapshot) string { return key(o.Selection) + "/" + o.Bookmaker }
func (s *Service) Run(ctx context.Context, seed bool) error {
	now := s.clock.Now()
	incoming, err := s.football.Fetch(ctx, now, s.timezone)
	if err != nil {
		return err
	}
	return s.store.Update(ctx, func(st *ports.State) error {
		for _, m := range incoming.Matches {
			found := false
			for i, old := range st.Matches {
				if old.ID == m.ID {
					m.Kickoff = old.Kickoff
					if old.Status != "scheduled" {
						m.Status = old.Status
					}
					st.Matches[i] = m
					found = true
					break
				}
			}
			if !found {
				st.Matches = append(st.Matches, m)
			}
		}
		sources := map[string]bool{}
		for _, o := range st.Odds {
			sources[o.SourceKey] = true
		}
		for _, o := range incoming.Odds {
			if !seed && o.Phase != "manual" {
				continue
			}
			if !sources[o.SourceKey] {
				st.Odds = append(st.Odds, o)
				sources[o.SourceKey] = true
			}
		}
		sources = map[string]bool{}
		for _, l := range st.Lineups {
			sources[l.SourceKey] = true
		}
		for _, l := range incoming.Lineups {
			if !seed && l.Phase != "manual" {
				continue
			}
			if !sources[l.SourceKey] {
				st.Lineups = append(st.Lineups, l)
				sources[l.SourceKey] = true
			}
		}
		for _, r := range incoming.Results {
			found := false
			for _, old := range st.Results {
				if old.MatchID == r.MatchID {
					found = true
				}
			}
			if !found {
				st.Results = append(st.Results, r)
			}
		}
		for _, m := range incoming.Matches {
			for _, stored := range st.Matches {
				if stored.ID == m.ID {
					m = stored
					break
				}
			}
			// Historical demo predictions use honest mock pre-kickoff input times; current runs never backdate.
			generated := now
			if m.Status != "scheduled" {
				if !seed {
					continue
				}
				if latest(*st, m.ID) != nil {
					continue
				}
				generated = m.Kickoff.Add(-time.Minute)
			}
			r := rec.Recommendation{ID: s.ids.New(), Generation: len(st.Recommendations) + 1, MatchID: m.ID, Status: "PASS", ReasonCode: "MISSING_INPUTS", ModelVersion: s.engine.Version(), RuleVersion: "v1-value", Rules: s.rules, GeneratedAt: generated}
			os := map[string]odds.Snapshot{}
			for _, o := range st.Odds {
				if o.MatchID == m.ID {
					old, ok := os[oddsKey(o)]
					if !ok || o.CapturedAt.After(old.CapturedAt) {
						os[oddsKey(o)] = o
					}
				}
			}
			keys := []string{}
			for k := range os {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			bestEligible := false
			bestEV := math.Inf(-1)
			type quoteGroup struct {
				prices [3]float64
				times  [3]time.Time
			}
			quotes := map[string]quoteGroup{}
			for _, o := range os {
				if o.Market != "1X2" {
					continue
				}
				q := quotes[o.Bookmaker]
				i := sideIndex(o.Side)
				q.prices[i] = o.Odds
				q.times[i] = o.CapturedAt
				quotes[o.Bookmaker] = q
			}

			for _, k := range keys {
				o := os[k]
				estimate, err := s.engine.Estimate(ctx, m.ExpectedHome, m.ExpectedAway, o.Selection)
				if err != nil {
					continue
				}
				ev, fair, min, err := estimate.Distribution.Value(o.Odds, s.rules.PlayEV)
				if err != nil {
					continue
				}
				implied, _ := odds.Implied(o.Odds)
				p := prediction.Prediction{ID: s.ids.New(), MatchID: m.ID, OddsID: o.ID, Selection: o.Selection, Distribution: estimate.Distribution, Probability: estimate.Distribution.Win + estimate.Distribution.HalfWin, EV: ev, FairOdds: fair, MinimumOdds: min, Implied: implied, OneXTwo: estimate.OneXTwo, ExpectedHome: m.ExpectedHome, ExpectedAway: m.ExpectedAway, ModelVersion: s.engine.Version(), GeneratedAt: generated}
				if o.Market == "1X2" {
					q := quotes[o.Bookmaker]
					if q.times[0].Equal(q.times[1]) && q.times[1].Equal(q.times[2]) {
						normalized, err := odds.RemoveMargin(q.prices)
						if err == nil {
							v := normalized[sideIndex(o.Side)]
							p.Normalized = &v
						}
					}
				}

				st.Predictions = append(st.Predictions, p)
				status, reason := s.rules.Status(generated, m, o, ev)
				// Demo finished fixtures are evaluated as the pre-match state they represent.
				if seed && m.Status == "finished" {
					pre := m
					pre.Status = "scheduled"
					status, reason = s.rules.Status(generated, pre, o, ev)
				}
				eligible := status != "PASS"
				if r.Prediction == nil || eligible && !bestEligible || eligible == bestEligible && ev > bestEV {
					oid, pid := o.ID, p.ID
					r.OddsID = &oid
					r.PredictionID = &pid
					r.Odds = &o
					r.Prediction = &p
					r.Status = status
					r.ReasonCode = reason
					bestEV = ev
					bestEligible = eligible
				}
			}
			reasons, err := s.ai.Reasons(ctx, r.ReasonCode)
			if err != nil {
				return fmt.Errorf("summary: %w", err)
			}
			r.Reasons = reasons
			st.Recommendations = append(st.Recommendations, r)
			if seed && m.Status == "finished" {
				exists := false
				for _, old := range st.Picks {
					if old.MatchID == m.ID && old.Demo {
						exists = true
					}
				}
				if !exists && r.Status != "PASS" {
					pre := m
					pre.Status = "scheduled"
					p, err := pick.Select(s.ids.New(), generated, pre, r)
					if err == nil {
						p.Demo = true
						st.Picks = append(st.Picks, p)
					}
				}
			}
		}
		for i, p := range st.Picks {
			for _, r := range st.Results {
				if r.MatchID == p.MatchID {
					v, err := p.Settle(now, r)
					if err != nil {
						return err
					}
					st.Picks[i] = v
				}
			}
		}
		return nil
	})
}

// FinishDemo explicitly simulates a final score for a mock fixture; it never calls a real provider.
func (s *Service) FinishDemo(ctx context.Context, id string, home, away int) error {
	if home < 0 || away < 0 {
		return fail("INVALID_SCORE", "คะแนนไม่ถูกต้อง", 400)
	}
	return s.store.Update(ctx, func(st *ports.State) error {
		found := false
		for i, m := range st.Matches {
			if m.ID == id {
				if m.Provider != "mock" {
					return fail("NOT_MOCK", "ใช้ได้เฉพาะข้อมูลจำลอง", 400)
				}
				st.Matches[i].Status = "finished"
				found = true
			}
		}
		if !found {
			return fail("NOT_FOUND", "ไม่พบคู่แข่งขัน", 404)
		}
		for _, r := range st.Results {
			if r.MatchID == id {
				return fail("ALREADY_FINISHED", "มีผลการแข่งขันแล้ว", 409)
			}
		}
		now := s.clock.Now()
		r := match.Result{MatchID: id, Home: home, Away: away, Status: "finished", RecordedAt: now}
		st.Results = append(st.Results, r)
		for i, p := range st.Picks {
			if p.MatchID == id {
				v, err := p.Settle(now, r)
				if err != nil {
					return err
				}
				st.Picks[i] = v
			}
		}
		return nil
	})
}

func sideIndex(side string) int {
	if side == "draw" {
		return 1
	}
	if side == "away" {
		return 2
	}
	return 0
}

func (s *Service) currentCard(c Card) Card {
	if c.Recommendation == nil || c.Recommendation.Odds == nil || c.Recommendation.Prediction == nil {
		return c
	}
	r := *c.Recommendation
	status, reason := r.Rules.Status(s.clock.Now(), c.Match, *r.Odds, r.Prediction.EV)
	if status == "PASS" && r.Status != "PASS" {
		r.Status = status
		r.ReasonCode = reason
		if reason == "STALE_ODDS" {
			r.Reasons = []string{"ราคาหมดอายุ กรุณารัน worker เพื่ออัปเดตข้อมูลจำลอง"}
		} else {
			r.Reasons = []string{"การแข่งขันเริ่มแล้วหรือจบแล้ว"}
		}
		c.Recommendation = &r
	}
	return c
}
