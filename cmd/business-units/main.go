package main

import (
	"context"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/service"
	"skeji/internal/businessunits/validator"
	"skeji/pkg/logger"
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

	// 7. Initialize HTTP handlers
	// TODO: Initialize router and register handlers
	_ = businessUnitService // Avoid unused variable error for now

	// 8. Start HTTP server
	// TODO: Start HTTP server with graceful shutdown
	// Example:
	// server := &http.Server{
	//     Addr:    ":8080",
	//     Handler: router,
	// }
	//
	// if err := server.ListenAndServe(); err != nil {
	//     log.Fatal("HTTP server failed", "error", err)
	// }

	log.Info("Business Units service ready")

	// TODO: Handle graceful shutdown (listen for signals, drain connections, etc.)
}
