package handler

import (
	"encoding/json"
	"net/http"
	"skeji/internal/businessunits/service"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strconv"
	"strings"
)

type BusinessUnitHandler struct {
	service service.BusinessUnitService
	logger  *logger.Logger
}

func NewBusinessUnitHandler(service service.BusinessUnitService, logger *logger.Logger) *BusinessUnitHandler {
	return &BusinessUnitHandler{
		service: service,
		logger:  logger,
	}
}

func (h *BusinessUnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (h *BusinessUnitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/business-units/")
	if id == "" || strings.Contains(id, "/") {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid business unit ID",
		})
		return
	}

	bu, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteSuccess(w, bu)
}

func (h *BusinessUnitHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (h *BusinessUnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/business-units/")
	if id == "" || strings.Contains(id, "/") {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid business unit ID",
		})
		return
	}

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

func (h *BusinessUnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/business-units/")
	if id == "" || strings.Contains(id, "/") {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid business unit ID",
		})
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

func (h *BusinessUnitHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	citiesParam := query.Get("cities")
	labelsParam := query.Get("labels")

	if citiesParam == "" || labelsParam == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Both 'cities' and 'labels' query parameters are required",
		})
		return
	}

	cities := strings.Split(citiesParam, ",")
	labels := strings.Split(labelsParam, ",")

	for i := range cities {
		cities[i] = strings.TrimSpace(cities[i])
	}
	for i := range labels {
		labels[i] = strings.TrimSpace(labels[i])
	}

	units, err := h.service.Search(r.Context(), cities, labels)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	httputil.WriteSuccess(w, units)
}
