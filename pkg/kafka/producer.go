package kafka

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	kafka_config "skeji/pkg/kafka/config"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

// Producer wraps kafka-go writer with additional functionality
type Producer struct {
	writer     *kafka.Writer
	dlqWriter  *kafka.Writer
	topic      string
	dlqTopic   string
	middleware []ProducerMiddleware
	closed     bool
	mu         sync.RWMutex
}

// ProducerMiddleware allows intercepting publish operations
type ProducerMiddleware func(ctx context.Context, msg Message, next func(ctx context.Context, msg Message) error) error

// NewProducer creates a new Kafka producer
func NewProducer(cfg *kafka_config.Config, topic string, dlqTopic string) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}

	if topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	// Map compression type
	var compression compress.Compression
	switch cfg.ProducerCompression {
	case "gzip":
		compression = compress.Gzip
	case "snappy":
		compression = compress.Snappy
	case "lz4":
		compression = compress.Lz4
	case "zstd":
		compression = compress.Zstd
	default:
		compression = compress.Snappy // Default to snappy
	}

	// Map RequireAcks
	var requiredAcks kafka.RequiredAcks
	switch cfg.ProducerRequireAcks {
	case -1:
		requiredAcks = kafka.RequireAll
	case 0:
		requiredAcks = kafka.RequireNone
	case 1:
		requiredAcks = kafka.RequireOne
	default:
		requiredAcks = kafka.RequireAll
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Hash by key for ordering
		RequiredAcks: requiredAcks,
		Compression:  compression,
		MaxAttempts:  cfg.ProducerMaxAttempts,
		BatchTimeout: cfg.ProducerBatchTimeout,
		Async:        cfg.ProducerAsync,
		Logger:       kafka.LoggerFunc(func(msg string, args ...any) {}), // Silence default logger
		ErrorLogger:  kafka.LoggerFunc(log.Printf),
	}

	producer := &Producer{
		writer:     writer,
		topic:      topic,
		dlqTopic:   dlqTopic,
		middleware: make([]ProducerMiddleware, 0),
	}

	// Create DLQ writer if configured
	if dlqTopic != "" {
		dlqWriter := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        dlqTopic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll, // DLQ should be reliable
			Compression:  compression,
			MaxAttempts:  3,
			Logger:       kafka.LoggerFunc(func(msg string, args ...any) {}),
			ErrorLogger:  kafka.LoggerFunc(log.Printf),
		}
		producer.dlqWriter = dlqWriter
	}

	return producer, nil
}

// Use adds middleware to the producer
func (p *Producer) Use(middleware ProducerMiddleware) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.middleware = append(p.middleware, middleware)
}

// Publish publishes a message to Kafka
func (p *Producer) Publish(ctx context.Context, msg Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrProducerClosed
	}
	p.mu.RUnlock()

	// Validate message
	if msg.Key == "" {
		return ErrEmptyKey
	}
	if len(msg.Value) == 0 {
		return ErrEmptyValue
	}

	// Execute middleware chain
	handler := p.publishInternal
	for i := len(p.middleware) - 1; i >= 0; i-- {
		middleware := p.middleware[i]
		next := handler
		handler = func(ctx context.Context, m Message) error {
			return middleware(ctx, m, next)
		}
	}

	return handler(ctx, msg)
}

// publishInternal performs the actual publish operation
func (p *Producer) publishInternal(ctx context.Context, msg Message) error {
	// Convert to kafka-go message
	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Key),
		Value: msg.Value,
		Time:  msg.Timestamp,
	}

	// Add headers
	for k, v := range msg.Headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Write message
	err := p.writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		// Send to DLQ if configured
		if p.dlqWriter != nil {
			dlqErr := p.sendToDLQ(ctx, msg, err)
			if dlqErr != nil {
				return fmt.Errorf("failed to send to DLQ: %v (original error: %v)", dlqErr, err)
			}
		}
		return err
	}

	return nil
}

// PublishBatch publishes multiple messages in a batch
func (p *Producer) PublishBatch(ctx context.Context, messages []Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrProducerClosed
	}
	p.mu.RUnlock()

	kafkaMessages := make([]kafka.Message, 0, len(messages))

	for _, msg := range messages {
		if msg.Key == "" || len(msg.Value) == 0 {
			continue // Skip invalid messages
		}

		kafkaMsg := kafka.Message{
			Key:   []byte(msg.Key),
			Value: msg.Value,
			Time:  msg.Timestamp,
		}

		for k, v := range msg.Headers {
			kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}

		kafkaMessages = append(kafkaMessages, kafkaMsg)
	}

	if len(kafkaMessages) == 0 {
		return ErrInvalidMessage
	}

	return p.writer.WriteMessages(ctx, kafkaMessages...)
}

// sendToDLQ sends a failed message to the dead letter queue
func (p *Producer) sendToDLQ(ctx context.Context, msg Message, originalErr error) error {
	if p.dlqWriter == nil {
		return nil
	}

	// Add DLQ metadata to headers
	msg.Headers[HeaderOriginalTopic] = p.topic
	msg.Headers["dlq-error"] = originalErr.Error()
	msg.Headers["dlq-timestamp"] = time.Now().Format(time.RFC3339)

	kafkaMsg := kafka.Message{
		Key:   []byte(msg.Key),
		Value: msg.Value,
		Time:  time.Now(),
	}

	for k, v := range msg.Headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	return p.dlqWriter.WriteMessages(ctx, kafkaMsg)
}

// Close closes the producer and releases resources
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	var err error
	if p.writer != nil {
		err = p.writer.Close()
	}

	if p.dlqWriter != nil {
		dlqErr := p.dlqWriter.Close()
		if err == nil {
			err = dlqErr
		}
	}

	return err
}

// Stats returns producer statistics
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
