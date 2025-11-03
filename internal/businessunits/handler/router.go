package handler

import (
	"net/http"
	"strings"
)

func (h *BusinessUnitHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/business-units", h.handleBusinessUnitsRoot)
	mux.HandleFunc("/api/v1/business-units/", h.handleBusinessUnitsWithID)
	mux.HandleFunc("/api/v1/business-units/search", h.Search)
}

func (h *BusinessUnitHandler) handleBusinessUnitsRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/business-units" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.Create(w, r)
	case http.MethodGet:
		h.GetAll(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BusinessUnitHandler) handleBusinessUnitsWithID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/search") {
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/business-units/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r)
	case http.MethodPatch:
		h.Update(w, r)
	case http.MethodDelete:
		h.Delete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
