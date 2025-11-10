package config

import "time"

type Weekday string

const (
	Sunday    Weekday = "sunday"
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
)

const (
	DefaultMongoURI          = "mongodb://localhost:27017"
	DefaultMongoDatabaseName = "skeji"
	DefaultMongoConnTimeout  = 10 * time.Second

	DefaultPort     = "8080"
	DefaultLogLevel = "info"

	DefaultRateLimitRequests = 10
	DefaultRateLimitWindow   = 1 * time.Minute

	DefaultRequestTimeout = 30 * time.Second
	DefaultIdempotencyTTL = 24 * time.Hour
	DefaultMaxRequestSize = 1 * 1024 * 1024 // 1MB

	DefaultReadTimeout     = 15 * time.Second
	DefaultWriteTimeout    = 15 * time.Second
	DefaultIdleTimeout     = 60 * time.Second
	DefaultShutdownTimeout = 30 * time.Second

	DefaultDefaultBusinessPriority = 10
	DefaultMinBusinessPriority     = 0
	DefaultMaxBusinessPriority     = 1000

	DefaultDefaultMeetingDurationMin     = 45
	DefaultDefaultBreakDurationMin       = 15
	DefaultDefaultMaxParticipantsPerSlot = 1

	DefaultDefaultStartOfDay = "09:00"
	DefaultDefaultEndOfDay   = "18:00"
)

var (
	DefaultWorkingDaysIsrael = []Weekday{Sunday, Monday, Tuesday, Wednesday, Thursday}
	DefaultWorkingDaysUs     = []Weekday{Monday, Tuesday, Wednesday, Thursday, Friday}
)
