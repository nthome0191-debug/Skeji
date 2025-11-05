package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"

	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
)

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

type HealthHandler struct {
	mongoClient *mongo.Client
	log         *logger.Logger
}

func NewHealthHandler(mongoClient *mongo.Client, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		mongoClient: mongoClient,
		log:         log,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := httputil.WriteJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	}); err != nil {
		h.log.Error("failed to write JSON response", "handler", "Health", "operation", "WriteJSON", "error", err)
	}
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		h.log.Error("Database health check failed",
			"error", err,
			"path", r.URL.Path,
		)
		if writeErr := httputil.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status:   "unavailable",
			Database: "error",
		}); writeErr != nil {
			h.log.Error("failed to write JSON response", "handler", "Ready", "operation", "WriteJSON", "error", writeErr)
		}
		return
	}

	if err := httputil.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:   "ready",
		Database: "ok",
	}); err != nil {
		h.log.Error("failed to write JSON response", "handler", "Ready", "operation", "WriteJSON", "error", err)
	}
}

func (h *HealthHandler) RegisterRoutes(router *httprouter.Router) {
	router.GET("/health", h.Health)
	router.GET("/ready", h.Ready)
}
