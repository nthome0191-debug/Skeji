package api

import (
	"net/http"
	"skeji/internal/maestro/handlers"
	"skeji/internal/maestro/service"
	"skeji/pkg/client"
	"skeji/pkg/logger"
)

func SetupRouter(client *client.Client, log *logger.Logger) *http.ServeMux {
	maestroService := service.NewMaestroService(client, log)
	flowHandler := handlers.NewFlowHandler(maestroService, log)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/maestro/execute", flowHandler.ExecuteFlow)
	mux.HandleFunc("/api/v1/maestro/flows", flowHandler.ListFlows)
	mux.HandleFunc("/api/v1/maestro/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	return mux
}
