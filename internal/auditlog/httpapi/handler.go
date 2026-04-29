package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"audit-log/internal/auditlog/httpapi/internal/mapconv"
	"audit-log/internal/auditlog/usecases"
)

type Handler struct {
	svc usecases.AuditService
}

func NewHandler(svc usecases.AuditService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", h.queryEvents)
	mux.HandleFunc("GET /api/namespaces", h.listNamespaces)
}

func (h *Handler) queryEvents(w http.ResponseWriter, r *http.Request) {
	opts, err := mapconv.QueryParamsToOpts(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.QueryEvents(r.Context(), opts)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidPageSize) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, mapconv.QueryResultToResponse(res))
}

func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	ns, err := h.svc.ListNamespaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, mapconv.NamespacesToResponse(ns))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
