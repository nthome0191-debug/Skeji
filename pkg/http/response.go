package http

import (
	"encoding/json"
	"net/http"
	apperrors "skeji/pkg/errors"
)

type ErrorResponse struct {
	Error   string         `json:"error"`
	Details map[string]any `json:"details,omitempty"`
}

type SuccessResponse struct {
	Data any `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Data       any   `json:"data"`
	TotalCount int64 `json:"total_count"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, err error) {
	var statusCode int
	var errResp ErrorResponse

	switch e := err.(type) {
	case *apperrors.AppError:
		switch e.Code {
		case apperrors.CodeInvalidInput:
			statusCode = http.StatusBadRequest
		case apperrors.CodeNotFound:
			statusCode = http.StatusNotFound
		case apperrors.CodeValidation:
			statusCode = http.StatusUnprocessableEntity
		case apperrors.CodeInternal:
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}
		errResp = ErrorResponse{
			Error:   e.Message,
			Details: e.Details,
		}
	default:
		statusCode = http.StatusInternalServerError
		errResp = ErrorResponse{
			Error: "Internal server error",
		}
	}

	WriteJSON(w, statusCode, errResp)
}

func WriteSuccess(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, SuccessResponse{Data: data})
}

func WriteCreated(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusCreated, SuccessResponse{Data: data})
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WritePaginated(w http.ResponseWriter, data any, totalCount int64, limit int, offset int) {
	WriteJSON(w, http.StatusOK, PaginatedResponse{
		Data:       data,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	})
}
