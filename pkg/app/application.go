package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"skeji/internal/businessunits/handler"
	"skeji/pkg/config"
	"skeji/pkg/contracts"
	"skeji/pkg/middleware"
	"syscall"

	"github.com/julienschmidt/httprouter"
)

type Application struct {
	cfg              *config.Config
	server           *http.Server
	idempotencyStore *middleware.InMemoryIdempotencyStore
	rateLimiter      *middleware.PhoneRateLimiter
	healthHandler    *http.Handler
	appHttpHandler   *http.Handler
}

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) SetApp(cfg *config.Config, appHandler contracts.Handler) {
	a.setHealthHandler(cfg)
	a.setAppHandler(cfg, appHandler)
	a.setAppServer()
}

func (a *Application) setHealthHandler(cfg *config.Config) {
	healthRouter := httprouter.New()
	healthHandler := handler.NewHealthHandler(cfg.Client.Mongo, cfg.Log)
	healthHandler.RegisterRoutes(healthRouter)

	var healthHTTPHandler http.Handler = healthRouter
	healthHTTPHandler = middleware.RequestLogging(cfg.Log)(healthHTTPHandler)
	healthHTTPHandler = middleware.Recovery(cfg.Log)(healthHTTPHandler)
	a.healthHandler = &healthHTTPHandler
	cfg.Log.Info("Health endpoints configured with minimal middleware (Recovery + Logging only)")
}

func (a *Application) setAppHandler(cfg *config.Config, appHandler contracts.Handler) {
	appRouter := httprouter.New()
	appHandler.RegisterRoutes(appRouter)

	a.idempotencyStore = middleware.NewInMemoryIdempotencyStore(cfg.IdempotencyTTL)
	a.rateLimiter = middleware.NewPhoneRateLimiter(
		cfg.RateLimitRequests,
		cfg.RateLimitWindow,
		middleware.DefaultPhoneExtractor,
		cfg.Log,
	)

	var appHttpHandler http.Handler = appRouter
	appHttpHandler = middleware.Idempotency(a.idempotencyStore, "Idempotency-Key")(appHttpHandler)
	appHttpHandler = middleware.RequestTimeout(cfg.RequestTimeout)(appHttpHandler)
	appHttpHandler = middleware.PhoneRateLimit(a.rateLimiter)(appHttpHandler)
	if cfg.WhatsAppAppSecret != "" {
		appHttpHandler = middleware.WhatsAppSignatureVerification(cfg.WhatsAppAppSecret, cfg.Log)(appHttpHandler)
		cfg.Log.Info("WhatsApp signature verification enabled")
	}
	appHttpHandler = middleware.ContentTypeValidation(cfg.Log)(appHttpHandler)
	appHttpHandler = middleware.MaxRequestSize(int64(cfg.MaxRequestSize))(appHttpHandler)
	appHttpHandler = middleware.RequestLogging(cfg.Log)(appHttpHandler)
	appHttpHandler = middleware.Recovery(cfg.Log)(appHttpHandler)
	a.appHttpHandler = &appHttpHandler
	cfg.Log.Info("Application endpoints configured with full security middleware stack")
}

func (a *Application) setAppServer() {
	mux := http.NewServeMux()
	mux.Handle("/health", *a.healthHandler)
	mux.Handle("/ready", *a.healthHandler)
	mux.Handle("/", *a.appHttpHandler)

	a.server = &http.Server{
		Addr:         ":" + a.cfg.Port,
		Handler:      mux,
		ReadTimeout:  a.cfg.ReadTimeout,
		WriteTimeout: a.cfg.WriteTimeout,
		IdleTimeout:  a.cfg.IdleTimeout,
	}

	a.cfg.Log.Info("HTTP server configured", "port", a.cfg.Port)
}

func (a *Application) Run() {
	serverErrors := make(chan error, 1)

	go func() {
		a.cfg.Log.Info("Starting HTTP server", "address", a.server.Addr)
		serverErrors <- a.server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		a.cfg.Log.Fatal("HTTP server failed", "error", err)

	case sig := <-shutdown:
		a.cfg.Log.Info("Shutdown signal received", "signal", sig)
		a.gracefulShutdown()
	}
}

func (a *Application) gracefulShutdown() {
	a.cfg.Log.Info("Starting graceful shutdown...")

	a.cfg.Log.Info("Stopping background workers...")
	a.idempotencyStore.Stop()
	a.rateLimiter.Stop()
	a.cfg.Log.Info("Background workers stopped")

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.cfg.Log.Error("Server shutdown failed", "error", err)
		if err := a.server.Close(); err != nil {
			a.cfg.Log.Fatal("Could not stop server gracefully", "error", err)
		}
	}

	a.cfg.Log.Info("Server stopped gracefully")
}
