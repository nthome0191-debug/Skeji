package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"skeji/internal/businessunits/handler"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/service"
	"skeji/internal/businessunits/validator"
	"skeji/pkg/logger"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log := logger.New(logger.Config{
		Level:     logger.INFO,
		Format:    logger.JSON,
		AddSource: true,
		Service:   "business-units",
	})
	log.Info("Starting Business Units service")
	mongoURI := "mongodb://localhost:27017" // TODO: Load from config
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB",
			"error", err,
			"uri", mongoURI,
		)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB",
			"error", err,
		)
	}

	log.Info("Successfully connected to MongoDB")

	businessUnitValidator := validator.NewBusinessUnitValidator(log)
	businessUnitRepo := repository.NewMongoBusinessUnitRepository(client)
	businessUnitService := service.NewBusinessUnitService(
		businessUnitRepo,
		businessUnitValidator,
		log,
	)

	mux := http.NewServeMux()
	businessUnitHandler := handler.NewBusinessUnitHandler(businessUnitService, log)
	businessUnitHandler.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server",
			"port", port,
			"address", server.Addr,
		)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal("HTTP server failed", "error", err)

	case sig := <-shutdown:
		log.Info("Shutdown signal received",
			"signal", sig,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("Server shutdown failed",
				"error", err,
			)
			if err := server.Close(); err != nil {
				log.Fatal("Could not stop server gracefully", "error", err)
			}
		}

		log.Info("Server stopped gracefully")
	}
}
