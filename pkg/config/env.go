package config

const (
	EnvMongoURI          = "MONGO_URI"
	EnvMongoDatabaseName = "MONGO_DATABASE_NAME"
	EnvMongoConnTimeout  = "MONGO_CONN_TIMEOUT"

	EnvPort     = "PORT"
	EnvLogLevel = "LOG_LEVEL"

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

	EnvBusinessPriority    = "BUSINESS_PRIORITY"
	EnvMinBusinessPriority = "MIN_BUSINESS_PRIORITY"
	EnvMaxBusinessPriority = "MAX_BUSINESS_PRIORITY"

	EnvDefaultMeetingDurationMin    = "DEFAULT_MEETING_DURATION_MIN"
	EnvDefaultBreakDurationMin      = "DEFAULT_BREAK_DURATION_MIN"
	EnvDefaultMaxParticipantsInSlot = "DEFAULT_MAX_PARTICIPANTS_IN_SLOT"
	EnvDefaultStartOfDay            = "DEFAULT_START_OF_DAY"
	EnvDefaultEndOfDay              = "DEFAULT_END_OF_DAY"

	EnvBusinessUnitBaseUrl = "BUSINESS_UNIT_BASE_URL"
	EnvScheduleBaseUrl     = "SCHEDULE_BASE_URL"
	EnvBookingBaseUrl      = "BOOKING_BASE_URL"
)
