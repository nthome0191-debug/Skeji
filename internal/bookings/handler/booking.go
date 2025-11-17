package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "skeji/internal/bookings/docs" // Import generated swagger docs
	"skeji/internal/bookings/service"
	"skeji/pkg/config"
	apperrors "skeji/pkg/errors"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
	"skeji/pkg/model"

	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger"
)

var _ = httputil.ErrorResponse{}

type BookingHandler struct {
	service service.BookingService
	log     *logger.Logger
}

func NewBookingHandler(service service.BookingService, log *logger.Logger) *BookingHandler {
	return &BookingHandler{
		service: service,
		log:     log,
	}
}

// @Summary Create a new booking
// @Tags Bookings
// @Accept json
// @Produce json
// @Param booking body model.Booking true "Booking data"
// @Success 201 {object} model.Booking
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings [post]
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var booking model.Booking
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Invalid request body",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Create", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if err := h.service.Create(r.Context(), &booking); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Create", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteCreated(w, booking); err != nil {
		h.log.Error("failed to write created response", "handler", "Create", "operation", "WriteCreated", "error", err)
	}
}

// @Summary Get booking by ID
// @Tags Bookings
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} model.Booking
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings/id/{id} [get]
func (h *BookingHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	booking, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetByID", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteSuccess(w, booking); err != nil {
		h.log.Error("failed to write success response", "handler", "GetByID", "operation", "WriteSuccess", "error", err)
	}
}

// @Summary Get all bookings
// @Tags Bookings
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} httputil.PaginatedResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings [get]
func (h *BookingHandler) GetAll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()

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

	limit = config.NormalizePaginationLimit(limit)
	offset = config.NormalizeOffset(offset)

	bookings, total, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "GetAll", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WritePaginated(w, bookings, total, limit, offset); err != nil {
		h.log.Error("failed to write paginated response", "handler", "GetAll", "operation", "WritePaginated", "error", err)
	}
}

// @Summary Update booking
// @Tags Bookings
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Param booking body model.BookingUpdate true "Booking update"
// @Success 204 "No Content"
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings/id/{id} [patch]
func (h *BookingHandler) Update(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	var updates model.BookingUpdate
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

// @Summary Delete booking
// @Tags Bookings
// @Produce json
// @Param id path string true "Booking ID"
// @Success 204 "No Content"
// @Failure 404 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings/id/{id} [delete]
func (h *BookingHandler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	if err := h.service.Delete(r.Context(), id); err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Delete", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	httputil.WriteNoContent(w)
}

// @Summary Search bookings by business, schedule, and time range
// @Tags Bookings
// @Produce json
// @Param business_id query string true "Business ID"
// @Param schedule_id query string true "Schedule ID"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Success 200 {array} model.Booking
// @Failure 400 {object} httputil.ErrorResponse
// @Failure 500 {object} httputil.ErrorResponse
// @Router /api/v1/bookings/search [get]
func (h *BookingHandler) Search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	businessID := query.Get("business_id")
	scheduleID := query.Get("schedule_id")
	startStr := query.Get("start_time")
	endStr := query.Get("end_time")

	if businessID == "" || scheduleID == "" {
		if writeErr := httputil.WriteJSON(w, http.StatusBadRequest, httputil.ErrorResponse{
			Error: "Both 'business_id' and 'schedule_id' query parameters are required",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Search", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	startStr = strings.ReplaceAll(startStr, " ", "+")
	endStr = strings.ReplaceAll(endStr, " ", "+")

	var startTime, endTime *time.Time
	if startStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &parsed
		} else {
			if writeErr := httputil.WriteError(w, apperrors.InvalidInput("invalid start_time format, must be RFC3339")); writeErr != nil {
				h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
			}
			return
		}
	}
	if endStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &parsed
		} else {
			if writeErr := httputil.WriteError(w, apperrors.InvalidInput("invalid end_time format, must be RFC3339")); writeErr != nil {
				h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
			}
			return
		}
	}

	bookings, err := h.service.SearchBySchedule(r.Context(), businessID, scheduleID, startTime, endTime)
	if err != nil {
		if writeErr := httputil.WriteError(w, err); writeErr != nil {
			h.log.Error("failed to write error response", "handler", "Search", "operation", "WriteError", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteSuccess(w, bookings); err != nil {
		h.log.Error("failed to write success response", "handler", "Search", "operation", "WriteSuccess", "error", err)
	}
}

func (h *BookingHandler) RegisterRoutes(router *httprouter.Router) {
	// Swagger UI routes
	router.Handler("GET", "/swagger/*any", httpSwagger.WrapHandler)

	// API routes
	router.POST("/api/v1/bookings", h.Create)
	router.GET("/api/v1/bookings", h.GetAll)
	router.GET("/api/v1/bookings/search", h.Search)
	router.GET("/api/v1/bookings/id/:id", h.GetByID)
	router.PATCH("/api/v1/bookings/id/:id", h.Update)
	router.DELETE("/api/v1/bookings/id/:id", h.Delete)
}
