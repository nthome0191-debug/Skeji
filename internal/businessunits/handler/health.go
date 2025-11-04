package handler

import (
	"context"
	"net/http"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
)

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

func (h *HealthHandler) RegisterRoutes(router *httprouter.Router) {
	router.GET("/health", h.Health)
	router.GET("/ready", h.Ready)
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		h.log.Error("Database health check failed",
			"error", err,
			"path", r.URL.Path,
		)
		httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  "database unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready","database":"ok"}`))
}
