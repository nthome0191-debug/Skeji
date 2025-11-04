package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	MongoURI         string
	MongoConnTimeout time.Duration

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
}

func Load() *Config {
	return &Config{
		MongoURI:         getEnvStr(EnvMongoURI, DefaultMongoURI),
		MongoConnTimeout: getEnvDuration(EnvMongoConnTimeout, DefaultMongoConnTimeout),

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
	}
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
