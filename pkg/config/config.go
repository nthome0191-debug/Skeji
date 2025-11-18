package config

import (
	"fmt"
	"os"
	"regexp"
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
	MinBusinessPriority     int
	MaxBusinessPriority     int

	DefaultMeetingDurationMin     int
	DefaultBreakDurationMin       int
	DefaultMaxParticipantsPerSlot int
	DefaultStartOfDay             string
	DefaultEndOfDay               string
	DefaultWorkingDaysIsrael      []string
	DefaultWorkingDaysUs          []string

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
		MinBusinessPriority:     getEnvNum(EnvMinBusinessPriority, DefaultMinBusinessPriority),
		MaxBusinessPriority:     getEnvNum(EnvMaxBusinessPriority, DefaultMaxBusinessPriority),

		DefaultMeetingDurationMin:     getEnvNum(EnvDefaultMeetingDurationMin, DefaultDefaultMeetingDurationMin),
		DefaultBreakDurationMin:       getEnvNum(EnvDefaultBreakDurationMin, DefaultDefaultBreakDurationMin),
		DefaultMaxParticipantsPerSlot: getEnvNum(EnvDefaultMaxParticipantsInSlot, DefaultDefaultMaxParticipantsPerSlot),
		DefaultStartOfDay:             getEnvStr(EnvDefaultStartOfDay, DefaultDefaultStartOfDay),
		DefaultEndOfDay:               getEnvStr(EnvDefaultEndOfDay, DefaultDefaultEndOfDay),
		DefaultWorkingDaysIsrael:      DefaultWorkingDaysIsrael,
		DefaultWorkingDaysUs:          DefaultWorkingDaysUs,

		Log: logger.New(logger.Config{
			Level:     getEnvStr(EnvLogLevel, DefaultLogLevel),
			Format:    logger.JSON,
			AddSource: true,
			Service:   serviceName,
		}),
		Client: client.NewClient(),
	}

	err := cfg.Validate()
	if err != nil {
		cfg.Log.Fatal(err.Error())
	}
	cfg.LogConfiguration()
	return cfg
}

func (cfg *Config) SetMongo() {
	cfg.Client.SetMongo(cfg.Log, cfg.MongoURI, cfg.MongoConnTimeout)
}

