package kafka_config

import "time"

const (
	// Default Kafka broker
	DefaultKafkaBrokers = "localhost:9092"

	// Producer defaults
	DefaultProducerMaxAttempts  = 3
	DefaultProducerBatchTimeout = 10 * time.Millisecond
	DefaultProducerRequireAcks  = -1 // Require all replicas
	DefaultProducerCompression  = "snappy"
	DefaultProducerAsync        = false

	// Consumer defaults
	DefaultConsumerStartOffset       = -1 // Newest messages
	DefaultConsumerMinBytes          = 1
	DefaultConsumerMaxBytes          = 10 * 1024 * 1024 // 10MB
	DefaultConsumerMaxWait           = 500 * time.Millisecond
	DefaultConsumerCommitInterval    = 1 * time.Second
	DefaultConsumerHeartbeatInterval = 3 * time.Second
	DefaultConsumerSessionTimeout    = 10 * time.Second
	DefaultConsumerRebalanceTimeout  = 60 * time.Second
	DefaultConsumerMaxRetries        = 3

	// Middleware defaults
	DefaultEnableMiddleware = true
)
