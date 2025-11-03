package handler

import (
	"encoding/json"
	"net/http"
	"skeji/internal/businessunits/service"
	httputil "skeji/pkg/http"
	"skeji/pkg/model"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type BusinessUnitHandler struct {
	service service.BusinessUnitService
}

func NewBusinessUnitHandler(service service.BusinessUnitService) *BusinessUnitHandler {
	return &BusinessUnitHandler{
		service: service,
	}
}

func (h *BusinessUnitHandler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var bu model.BusinessUnit
	if err := json.NewDecoder(r.Body).Decode(&bu); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	if err := h.service.Create(r.Context(), &bu); err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteCreated(w, bu)
}

func (h *BusinessUnitHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	bu, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteSuccess(w, bu)
}

func (h *BusinessUnitHandler) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	units, totalCount, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WritePaginated(w, units, totalCount, limit, offset)
}

func (h *BusinessUnitHandler) Update(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	var updates model.BusinessUnitUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	if err := h.service.Update(r.Context(), id, &updates); err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

func (h *BusinessUnitHandler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	if err := h.service.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

func (h *BusinessUnitHandler) Search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	citiesParam := query.Get("cities")
	labelsParam := query.Get("labels")

	if citiesParam == "" || labelsParam == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Both 'cities' and 'labels' query parameters are required",
		})
		return
	}

	cities := splitAndTrim(citiesParam)
	labels := splitAndTrim(labelsParam)

	if len(cities) == 0 || len(labels) == 0 {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Cities and labels must contain at least one non-empty value",
		})
		return
	}

	units, err := h.service.Search(r.Context(), cities, labels)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteSuccess(w, units)
}

func splitAndTrim(param string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(param, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func (h *BusinessUnitHandler) RegisterRoutes(router *httprouter.Router) {
	router.POST("/api/v1/business-units", h.Create)
	router.GET("/api/v1/business-units", h.GetAll)
	router.GET("/api/v1/business-units/search", h.Search)
	router.GET("/api/v1/business-units/:id", h.GetByID)
	router.PATCH("/api/v1/business-units/:id", h.Update)
	router.DELETE("/api/v1/business-units/:id", h.Delete)
}
