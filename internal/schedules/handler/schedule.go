package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"skeji/internal/schedules/service"
	apperrors "skeji/pkg/errors"
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

func (h *ScheduleHandler) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()

	// Validate limit parameter
	limit := 0
	if limitStr := query.Get("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			if writeErr := httputil.WriteError(w, apperrors.InvalidInput(fmt.Sprintf("invalid limit parameter: %s", limitStr))); writeErr != nil {
				h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
			}
			return
		}
	}

	// Validate offset parameter
	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			if writeErr := httputil.WriteError(w, apperrors.InvalidInput(fmt.Sprintf("invalid offset parameter: %s", offsetStr))); writeErr != nil {
				h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
			}
			return
		}
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

	results, err := h.service.Search(r.Context(), businessID, city)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteSuccess(w, results); err != nil {
		h.log.Error("failed to write success response", "handler", "Search", "operation", "WriteSuccess", "error", err)
	}
}

func (h *ScheduleHandler) RegisterRoutes(router *httprouter.Router) {
	router.POST("/api/v1/schedules", h.Create)
	router.GET("/api/v1/schedules", h.GetAll)
	router.GET("/api/v1/schedules/search", h.Search)
	router.GET("/api/v1/schedules/id/:id", h.GetByID)
	router.PATCH("/api/v1/schedules/id/:id", h.Update)
	router.DELETE("/api/v1/schedules/id/:id", h.Delete)
}
