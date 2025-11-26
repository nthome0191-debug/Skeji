package main

import (
	"net/http"
	"os"
	"skeji/internal/maestro/api"
	"skeji/pkg/client"
	"skeji/pkg/logger"
)

func main() {
	log := logger.New(logger.Config{
		Level:   logger.INFO,
		Format:  logger.JSON,
		Service: "maestro",
	})

	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	port := os.Getenv("MAESTRO_PORT")
	if port == "" {
		port = "8090"
	}

	apiClient := client.NewClient()
	apiClient.SetBusinessUnitClient(baseURL)
	apiClient.SetScheduleClient(baseURL)
	apiClient.SetBookingClient(baseURL)

	router := api.SetupRouter(apiClient, log)

	addr := ":" + port
	log.Info("Starting Maestro API server", "address", addr, "base_url", baseURL)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
