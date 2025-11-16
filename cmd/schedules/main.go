package main

import (
	"skeji/internal/schedules/handler"
	"skeji/internal/schedules/repository"
	"skeji/internal/schedules/service"
	"skeji/internal/schedules/validator"
	"skeji/pkg/app"
	"skeji/pkg/config"
)

const ServiceName = "schedules"

func main() {
	cfg := config.Load(ServiceName)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		cfg.Log.Fatal("Invalid configuration", "error", err)
	}

	// Log all configuration values
	cfg.LogConfiguration()

	cfg.Log.Info("Starting Schedules service")
	scheduleService := initServices(cfg)
	serverApp := app.NewApplication(cfg)
	serverApp.SetApp(handler.NewScheduleHandler(scheduleService, cfg.Log))
	serverApp.Run()
}

func initServices(cfg *config.Config) service.ScheduleService {
	businessUnitValidator := validator.NewScheduleValidator(cfg.Log)
	businessUnitRepo := repository.NewMongoScheduleRepository(cfg)
	businessUnitService := service.NewScheduleService(
		businessUnitRepo,
		businessUnitValidator,
		cfg,
	)

	cfg.Log.Info("Schedules service initialized", "database", cfg.MongoDatabaseName)
	return businessUnitService
}
