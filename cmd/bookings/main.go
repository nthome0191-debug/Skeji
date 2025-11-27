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

// @title Skeji Bookings API
// @version 1.0
// @description API documentation for the Bookings microservice.
// @BasePath /
func main() {
	cfg := config.Load(ServiceName)
	cfg.SetMongo()

	cfg.Log.Info("Starting Bookings service")
	bookingService := initServices(cfg)
	serverApp := app.NewApplication(cfg)
	serverApp.SetApp(handler.NewBookingHandler(bookingService, cfg.Log))
	serverApp.Run()
}

func initServices(cfg *config.Config) service.BookingService {
	bookingValidator := validator.NewBookingValidator(cfg.Log)
	bookingRepo := repository.NewMongoBookingRepository(cfg)
	bookingLockRepo := repository.NewBookingLockRepository(cfg)
	bookingService := service.NewBookingService(
		bookingRepo,
		bookingLockRepo,
		bookingValidator,
		cfg,
	)

	cfg.Log.Info("Booking service initialized", "database", cfg.MongoDatabaseName)
	return bookingService
}
