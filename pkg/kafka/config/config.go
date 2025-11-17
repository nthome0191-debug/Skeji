package kafka_config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all Kafka configuration
type Config struct {
	// Broker configuration
	Brokers []string

	// Producer configuration
	ProducerMaxAttempts  int
	ProducerBatchTimeout time.Duration
	ProducerRequireAcks  int    // -1 = all, 0 = none, 1 = leader only
	ProducerCompression  string // "none", "gzip", "snappy", "lz4", "zstd"
	ProducerAsync        bool

	// Consumer configuration
	ConsumerStartOffset       int64 // -1 = newest, -2 = oldest
	ConsumerMinBytes          int
	ConsumerMaxBytes          int
	ConsumerMaxWait           time.Duration
	ConsumerCommitInterval    time.Duration
	ConsumerHeartbeatInterval time.Duration
	ConsumerSessionTimeout    time.Duration
	ConsumerRebalanceTimeout  time.Duration
	ConsumerMaxRetries        int

	// Middleware configuration
	EnableMiddleware bool
}

// Load creates a Kafka config from environment variables
func Load() *Config {
	brokersStr := getEnvStr(EnvKafkaBrokers, DefaultKafkaBrokers)
	brokers := strings.Split(brokersStr, ",")
	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}

	cfg := &Config{
		Brokers: brokers,

		ProducerMaxAttempts:  getEnvInt(EnvKafkaProducerMaxAttempts, DefaultProducerMaxAttempts),
		ProducerBatchTimeout: getEnvDuration(EnvKafkaProducerBatchTimeout, DefaultProducerBatchTimeout),
		ProducerRequireAcks:  getEnvInt(EnvKafkaProducerRequireAcks, DefaultProducerRequireAcks),
		ProducerCompression:  getEnvStr(EnvKafkaProducerCompression, DefaultProducerCompression),
		ProducerAsync:        getEnvBool(EnvKafkaProducerAsync, DefaultProducerAsync),

		ConsumerStartOffset:       getEnvInt64(EnvKafkaConsumerStartOffset, DefaultConsumerStartOffset),
		ConsumerMinBytes:          getEnvInt(EnvKafkaConsumerMinBytes, DefaultConsumerMinBytes),
		ConsumerMaxBytes:          getEnvInt(EnvKafkaConsumerMaxBytes, DefaultConsumerMaxBytes),
		ConsumerMaxWait:           getEnvDuration(EnvKafkaConsumerMaxWait, DefaultConsumerMaxWait),
		ConsumerCommitInterval:    getEnvDuration(EnvKafkaConsumerCommitInterval, DefaultConsumerCommitInterval),
		ConsumerHeartbeatInterval: getEnvDuration(EnvKafkaConsumerHeartbeatInterval, DefaultConsumerHeartbeatInterval),
		ConsumerSessionTimeout:    getEnvDuration(EnvKafkaConsumerSessionTimeout, DefaultConsumerSessionTimeout),
		ConsumerRebalanceTimeout:  getEnvDuration(EnvKafkaConsumerRebalanceTimeout, DefaultConsumerRebalanceTimeout),
		ConsumerMaxRetries:        getEnvInt(EnvKafkaConsumerMaxRetries, DefaultConsumerMaxRetries),

		EnableMiddleware: getEnvBool(EnvKafkaEnableMiddleware, DefaultEnableMiddleware),
	}

	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Kafka configuration validation failed: %v", err))
	}

	return cfg
}

