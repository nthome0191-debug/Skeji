package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "skeji/internal/businessunits/docs" // Import generated swagger docs
	"skeji/internal/businessunits/service"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
	"skeji/pkg/model"
)

type BusinessUnitHandler struct {
	service service.BusinessUnitService
	log     *logger.Logger
}

func NewBusinessUnitHandler(service service.BusinessUnitService, log *logger.Logger) *BusinessUnitHandler {
	return &BusinessUnitHandler{
		service: service,
		log:     log,
	}
}

// @Summary Create a new business unit
// @Tags BusinessUnits
// @Accept json
// @Produce json
// @Param business_unit body model.BusinessUnit true "Business unit data"
// @Success 201 {object} model.BusinessUnit
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units [post]
func (h *BusinessUnitHandler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var bu model.BusinessUnit
	if err := json.NewDecoder(r.Body).Decode(&bu); err != nil {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Create", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if err := h.service.Create(r.Context(), &bu); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Create", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteCreated(w, bu); err != nil {
		h.log.Error("failed to write created response", "handler", "Create", "operation", "WriteCreated", "error", err)
	}
}

// @Summary Get business unit by ID
// @Tags BusinessUnits
// @Produce json
// @Param id path string true "Business Unit ID"
// @Success 200 {object} model.BusinessUnit
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units/id/{id} [get]
func (h *BusinessUnitHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	bu, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetByID", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteSuccess(w, bu); err != nil {
		h.log.Error("failed to write success response", "handler", "GetByID", "operation", "WriteSuccess", "error", err)
	}
}

// @Summary Get business units by phone
// @Description Returns all business units where the phone is either the admin or a maintainer
// @Tags BusinessUnits
// @Produce json
// @Param phone path string true "Phone Number"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units/phone/{phone} [get]
func (h *BusinessUnitHandler) GetByPhone(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	phone := ps.ByName("phone")

	limit, offset, err := httputil.ExtractLimitOffset(r)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetByPhone", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	units, totalCount, err := h.service.GetByPhone(r.Context(), phone, limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetByPhone", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, units, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "GetByPhone", "operation", "WritePaginated", "error", err)
	}
}

// @Summary Get all business units
// @Tags BusinessUnits
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units [get]
func (h *BusinessUnitHandler) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	limit, offset, err := httputil.ExtractLimitOffset(r)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	units, totalCount, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, units, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "GetAll", "operation", "WritePaginated", "error", err)
	}
}

// @Summary Update business unit
// @Tags BusinessUnits
// @Accept json
// @Produce json
// @Param id path string true "Business Unit ID"
// @Param business_unit body model.BusinessUnitUpdate true "Business unit update"
// @Success 204 "No Content"
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units/id/{id} [patch]
func (h *BusinessUnitHandler) Update(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	var updates model.BusinessUnitUpdate
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

// @Summary Delete business unit
// @Tags BusinessUnits
// @Produce json
// @Param id path string true "Business Unit ID"
// @Success 204 "No Content"
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units/id/{id} [delete]
func (h *BusinessUnitHandler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	if err := h.service.Delete(r.Context(), id); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Delete", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	httputil.WriteNoContent(w)
}

// @Summary Search business units by cities and labels
// @Tags BusinessUnits
// @Produce json
// @Param cities query string true "Comma-separated cities"
// @Param labels query string true "Comma-separated labels"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/business-units/search [get]
func (h *BusinessUnitHandler) Search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	citiesParam := query.Get("cities")
	labelsParam := query.Get("labels")

	if citiesParam == "" || labelsParam == "" {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Both 'cities' and 'labels' query parameters are required",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Search", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	cities := splitAndTrim(citiesParam)
	labels := splitAndTrim(labelsParam)

	if len(cities) == 0 || len(labels) == 0 {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Cities and labels must contain at least one non-empty value",
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

	units, totalCount, err := h.service.Search(r.Context(), cities, labels, limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, units, totalCount, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "Search", "operation", "WritePaginated", "error", err)
	}
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
	// Swagger UI routes
	router.Handler("GET", "/swagger/*any", httpSwagger.WrapHandler)

	// API routes
	router.POST("/api/v1/business-units", h.Create)
	router.GET("/api/v1/business-units", h.GetAll)
	router.GET("/api/v1/business-units/search", h.Search)
	router.GET("/api/v1/business-units/phone/:phone", h.GetByPhone)
	router.GET("/api/v1/business-units/id/:id", h.GetByID)
	router.PATCH("/api/v1/business-units/id/:id", h.Update)
	router.DELETE("/api/v1/business-units/id/:id", h.Delete)
}
