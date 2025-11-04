package config

const (
	EnvMongoURI         = "MONGO_URI"
	EnvMongoConnTimeout = "MONGO_CONN_TIMEOUT"

	EnvPort = "PORT"

	EnvWhatsAppAppSecret = "WHATSAPP_APP_SECRET"

	EnvRateLimitRequests = "RATE_LIMIT_REQUESTS"
	EnvRateLimitWindow   = "RATE_LIMIT_WINDOW"

	EnvRequestTimeout = "REQUEST_TIMEOUT"
	EnvIdempotencyTTL = "IDEMPOTENCY_TTL"
	EnvMaxRequestSize = "MAX_REQUEST_SIZE"

	EnvReadTimeout     = "READ_TIMEOUT"
	EnvWriteTimeout    = "WRITE_TIMEOUT"
	EnvIdleTimeout     = "IDLE_TIMEOUT"
	EnvShutdownTimeout = "SHUTDOWN_TIMEOUT"
)
