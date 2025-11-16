package main

import (
	"skeji/internal/businessunits/handler"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/service"
	"skeji/internal/businessunits/validator"
	"skeji/pkg/app"
	"skeji/pkg/config"
)

const ServiceName = "business-units"

func main() {
	cfg := config.Load(ServiceName)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		cfg.Log.Fatal("Invalid configuration", "error", err)
	}

	// Log all configuration values
	cfg.LogConfiguration()

	cfg.Log.Info("Starting Business Units service")
	businessUnitService := initServices(cfg)
	serverApp := app.NewApplication(cfg)
	serverApp.SetApp(handler.NewBusinessUnitHandler(businessUnitService, cfg.Log))
	serverApp.Run()
}

func initServices(cfg *config.Config) service.BusinessUnitService {
	businessUnitValidator := validator.NewBusinessUnitValidator(cfg.Log)
	businessUnitRepo := repository.NewMongoBusinessUnitRepository(cfg)
	businessUnitService := service.NewBusinessUnitService(
		businessUnitRepo,
		businessUnitValidator,
		cfg,
	)

	cfg.Log.Info("Business unit service initialized", "database", cfg.MongoDatabaseName)
	return businessUnitService
}
