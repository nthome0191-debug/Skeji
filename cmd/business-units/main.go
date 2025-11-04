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
	"skeji/pkg/middleware"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log := initLogger()
	log.Info("Starting Business Units service")

	mongoClient := connectMongoDB(log)
	defer mongoClient.Disconnect(context.Background())

	businessUnitService := initServices(mongoClient, log)

	server := setupHTTPServer(businessUnitService, mongoClient, log)

	run(server, log)
}

func initLogger() *logger.Logger {
	return logger.New(logger.Config{
		Level:     logger.INFO,
		Format:    logger.JSON,
		AddSource: true,
		Service:   "business-units",
	})
}

func connectMongoDB(log *logger.Logger) *mongo.Client {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB",
			"error", err,
			"uri", mongoURI,
		)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB", "error", err)
	}

	log.Info("Successfully connected to MongoDB")
	return client
}

func initServices(mongoClient *mongo.Client, log *logger.Logger) service.BusinessUnitService {
	businessUnitValidator := validator.NewBusinessUnitValidator(log)
	businessUnitRepo := repository.NewMongoBusinessUnitRepository(mongoClient)
	businessUnitService := service.NewBusinessUnitService(
		businessUnitRepo,
		businessUnitValidator,
		log,
	)

	log.Info("Business unit service initialized")
	return businessUnitService
}

func setupHTTPServer(businessUnitService service.BusinessUnitService, mongoClient *mongo.Client, log *logger.Logger) *http.Server {
	router := httprouter.New()

	healthHandler := handler.NewHealthHandler(mongoClient)
	healthHandler.RegisterRoutes(router)

	businessUnitHandler := handler.NewBusinessUnitHandler(businessUnitService)
	businessUnitHandler.RegisterRoutes(router)

	idempotencyStore := middleware.NewInMemoryIdempotencyStore(24 * time.Hour)
	phoneRateLimiter := middleware.NewPhoneRateLimiter(
		10,                        // 10 requests
		1*time.Minute,             // per minute
		middleware.DefaultPhoneExtractor,
		log,
	)

	var handler http.Handler = router
	handler = middleware.MaxRequestSize(1024 * 1024)(handler)
	handler = middleware.Idempotency(idempotencyStore, "Idempotency-Key")(handler)
	handler = middleware.RequestTimeout(30 * time.Second)(handler)
	handler = middleware.RequestLogging(log)(handler)
	handler = middleware.PhoneRateLimit(phoneRateLimiter)(handler)
	handler = middleware.ContentTypeValidation(log)(handler)

	whatsappSecret := os.Getenv("WHATSAPP_APP_SECRET")
	if whatsappSecret != "" {
		handler = middleware.WhatsAppSignatureVerification(whatsappSecret, log)(handler)
		log.Info("WhatsApp signature verification enabled")
	}

	handler = middleware.Recovery(log)(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("HTTP server configured", "port", port)
	return server
}

func run(server *http.Server, log *logger.Logger) {
	serverErrors := make(chan error, 1)

	go func() {
		log.Info("Starting HTTP server", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal("HTTP server failed", "error", err)

	case sig := <-shutdown:
		log.Info("Shutdown signal received", "signal", sig)
		gracefulShutdown(server, log)
	}
}

func gracefulShutdown(server *http.Server, log *logger.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", "error", err)
		if err := server.Close(); err != nil {
			log.Fatal("Could not stop server gracefully", "error", err)
		}
	}

	log.Info("Server stopped gracefully")
}
