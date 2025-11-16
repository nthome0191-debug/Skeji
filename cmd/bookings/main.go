package main

import (
	"skeji/internal/bookings/handler"
	"skeji/internal/bookings/repository"
	"skeji/internal/bookings/service"
	"skeji/internal/bookings/validator"
	"skeji/pkg/app"
	"skeji/pkg/config"
)

const ServiceName = "bookings"

func main() {
	cfg := config.Load(ServiceName)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		cfg.Log.Fatal("Invalid configuration", "error", err)
	}

	// Log all configuration values
	cfg.LogConfiguration()

	cfg.Log.Info("Starting Bookings service")
	bookingService := initServices(cfg)
	serverApp := app.NewApplication(cfg)
	serverApp.SetApp(handler.NewBookingHandler(bookingService, cfg.Log))
	serverApp.Run()
}

func initServices(cfg *config.Config) service.BookingService {
	bookingValidator := validator.NewBookingValidator(cfg.Log)
	bookingRepo := repository.NewMongoBookingRepository(cfg)
	bookingService := service.NewBookingService(
		bookingRepo,
		bookingValidator,
		cfg,
	)

	cfg.Log.Info("Booking service initialized", "database", cfg.MongoDatabaseName)
	return bookingService
}
