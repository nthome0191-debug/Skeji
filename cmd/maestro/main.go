package main

import (
	"net/http"
	"os"
	"skeji/internal/maestro/api"
	"skeji/pkg/client"
	"skeji/pkg/config"
)

const ServiceName = "maestro"

func main() {
	cfg := config.Load(ServiceName)

	cfg.Log.Info("Starting maestro service")

	apiClient := client.NewClient()
	apiClient.SetBusinessUnitClient(cfg.BusinessUnitBaseUrl)
	apiClient.SetScheduleClient(cfg.ScheduleBaseUrl)
	apiClient.SetBookingClient(cfg.BookingBaseUrl)

	router := api.SetupRouter(apiClient, cfg.Log)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		cfg.Log.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
