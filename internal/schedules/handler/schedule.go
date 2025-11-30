package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "skeji/internal/schedules/docs" // Import generated swagger docs
	"skeji/internal/schedules/service"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
	"skeji/pkg/model"
)

type ScheduleHandler struct {
	service service.ScheduleService
	log     *logger.Logger
}

func NewScheduleHandler(service service.ScheduleService, log *logger.Logger) *ScheduleHandler {
	return &ScheduleHandler{
		service: service,
		log:     log,
	}
}

// @Summary Create a new schedule
// @Tags Schedules
// @Accept json
// @Produce json
// @Param schedule body model.Schedule true "Schedule data"
// @Success 201 {object} model.Schedule
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules [post]
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var sc model.Schedule
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Create", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if err := h.service.Create(r.Context(), &sc); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Create", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteCreated(w, sc); err != nil {
		h.log.Error("failed to write created response", "handler", "Create", "operation", "WriteCreated", "error", err)
	}
}

// @Summary Get schedule by ID
// @Tags Schedules
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} model.Schedule
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules/id/{id} [get]
func (h *ScheduleHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		if err := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "ID parameter is required",
		}); err != nil {
			h.log.Error("failed to write bad request response", "handler", "GetByID", "operation", "WriteJSON", "error", err)
		}
		return
	}

	sc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetByID", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteSuccess(w, sc); err != nil {
		h.log.Error("failed to write success response", "handler", "GetByID", "operation", "WriteSuccess", "error", err)
	}
}

// @Summary Get all schedules
// @Tags Schedules
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules [get]
func (h *ScheduleHandler) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	limit, offset, err := httputil.ExtractLimitOffset(r)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	schedules, totalCount, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, schedules, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "GetAll", "operation", "WritePaginated", "error", err)
	}
}

// @Summary Update schedule
// @Tags Schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Param schedule body model.ScheduleUpdate true "Schedule update"
// @Success 204 "No Content"
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules/id/{id} [patch]
func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		if err := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "ID parameter is required",
		}); err != nil {
			h.log.Error("failed to write bad request response", "handler", "Update", "operation", "WriteJSON", "error", err)
		}
		return
	}

	var updates model.ScheduleUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Update", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if err := h.service.Update(r.Context(), id, &updates); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Update", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	httputil.WriteNoContent(w)
}

// @Summary Delete schedule
// @Tags Schedules
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 204 "No Content"
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules/id/{id} [delete]
func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		if err := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "ID parameter is required",
		}); err != nil {
			h.log.Error("failed to write bad request response", "handler", "Delete", "operation", "WriteJSON", "error", err)
		}
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Delete", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	httputil.WriteNoContent(w)
}

// @Summary Search schedules by business and city
// @Tags Schedules
// @Produce json
// @Param business_id query string true "Business ID"
// @Param city query string false "City"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules/search [get]
func (h *ScheduleHandler) Search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	businessID := strings.TrimSpace(query.Get("business_id"))
	city := strings.TrimSpace(query.Get("city"))

	if businessID == "" {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "'business_id' query parameter is required",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Search", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	limit, offset, err := httputil.ExtractLimitOffset(r)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	results, totalCount, err := h.service.Search(r.Context(), businessID, city, limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, results, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "Search", "operation", "WritePaginated", "error", err)
	}
}

// @Summary Batch search schedules across multiple cities
// @Tags Schedules
// @Produce json
// @Param business_id query string true "Business ID"
// @Param cities query string true "Comma-separated list of cities"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/schedules/batch-search [get]
func (h *ScheduleHandler) BatchSearch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	businessID := strings.TrimSpace(query.Get("business_id"))
	citiesParam := strings.TrimSpace(query.Get("cities"))

	if businessID == "" {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "'business_id' query parameter is required",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "BatchSearch", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if citiesParam == "" {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "'cities' query parameter is required (comma-separated)",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "BatchSearch", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	cities := strings.Split(citiesParam, ",")
	for i := range cities {
		cities[i] = strings.TrimSpace(cities[i])
	}

	limit, offset, err := httputil.ExtractLimitOffset(r)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "BatchSearch", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	results, totalCount, err := h.service.BatchSearch(r.Context(), businessID, cities, limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "BatchSearch", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, results, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "BatchSearch", "operation", "WritePaginated", "error", err)
	}
}

func (h *ScheduleHandler) RegisterRoutes(router *httprouter.Router) {
	// Swagger UI routes
	router.Handler("GET", "/swagger/*any", httpSwagger.WrapHandler)

	// API routes
	router.POST("/api/v1/schedules", h.Create)
	router.GET("/api/v1/schedules", h.GetAll)
	router.GET("/api/v1/schedules/search", h.Search)
	router.GET("/api/v1/schedules/batch-search", h.BatchSearch)
	router.GET("/api/v1/schedules/id/:id", h.GetByID)
	router.PATCH("/api/v1/schedules/id/:id", h.Update)
	router.DELETE("/api/v1/schedules/id/:id", h.Delete)
}