func (cfg *Config) Validate() error {
	var errors []string

	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		errors = append(errors, fmt.Sprintf("Port must be between 1 and 65535, got: %s", cfg.Port))
	}

	timeRegex := regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	if !timeRegex.MatchString(cfg.DefaultStartOfDay) {
		errors = append(errors, fmt.Sprintf("DefaultStartOfDay must be in HH:MM format (00:00-23:59), got: %s", cfg.DefaultStartOfDay))
	}
	if !timeRegex.MatchString(cfg.DefaultEndOfDay) {
		errors = append(errors, fmt.Sprintf("DefaultEndOfDay must be in HH:MM format (00:00-23:59), got: %s", cfg.DefaultEndOfDay))
	}

	if cfg.MongoURI == "" {
		errors = append(errors, "MongoURI cannot be empty")
	} else if len(cfg.MongoURI) < 10 || !regexp.MustCompile(`^mongodb(\+srv)?://`).MatchString(cfg.MongoURI) {
		errors = append(errors, fmt.Sprintf("MongoURI must start with 'mongodb://' or 'mongodb+srv://', got: %s", cfg.MongoURI))
	}

	if cfg.MongoDatabaseName == "" {
		errors = append(errors, "MongoDatabaseName cannot be empty")
	}

	if cfg.MongoConnTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("MongoConnTimeout must be positive, got: %s", cfg.MongoConnTimeout))
	}
	if cfg.RateLimitWindow <= 0 {
		errors = append(errors, fmt.Sprintf("RateLimitWindow must be positive, got: %s", cfg.RateLimitWindow))
	}
	if cfg.RequestTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("RequestTimeout must be positive, got: %s", cfg.RequestTimeout))
	}
	if cfg.IdempotencyTTL <= 0 {
		errors = append(errors, fmt.Sprintf("IdempotencyTTL must be positive, got: %s", cfg.IdempotencyTTL))
	}
	if cfg.ReadTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("ReadTimeout must be positive, got: %s", cfg.ReadTimeout))
	}
	if cfg.WriteTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("WriteTimeout must be positive, got: %s", cfg.WriteTimeout))
	}
	if cfg.IdleTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("IdleTimeout must be positive, got: %s", cfg.IdleTimeout))
	}
	if cfg.ShutdownTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("ShutdownTimeout must be positive, got: %s", cfg.ShutdownTimeout))
	}

	if cfg.RateLimitRequests <= 0 {
		errors = append(errors, fmt.Sprintf("RateLimitRequests must be positive, got: %d", cfg.RateLimitRequests))
	}
	if cfg.MaxRequestSize <= 0 {
		errors = append(errors, fmt.Sprintf("MaxRequestSize must be positive, got: %d", cfg.MaxRequestSize))
	}

	if cfg.MinBusinessPriority < 0 {
		errors = append(errors, fmt.Sprintf("MinBusinessPriority cannot be negative, got: %d", cfg.MinBusinessPriority))
	}
	if cfg.MaxBusinessPriority < cfg.MinBusinessPriority {
		errors = append(errors, fmt.Sprintf("MaxBusinessPriority (%d) must be >= MinBusinessPriority (%d)", cfg.MaxBusinessPriority, cfg.MinBusinessPriority))
	}
	if cfg.DefaultBusinessPriority < cfg.MinBusinessPriority || cfg.DefaultBusinessPriority > cfg.MaxBusinessPriority {
		errors = append(errors, fmt.Sprintf("DefaultBusinessPriority (%d) must be between MinBusinessPriority (%d) and MaxBusinessPriority (%d)", cfg.DefaultBusinessPriority, cfg.MinBusinessPriority, cfg.MaxBusinessPriority))
	}

	if cfg.DefaultMeetingDurationMin <= 0 {
		errors = append(errors, fmt.Sprintf("DefaultMeetingDurationMin must be positive, got: %d", cfg.DefaultMeetingDurationMin))
	}
	if cfg.DefaultBreakDurationMin < 0 {
		errors = append(errors, fmt.Sprintf("DefaultBreakDurationMin cannot be negative, got: %d", cfg.DefaultBreakDurationMin))
	}
	if cfg.DefaultMaxParticipantsPerSlot <= 0 {
		errors = append(errors, fmt.Sprintf("DefaultMaxParticipantsPerSlot must be positive, got: %d", cfg.DefaultMaxParticipantsPerSlot))
	}

	if len(errors) > 0 {
		errMsg := "Configuration validation failed:\n"
		for i, err := range errors {
			errMsg += fmt.Sprintf("  %d. %s\n", i+1, err)
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func (cfg *Config) LogConfiguration() {
	cfg.Log.Info("Configuration loaded successfully",
		"mongo_uri", redactMongoURI(cfg.MongoURI),
		"mongo_database", cfg.MongoDatabaseName,
		"mongo_conn_timeout", cfg.MongoConnTimeout,
		"port", cfg.Port,
		"whatsapp_secret_set", cfg.WhatsAppAppSecret != "",
		"rate_limit_requests", cfg.RateLimitRequests,
		"rate_limit_window", cfg.RateLimitWindow,
		"request_timeout", cfg.RequestTimeout,
		"idempotency_ttl", cfg.IdempotencyTTL,
		"max_request_size", cfg.MaxRequestSize,
		"read_timeout", cfg.ReadTimeout,
		"write_timeout", cfg.WriteTimeout,
		"idle_timeout", cfg.IdleTimeout,
		"shutdown_timeout", cfg.ShutdownTimeout,
		"default_business_priority", cfg.DefaultBusinessPriority,
		"min_business_priority", cfg.MinBusinessPriority,
		"max_business_priority", cfg.MaxBusinessPriority,
		"default_meeting_duration_min", cfg.DefaultMeetingDurationMin,
		"default_break_duration_min", cfg.DefaultBreakDurationMin,
		"default_max_participants_per_slot", cfg.DefaultMaxParticipantsPerSlot,
		"default_start_of_day", cfg.DefaultStartOfDay,
		"default_end_of_day", cfg.DefaultEndOfDay,
	)
}

func redactMongoURI(uri string) string {
	credentialRegex := regexp.MustCompile(`(mongodb(\+srv)?://)[^:]+:[^@]+@`)
	return credentialRegex.ReplaceAllString(uri, "${1}***:***@")
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
	cfg.Client.GracefulShutdown()
}

func NormalizePaginationLimit(limit int) int {
	if limit <= 0 {
		limit = 10
	} else if limit > DefaultPaginationLimit {
		limit = DefaultPaginationLimit
	}
	return limit
}

func NormalizeOffset(offset int64) int64 {
	return max(0, offset)
}
