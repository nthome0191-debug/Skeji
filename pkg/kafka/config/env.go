package kafka_config

const (
	// Kafka broker configuration
	EnvKafkaBrokers = "KAFKA_BROKERS"

	// Producer configuration
	EnvKafkaProducerMaxAttempts  = "KAFKA_PRODUCER_MAX_ATTEMPTS"
	EnvKafkaProducerBatchTimeout = "KAFKA_PRODUCER_BATCH_TIMEOUT"
	EnvKafkaProducerRequireAcks  = "KAFKA_PRODUCER_REQUIRE_ACKS"
	EnvKafkaProducerCompression  = "KAFKA_PRODUCER_COMPRESSION"
	EnvKafkaProducerAsync        = "KAFKA_PRODUCER_ASYNC"

	// Consumer configuration
	EnvKafkaConsumerStartOffset       = "KAFKA_CONSUMER_START_OFFSET"
	EnvKafkaConsumerMinBytes          = "KAFKA_CONSUMER_MIN_BYTES"
	EnvKafkaConsumerMaxBytes          = "KAFKA_CONSUMER_MAX_BYTES"
	EnvKafkaConsumerMaxWait           = "KAFKA_CONSUMER_MAX_WAIT"
	EnvKafkaConsumerCommitInterval    = "KAFKA_CONSUMER_COMMIT_INTERVAL"
	EnvKafkaConsumerHeartbeatInterval = "KAFKA_CONSUMER_HEARTBEAT_INTERVAL"
	EnvKafkaConsumerSessionTimeout    = "KAFKA_CONSUMER_SESSION_TIMEOUT"
	EnvKafkaConsumerRebalanceTimeout  = "KAFKA_CONSUMER_REBALANCE_TIMEOUT"
	EnvKafkaConsumerMaxRetries        = "KAFKA_CONSUMER_MAX_RETRIES"

	// Middleware configuration
	EnvKafkaEnableMiddleware = "KAFKA_ENABLE_MIDDLEWARE"
)
