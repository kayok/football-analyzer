package http

import (
	"encoding/json"
	"football/internal/application/usecase"
	"io"
	"net"
	"net/http"
	"time"
)

const sessionCookie = "football_session"

type attempt struct {
	start time.Time
	count int
}

func (h *Handler) allowLogin(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	now := h.clock.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	for key, v := range h.attempts {
		if now.Sub(v.start) >= time.Minute {
			delete(h.attempts, key)
		}
	}
	v := h.attempts[host]
	if v.start.IsZero() {
		v.start = now
	}
	if v.count >= 10 || len(h.attempts) >= 1000 {
		return false
	}
	v.count++
	h.attempts[host] = v
	return true
}

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) credentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var input credentials
	if !h.allowLogin(r) {
		h.err(w, &usecase.Error{Code: "RATE_LIMITED", Message: "ลองเข้าสู่ระบบบ่อยเกินไป กรุณารอ 1 นาที", Status: 429})
		return input, false
	}
	if r.Header.Get("Content-Type") != "application/json" {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ต้องส่ง JSON", Status: 400})
		return input, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ข้อมูลสมัครหรือเข้าสู่ระบบไม่ถูกต้อง", Status: 400})
		return input, false
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		h.err(w, &usecase.Error{Code: "INVALID_INPUT", Message: "ส่ง JSON ได้หนึ่งรายการ", Status: 400})
		return input, false
	}
	return input, true
}
func (h *Handler) cookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	in, ok := h.credentials(w, r)
	if !ok {
		return
	}
	user, token, err := h.members.Register(r.Context(), in.Name, in.Email, in.Password)
	if err != nil {
		h.err(w, err)
		return
	}
	h.cookie(w, r, token, int(usecase.SessionLifetime.Seconds()))
	h.json(w, 201, user)
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	in, ok := h.credentials(w, r)
	if !ok {
		return
	}
	user, token, err := h.members.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		h.err(w, err)
		return
	}
	// Revoke this browser's previous session before replacing its cookie.
	if previous, err := r.Cookie(sessionCookie); err == nil {
		if err = h.members.Logout(r.Context(), previous.Value); err != nil {
			h.err(w, err)
			return
		}
	}
	h.cookie(w, r, token, int(usecase.SessionLifetime.Seconds()))
	h.json(w, 200, user)
}
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(sessionCookie)
	user, err := h.members.Current(r.Context(), c.Value)
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, user)
}
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(sessionCookie)
	if err := h.members.Logout(r.Context(), c.Value); err != nil {
		h.err(w, err)
		return
	}
	h.cookie(w, r, "", -1)
	w.WriteHeader(204)
}
