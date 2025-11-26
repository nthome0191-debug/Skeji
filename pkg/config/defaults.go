package config

import "time"

const (
	Sunday    string = "sunday"
	Monday    string = "monday"
	Tuesday   string = "tuesday"
	Wednesday string = "wednesday"
	Thursday  string = "thursday"
	Friday    string = "friday"
	Saturday  string = "saturday"

	Pending   string = "pending"
	Confirmed string = "confirmed"
	Cancelled string = "cancelled"
)

const (
	DefaultMongoURI          = "mongodb://localhost:27017"
	DefaultMongoDatabaseName = "skeji"
	DefaultMongoConnTimeout  = 10 * time.Second

	DefaultPort     = "8080"
	DefaultLogLevel = "info"

	DefaultPaginationLimit = 100

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

	DefaultMaxMaintainersPerBusiness     = 10
	MaxCitiesForBusiness                 = 50
	MaxLabelsForBusiness                 = 10
	DefaultMaxBusinessUnitsPerAdminPhone = 10
	DefaultMaxSchedulesPerBusinessUnits  = 10
	DefaultMaxBookingsPerView            = 10

	DefaultDefaultMeetingDurationMin     = 45
	DefaultDefaultBreakDurationMin       = 15
	DefaultDefaultMaxParticipantsPerSlot = 1

	DefaultDefaultStartOfDay = "09:00"
	DefaultDefaultEndOfDay   = "18:00"

	DefaultBusinessUnitBaseUrl = "http://business-units.apps.svc.cluster.local"
	DefaultScheduleBaseUrl     = "http://schedules.apps.svc.cluster.local"
	DefaultBookingBaseUrl      = "http://bookings.apps.svc.cluster.local"
)

var (
	DefaultWorkingDaysIsrael = []string{Sunday, Monday, Tuesday, Wednesday, Thursday}
	DefaultWorkingDaysUs     = []string{Monday, Tuesday, Wednesday, Thursday, Friday}
)
