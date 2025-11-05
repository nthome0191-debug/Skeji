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
	httputil.WriteJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		h.log.Error("Database health check failed",
			"error", err,
			"path", r.URL.Path,
		)
		httputil.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status:   "unavailable",
			Database: "error",
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:   "ready",
		Database: "ok",
	})
}

func (h *HealthHandler) RegisterRoutes(router *httprouter.Router) {
	router.GET("/health", h.Health)
	router.GET("/ready", h.Ready)
}
