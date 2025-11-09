package config

import (
	"context"
	"os"
	"skeji/pkg/client"
	"skeji/pkg/logger"
	"strconv"
	"time"
)

type Config struct {
	MongoURI          string
	MongoDatabaseName string
	MongoConnTimeout  time.Duration

	Port string

	WhatsAppAppSecret string

	RateLimitRequests int
	RateLimitWindow   time.Duration

	RequestTimeout time.Duration
	IdempotencyTTL time.Duration
	MaxRequestSize int

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	DefaultBusinessPriority int
	MinBusinessPriotity     int
	MaxBusinessPriority     int

	Log    *logger.Logger
	Client *client.Client
}

func Load(serviceName string) *Config {
	cfg := &Config{
		MongoURI:          getEnvStr(EnvMongoURI, DefaultMongoURI),
		MongoDatabaseName: getEnvStr(EnvMongoDatabaseName, DefaultMongoDatabaseName),
		MongoConnTimeout:  getEnvDuration(EnvMongoConnTimeout, DefaultMongoConnTimeout),

		Port: getEnvStr(EnvPort, DefaultPort),

		WhatsAppAppSecret: getEnvStr(EnvWhatsAppAppSecret, ""),

		RateLimitRequests: getEnvNum(EnvRateLimitRequests, DefaultRateLimitRequests),
		RateLimitWindow:   getEnvDuration(EnvRateLimitWindow, DefaultRateLimitWindow),

		RequestTimeout: getEnvDuration(EnvRequestTimeout, DefaultRequestTimeout),
		IdempotencyTTL: getEnvDuration(EnvIdempotencyTTL, DefaultIdempotencyTTL),
		MaxRequestSize: getEnvNum(EnvMaxRequestSize, DefaultMaxRequestSize),

		ReadTimeout:     getEnvDuration(EnvReadTimeout, DefaultReadTimeout),
		WriteTimeout:    getEnvDuration(EnvWriteTimeout, DefaultWriteTimeout),
		IdleTimeout:     getEnvDuration(EnvIdleTimeout, DefaultIdleTimeout),
		ShutdownTimeout: getEnvDuration(EnvShutdownTimeout, DefaultShutdownTimeout),

		DefaultBusinessPriority: getEnvNum(EnvBusinessPriority, DefaultDefaultBusinessPriority),
		MinBusinessPriotity:     getEnvNum(EnvMinBusinessPriority, DefaultMinBusinessPriority),
		MaxBusinessPriority:     getEnvNum(EnvMaxBusinessPriority, DefaultMaxBusinessPriority),

		Log: logger.New(logger.Config{
			Level:     getEnvStr(EnvLogLevel, DefaultLogLevel),
			Format:    logger.JSON,
			AddSource: true,
			Service:   serviceName,
		}),
		Client: client.NewClient(),
	}
	cfg.Client.SetMongo(cfg.Log, cfg.MongoURI, cfg.MongoConnTimeout)
	return cfg
}

func getEnvStr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvNum(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func (cfg *Config) GracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := cfg.Client.Mongo.Disconnect(ctx)
	if err != nil {
		cfg.Log.Error("Failed to disconnect MongoDB client", "error", err)
	} else {
		cfg.Log.Info("MongoDB client disconnected successfully")
	}
}
