package http

import (
	"football/internal/application/usecase"
	"net/http"
)

func (h *Handler) syncStatus(w http.ResponseWriter, r *http.Request) {
	if h.syncer == nil {
		h.err(w, &usecase.Error{Code: "SYNC_DISABLED", Message: "ยังไม่ได้เปิดระบบซิงก์", Status: 503})
		return
	}
	status, err := h.syncer.Status(r.Context())
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 200, status)
}
func (h *Handler) startSync(w http.ResponseWriter, r *http.Request) {
	if h.syncer == nil {
		h.err(w, &usecase.Error{Code: "SYNC_DISABLED", Message: "ยังไม่ได้เปิดระบบซิงก์", Status: 503})
		return
	}
	status, err := h.syncer.Start(r.Context(), "owner")
	if err != nil {
		h.err(w, err)
		return
	}
	h.json(w, 202, status)
}