// Validate validates the Kafka configuration
func (cfg *Config) Validate() error {
	var errors []string

	if len(cfg.Brokers) == 0 {
		errors = append(errors, "At least one Kafka broker is required")
	}

	for i, broker := range cfg.Brokers {
		if broker == "" {
			errors = append(errors, fmt.Sprintf("Broker %d cannot be empty", i))
		}
	}

	if cfg.ProducerMaxAttempts <= 0 {
		errors = append(errors, fmt.Sprintf("ProducerMaxAttempts must be positive, got: %d", cfg.ProducerMaxAttempts))
	}

	if cfg.ProducerBatchTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("ProducerBatchTimeout must be positive, got: %s", cfg.ProducerBatchTimeout))
	}

	validCompressions := map[string]bool{
		"none": true, "gzip": true, "snappy": true, "lz4": true, "zstd": true,
	}
	if !validCompressions[cfg.ProducerCompression] {
		errors = append(errors, fmt.Sprintf("ProducerCompression must be one of [none, gzip, snappy, lz4, zstd], got: %s", cfg.ProducerCompression))
	}

	validAcks := map[int]bool{-1: true, 0: true, 1: true}
	if !validAcks[cfg.ProducerRequireAcks] {
		errors = append(errors, fmt.Sprintf("ProducerRequireAcks must be -1, 0, or 1, got: %d", cfg.ProducerRequireAcks))
	}

	if cfg.ConsumerStartOffset != -1 && cfg.ConsumerStartOffset != -2 && cfg.ConsumerStartOffset < 0 {
		errors = append(errors, fmt.Sprintf("ConsumerStartOffset must be -1 (newest), -2 (oldest), or >= 0, got: %d", cfg.ConsumerStartOffset))
	}

	if cfg.ConsumerMinBytes <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerMinBytes must be positive, got: %d", cfg.ConsumerMinBytes))
	}

	if cfg.ConsumerMaxBytes <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerMaxBytes must be positive, got: %d", cfg.ConsumerMaxBytes))
	}

	if cfg.ConsumerMaxWait <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerMaxWait must be positive, got: %s", cfg.ConsumerMaxWait))
	}

	if cfg.ConsumerCommitInterval <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerCommitInterval must be positive, got: %s", cfg.ConsumerCommitInterval))
	}

	if cfg.ConsumerHeartbeatInterval <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerHeartbeatInterval must be positive, got: %s", cfg.ConsumerHeartbeatInterval))
	}

	if cfg.ConsumerSessionTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerSessionTimeout must be positive, got: %s", cfg.ConsumerSessionTimeout))
	}

	if cfg.ConsumerRebalanceTimeout <= 0 {
		errors = append(errors, fmt.Sprintf("ConsumerRebalanceTimeout must be positive, got: %s", cfg.ConsumerRebalanceTimeout))
	}

	if cfg.ConsumerMaxRetries < 0 {
		errors = append(errors, fmt.Sprintf("ConsumerMaxRetries cannot be negative, got: %d", cfg.ConsumerMaxRetries))
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

// LogConfiguration logs the Kafka configuration (requires logger)
func (cfg *Config) LogConfiguration(logFunc func(msg string, keysAndValues ...interface{})) {
	if logFunc == nil {
		return
	}

	logFunc("Kafka configuration loaded successfully",
		"brokers", cfg.Brokers,
		"producer_max_attempts", cfg.ProducerMaxAttempts,
		"producer_batch_timeout", cfg.ProducerBatchTimeout,
		"producer_require_acks", cfg.ProducerRequireAcks,
		"producer_compression", cfg.ProducerCompression,
		"producer_async", cfg.ProducerAsync,
		"consumer_start_offset", cfg.ConsumerStartOffset,
		"consumer_min_bytes", cfg.ConsumerMinBytes,
		"consumer_max_bytes", cfg.ConsumerMaxBytes,
		"consumer_max_wait", cfg.ConsumerMaxWait,
		"consumer_commit_interval", cfg.ConsumerCommitInterval,
		"consumer_heartbeat_interval", cfg.ConsumerHeartbeatInterval,
		"consumer_session_timeout", cfg.ConsumerSessionTimeout,
		"consumer_rebalance_timeout", cfg.ConsumerRebalanceTimeout,
		"consumer_max_retries", cfg.ConsumerMaxRetries,
		"enable_middleware", cfg.EnableMiddleware,
	)
}

// Helper functions (private)

func getEnvStr(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if int64Value, err := strconv.ParseInt(value, 10, 64); err == nil {
			return int64Value
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
