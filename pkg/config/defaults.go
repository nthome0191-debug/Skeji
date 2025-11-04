package config

import "time"

const (
	DefaultMongoURI          = "mongodb://localhost:27017"
	DefaultMongoDatabaseName = "skeji"
	DefaultMongoConnTimeout  = 10 * time.Second

	DefaultPort = "8080"

	DefaultRateLimitRequests = 10
	DefaultRateLimitWindow   = 1 * time.Minute

	DefaultRequestTimeout = 30 * time.Second
	DefaultIdempotencyTTL = 24 * time.Hour
	DefaultMaxRequestSize = 1 * 1024 * 1024 // 1MB

	DefaultReadTimeout     = 15 * time.Second
	DefaultWriteTimeout    = 15 * time.Second
	DefaultIdleTimeout     = 60 * time.Second
	DefaultShutdownTimeout = 30 * time.Second
)
