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

const (
	// MongoDB configuration
	DefaultMongoConnTimeout = 10 * time.Second

	// Rate limiting configuration
	DefaultRateLimitRequests = 10
	DefaultRateLimitWindow   = 1 * time.Minute

	// Middleware configuration
	DefaultRequestTimeout  = 30 * time.Second
	DefaultIdempotencyTTL  = 24 * time.Hour
	DefaultMaxRequestSize  = 1 * 1024 * 1024 // 1MB

	// HTTP server configuration
	DefaultReadTimeout     = 15 * time.Second
	DefaultWriteTimeout    = 15 * time.Second
	DefaultIdleTimeout     = 60 * time.Second
	DefaultShutdownTimeout = 30 * time.Second
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

	ctx, cancel := context.WithTimeout(context.Background(), DefaultMongoConnTimeout)
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

	idempotencyStore := middleware.NewInMemoryIdempotencyStore(DefaultIdempotencyTTL)
	phoneRateLimiter := middleware.NewPhoneRateLimiter(
		DefaultRateLimitRequests,
		DefaultRateLimitWindow,
		middleware.DefaultPhoneExtractor,
		log,
	)

	// Apply middleware in order from innermost to outermost
	// Execution order will be: Recovery → Logging → MaxSize → ContentType → Signature → RateLimit → Timeout → Idempotency → Router
	var businessHTTPHandler http.Handler = businessRouter
	businessHTTPHandler = middleware.Idempotency(idempotencyStore, "Idempotency-Key")(businessHTTPHandler)
	businessHTTPHandler = middleware.RequestTimeout(DefaultRequestTimeout)(businessHTTPHandler)
	businessHTTPHandler = middleware.PhoneRateLimit(phoneRateLimiter)(businessHTTPHandler)

	whatsappSecret := os.Getenv("WHATSAPP_APP_SECRET")
	if whatsappSecret != "" {
		businessHTTPHandler = middleware.WhatsAppSignatureVerification(whatsappSecret, log)(businessHTTPHandler)
		log.Info("WhatsApp signature verification enabled")
	}

	businessHTTPHandler = middleware.ContentTypeValidation(log)(businessHTTPHandler)
	businessHTTPHandler = middleware.MaxRequestSize(DefaultMaxRequestSize)(businessHTTPHandler)
	businessHTTPHandler = middleware.RequestLogging(log)(businessHTTPHandler)
	businessHTTPHandler = middleware.Recovery(log)(businessHTTPHandler)
	log.Info("Business endpoints configured with full security middleware stack")

	// Combine both routers using http.ServeMux
	mux := http.NewServeMux()
	mux.Handle("/health", healthHTTPHandler)
	mux.Handle("/ready", healthHTTPHandler)
	mux.Handle("/", businessHTTPHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
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
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", "error", err)
		if err := server.Close(); err != nil {
			log.Fatal("Could not stop server gracefully", "error", err)
		}
	}

	log.Info("Server stopped gracefully")
}
