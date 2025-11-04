package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"skeji/internal/businessunits/handler"
	"skeji/internal/businessunits/repository"
	"skeji/internal/businessunits/service"
	"skeji/internal/businessunits/validator"
	"skeji/internal/config"
	"skeji/pkg/logger"
	"skeji/pkg/middleware"
)

func main() {
	cfg := config.Load()

	log := initLogger()
	log.Info("Starting Business Units service")

	mongoClient := connectMongoDB(cfg, log)
	defer mongoClient.Disconnect(context.Background())

	businessUnitService := initServices(mongoClient, log)

	server := setupHTTPServer(cfg, businessUnitService, mongoClient, log)

	run(cfg, server, log)
}

func initLogger() *logger.Logger {
	return logger.New(logger.Config{
		Level:     logger.INFO,
		Format:    logger.JSON,
		AddSource: true,
		Service:   "business-units",
	})
}

func connectMongoDB(cfg *config.Config, log *logger.Logger) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoConnTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB",
			"error", err,
			"uri", cfg.MongoURI,
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

func setupHTTPServer(cfg *config.Config, businessUnitService service.BusinessUnitService, mongoClient *mongo.Client, log *logger.Logger) *http.Server {
	healthRouter := httprouter.New()
	healthHandler := handler.NewHealthHandler(mongoClient, log)
	healthHandler.RegisterRoutes(healthRouter)

	var healthHTTPHandler http.Handler = healthRouter
	healthHTTPHandler = middleware.RequestLogging(log)(healthHTTPHandler)
	healthHTTPHandler = middleware.Recovery(log)(healthHTTPHandler)
	log.Info("Health endpoints configured with minimal middleware (Recovery + Logging only)")

	businessRouter := httprouter.New()
	businessUnitHandler := handler.NewBusinessUnitHandler(businessUnitService)
	businessUnitHandler.RegisterRoutes(businessRouter)

	idempotencyStore := middleware.NewInMemoryIdempotencyStore(cfg.IdempotencyTTL)
	phoneRateLimiter := middleware.NewPhoneRateLimiter(
		cfg.RateLimitRequests,
		cfg.RateLimitWindow,
		middleware.DefaultPhoneExtractor,
		log,
	)

	// Middleware order: Recovery → Logging → MaxSize → ContentType → Signature → RateLimit → Timeout → Idempotency → Router
	var businessHTTPHandler http.Handler = businessRouter
	businessHTTPHandler = middleware.Idempotency(idempotencyStore, "Idempotency-Key")(businessHTTPHandler)
	businessHTTPHandler = middleware.RequestTimeout(cfg.RequestTimeout)(businessHTTPHandler)
	businessHTTPHandler = middleware.PhoneRateLimit(phoneRateLimiter)(businessHTTPHandler)

	if cfg.WhatsAppAppSecret != "" {
		businessHTTPHandler = middleware.WhatsAppSignatureVerification(cfg.WhatsAppAppSecret, log)(businessHTTPHandler)
		log.Info("WhatsApp signature verification enabled")
	}

	businessHTTPHandler = middleware.ContentTypeValidation(log)(businessHTTPHandler)
	businessHTTPHandler = middleware.MaxRequestSize(int64(cfg.MaxRequestSize))(businessHTTPHandler)
	businessHTTPHandler = middleware.RequestLogging(log)(businessHTTPHandler)
	businessHTTPHandler = middleware.Recovery(log)(businessHTTPHandler)
	log.Info("Business endpoints configured with full security middleware stack")

	mux := http.NewServeMux()
	mux.Handle("/health", healthHTTPHandler)
	mux.Handle("/ready", healthHTTPHandler)
	mux.Handle("/", businessHTTPHandler)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	log.Info("HTTP server configured", "port", cfg.Port)
	return server
}

func run(cfg *config.Config, server *http.Server, log *logger.Logger) {
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
		gracefulShutdown(cfg, server, log)
	}
}

func gracefulShutdown(cfg *config.Config, server *http.Server, log *logger.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", "error", err)
		if err := server.Close(); err != nil {
			log.Fatal("Could not stop server gracefully", "error", err)
		}
	}

	log.Info("Server stopped gracefully")
}
