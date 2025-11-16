package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
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

func NewApplication(cfg *config.Config) *Application {
	return &Application{
		cfg: cfg,
	}
}

func (a *Application) SetApp(appHandler contracts.Handler) {
	a.setHealthHandler()
	a.setAppHandler(appHandler)
	a.setAppServer()
}

func (a *Application) setHealthHandler() {
	healthRouter := httprouter.New()
	healthHandler := NewHealthHandler(a.cfg.Client.Mongo, a.cfg.Log)
	healthHandler.RegisterRoutes(healthRouter)

	var healthHTTPHandler http.Handler = healthRouter
	healthHTTPHandler = middleware.Recovery(a.cfg.Log)(healthHTTPHandler)
	a.healthHandler = &healthHTTPHandler
	a.cfg.Log.Info("Health endpoints configured with minimal middleware (Recovery + Logging only)")
}

func (a *Application) setAppHandler(appHandler contracts.Handler) {
	appRouter := httprouter.New()
	appHandler.RegisterRoutes(appRouter)

	a.idempotencyStore = middleware.NewInMemoryIdempotencyStore(a.cfg.IdempotencyTTL)
	a.rateLimiter = middleware.NewPhoneRateLimiter(
		a.cfg.RateLimitRequests,
		a.cfg.RateLimitWindow,
		middleware.DefaultPhoneExtractor,
		a.cfg.Log,
	)

	var appHttpHandler http.Handler = appRouter
	appHttpHandler = middleware.Idempotency(a.idempotencyStore, "Idempotency-Key")(appHttpHandler)
	appHttpHandler = middleware.RequestTimeout(a.cfg.RequestTimeout)(appHttpHandler)
	appHttpHandler = middleware.PhoneRateLimit(a.rateLimiter)(appHttpHandler)
	appHttpHandler = middleware.ContentTypeValidation(a.cfg.Log)(appHttpHandler)
	appHttpHandler = middleware.MaxRequestSize(int64(a.cfg.MaxRequestSize))(appHttpHandler)
	appHttpHandler = middleware.RequestLogging(a.cfg.Log)(appHttpHandler)
	appHttpHandler = middleware.Recovery(a.cfg.Log)(appHttpHandler)
	a.appHttpHandler = &appHttpHandler
	a.cfg.Log.Info("Application endpoints configured with full security middleware stack")
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
	a.cfg.GracefulShutdown()
	a.cfg.Log.Info("Server stopped gracefully")
}
