package http

import (
	"context"
	"encoding/json"
	"errors"
	"football/internal/application/ports"
	"football/internal/application/usecase"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	clock    ports.Clock
	svc      *usecase.Service
	origin   string
	log      *slog.Logger
	members  *usecase.Membership
	mu       sync.Mutex
	attempts map[string]attempt
	syncer   *usecase.SyncManager
}

func New(svc *usecase.Service, members *usecase.Membership, origin string, log *slog.Logger, clock ports.Clock, syncers ...*usecase.SyncManager) http.Handler {
	h := &Handler{svc: svc, origin: origin, log: log, members: members, attempts: make(map[string]attempt), clock: clock}
	if len(syncers) > 0 {
		h.syncer = syncers[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	mux.HandleFunc("GET /api/v1/matches/today", h.today)
	mux.HandleFunc("GET /api/v1/matches/{id}", h.detail)
	mux.HandleFunc("GET /api/v1/matches/{id}/{kind}", h.part)
	mux.HandleFunc("GET /api/v1/user-picks", h.picks)
	mux.HandleFunc("POST /api/v1/user-picks", h.pick)
	mux.HandleFunc("GET /api/v1/user-picks/{id}", h.pickDetail)
	mux.HandleFunc("DELETE /api/v1/user-picks/{id}", h.cancel)
	mux.HandleFunc("GET /api/v1/history", h.history)
	mux.HandleFunc("GET /api/v1/history/{date}", h.history)
	mux.HandleFunc("GET /api/v1/sync", h.syncStatus)
	mux.HandleFunc("POST /api/v1/sync", h.startSync)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		defer func() {
			log.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != h.origin {
				h.err(w, &usecase.Error{Code: "ORIGIN_DENIED", Message: "ไม่ได้อนุญาต origin นี้", Status: 403})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", h.origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodDelete) && r.Header.Get("Origin") != h.origin {
			h.err(w, &usecase.Error{Code: "ORIGIN_DENIED", Message: "กรุณาทำรายการผ่านหน้าเว็บ", Status: 403})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/v1/auth/register" && r.URL.Path != "/api/v1/auth/login" {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil {
				h.err(w, &usecase.Error{Code: "UNAUTHENTICATED", Message: "กรุณาเข้าสู่ระบบ", Status: 401})
				return
			}
			user, err := h.members.Current(r.Context(), cookie.Value)
			if err != nil {
				h.err(w, err)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), userKey{}, user.ID))
		}
		mux.ServeHTTP(w, r)
	})
}

type userKey struct{}

func (h *Handler) service(r *http.Request) *usecase.Service {
	id, _ := r.Context().Value(userKey{}).(string)
	return h.svc.ForUser(id)
}
func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("encode response", "error", err)
	}
}
func (h *Handler) err(w http.ResponseWriter, err error) {
	var e *usecase.Error
	if errors.As(err, &e) {
		h.json(w, e.Status, map[string]interface{}{"error": map[string]string{"code": e.Code, "message": e.Message}})
		return
	}
	h.log.Error("request failed", "error", err)
	h.json(w, 500, map[string]interface{}{"error": map[string]string{"code": "INTERNAL", "message": "เกิดข้อผิดพลาด กรุณาลองใหม่"}})
}
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Health(r.Context()); err != nil {
		h.json(w, 503, map[string]string{"status": "unhealthy"})
		return
	}
	h.json(w, 200, map[string]string{"status": "ok"})
}
func pagination(r *http.Request) (int, int, error) {
	offset, limit := 0, 50
	var err error
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, &usecase.Error{Code: "INVALID_PAGINATION", Message: "offset ไม่ถูกต้อง", Status: 400}
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 200 {
			return 0, 0, &usecase.Error{Code: "INVALID_PAGINATION", Message: "limit ต้องอยู่ระหว่าง 1–200", Status: 400}
		}
	}
	return offset, limit, nil
}
func page[T any](v []T, offset, limit int) map[string]interface{} {
	total := len(v)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return map[string]interface{}{"items": v[offset:end], "total": total, "offset": offset, "limit": limit}
}
func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	a, b, err := pagination(r)
	if err != nil {
		h.err(w, err)
		return
	}
	v, err := h.service(r).Today(r.Context(), r.URL.Query().Get("timezone"))
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, page(v, a, b))
}
func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	v, err := h.service(r).Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, v)
}
func (h *Handler) part(w http.ResponseWriter, r *http.Request) {
	d, err := h.service(r).Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		h.err(w, err)
		return
	}
	kind := r.PathValue("kind")
	hist := r.URL.Query().Get("history") == "true"
	switch kind {
	case "odds":
		if hist {
			h.json(w, 200, d.Odds)
			return
		}
		h.json(w, 200, d.LatestOdds)
	case "lineup":
		if hist {
			h.json(w, 200, d.Lineups)
		} else {
			h.json(w, 200, d.LatestLineup)
		}

	case "prediction":
		if hist {
			h.json(w, 200, d.Predictions)
		} else {
			h.json(w, 200, d.LatestPrediction)
		}

	case "recommendation":
		if hist {
			h.json(w, 200, d.Recommendations)
		} else {
			h.json(w, 200, d.Recommendation)
		}
	default:
		h.err(w, &usecase.Error{Code: "NOT_FOUND", Message: "ไม่พบข้อมูล", Status: 404})
	}
}
func (h *Handler) picks(w http.ResponseWriter, r *http.Request) {
	a, b, err := pagination(r)
	if err != nil {
		h.err(w, err)
		return
	}
	v, err := h.service(r).Picks(r.Context(), r.URL.Query().Get("include_cancelled") == "true")
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, page(v, a, b))
}
func (h *Handler) pick(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ต้องส่ง JSON", Status: 400})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var input struct {
		RecommendationID string `json:"recommendation_id"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.RecommendationID == "" {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ต้องระบุ recommendation_id เท่านั้น", Status: 400})
		return
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ส่ง JSON ได้หนึ่งรายการ", Status: 400})
		return
	}
	p, err := h.service(r).Pick(r.Context(), input.RecommendationID)
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 201, p)
}
func (h *Handler) pickDetail(w http.ResponseWriter, r *http.Request) {
	p, err := h.service(r).PickByID(r.Context(), r.PathValue("id"))
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, p)
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.service(r).Cancel(r.Context(), r.PathValue("id")); err != nil {
		h.err(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	a, b, err := pagination(r)
	if err != nil {
		h.err(w, err)
		return
	}
	date := r.PathValue("date")
	if date == "" {
		date = r.URL.Query().Get("date")
	}
	v, err := h.service(r).History(r.Context(), date, r.URL.Query().Get("timezone"))
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, map[string]interface{}{"matches": page(v.Cards, a, b), "recommendations": page(v.Recommendations, a, b), "picks": page(v.Picks, a, b), "results": page(v.Results, a, b), "stats": v.Stats})
}
