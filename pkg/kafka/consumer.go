package kafka

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	kafka_config "skeji/pkg/kafka/config"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader     *kafka.Reader
	dlqWriter  *kafka.Writer
	topic      string
	groupID    string
	dlqTopic   string
	maxRetries int
	handler    MessageHandler
	middleware []ConsumerMiddleware
	closed     bool
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

type ConsumerMiddleware func(ctx context.Context, msg Message, next MessageHandler) error

func NewConsumer(cfg *kafka_config.Config, topic string, groupID string, dlqTopic string, handler MessageHandler) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}

	if topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	if groupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}

	if handler == nil {
		return nil, fmt.Errorf("message handler cannot be nil")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		Topic:             topic,
		GroupID:           groupID,
		MinBytes:          cfg.ConsumerMinBytes,
		MaxBytes:          cfg.ConsumerMaxBytes,
		MaxWait:           cfg.ConsumerMaxWait,
		CommitInterval:    cfg.ConsumerCommitInterval,
		HeartbeatInterval: cfg.ConsumerHeartbeatInterval,
		SessionTimeout:    cfg.ConsumerSessionTimeout,
		RebalanceTimeout:  cfg.ConsumerRebalanceTimeout,
		StartOffset:       cfg.ConsumerStartOffset,
		Logger:            kafka.LoggerFunc(func(msg string, args ...any) {}), // Silence default logger
		ErrorLogger:       kafka.LoggerFunc(log.Printf),
	})

	consumer := &Consumer{
		reader:     reader,
		topic:      topic,
		groupID:    groupID,
		dlqTopic:   dlqTopic,
		maxRetries: cfg.ConsumerMaxRetries,
		handler:    handler,
		middleware: make([]ConsumerMiddleware, 0),
	}

	if dlqTopic != "" {
		dlqWriter := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        dlqTopic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			Compression:  kafka.Snappy,
			MaxAttempts:  3,
			Logger:       kafka.LoggerFunc(func(msg string, args ...any) {}),
			ErrorLogger:  kafka.LoggerFunc(log.Printf),
		}
		consumer.dlqWriter = dlqWriter
	}

	return consumer, nil
}

func (c *Consumer) Use(middleware ConsumerMiddleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middleware = append(c.middleware, middleware)
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConsumerClosed
	}
	c.mu.RUnlock()

	c.wg.Add(1)
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Fetch message
			kafkaMsg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				// Log error and continue
				log.Printf("kafka consumer error fetching message: %v", err)
				time.Sleep(1 * time.Second) // Backoff
				continue
			}

			// Convert to internal message type
			msg := c.convertMessage(kafkaMsg)

			// Process message with retry logic
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("kafka consumer error processing message: %v", err)
				// Error already handled (retry/DLQ), continue to next message
			}

			// Commit offset after successful processing
			if err := c.reader.CommitMessages(ctx, kafkaMsg); err != nil {
				log.Printf("kafka consumer error committing offset: %v", err)
				// Don't return, just log and continue
			}
		}
	}
}

// processMessage processes a message with retry logic
func (c *Consumer) processMessage(ctx context.Context, msg Message) error {
	retries := msg.GetRetryCount()

	// Build middleware chain
	handler := c.handler
	for i := len(c.middleware) - 1; i >= 0; i-- {
		middleware := c.middleware[i]
		next := handler
		handler = func(ctx context.Context, m Message) error {
			return middleware(ctx, m, next)
		}
	}

	// Execute handler
	err := handler(ctx, msg)
	if err == nil {
		return nil
	}

	// Check if we should retry
	if ShouldRetry(err, retries, c.maxRetries) {
		msg.IncrementRetryCount()
		// In a real implementation, you might want to republish to the same topic
		// or a retry topic with exponential backoff
		log.Printf("retrying message (attempt %d/%d): %v", retries+1, c.maxRetries, err)
		return c.processMessage(ctx, msg)
	}

	// Max retries exceeded or permanent error, send to DLQ
	if c.dlqWriter != nil {
		if dlqErr := c.sendToDLQ(ctx, msg, err); dlqErr != nil {
			log.Printf("failed to send message to DLQ: %v (original error: %v)", dlqErr, err)
		} else {
			log.Printf("message sent to DLQ after %d retries: %v", retries, err)
		}
	}

	return err
}

// sendToDLQ sends a failed message to the dead letter queue
func (c *Consumer) sendToDLQ(ctx context.Context, msg Message, originalErr error) error {
	if c.dlqWriter == nil {
		return nil
	}

	// Add DLQ metadata to headers
	msg.Headers[HeaderOriginalTopic] = c.topic
	msg.Headers["dlq-error"] = originalErr.Error()
	msg.Headers["dlq-timestamp"] = time.Now().Format(time.RFC3339)
	msg.Headers["dlq-consumer-group"] = c.groupID

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

	return c.dlqWriter.WriteMessages(ctx, kafkaMsg)
}

// convertMessage converts a kafka-go message to internal Message type
func (c *Consumer) convertMessage(kafkaMsg kafka.Message) Message {
	msg := Message{
		Key:       string(kafkaMsg.Key),
		Value:     kafkaMsg.Value,
		Headers:   make(map[string]string),
		Topic:     kafkaMsg.Topic,
		Partition: kafkaMsg.Partition,
		Offset:    kafkaMsg.Offset,
		Timestamp: kafkaMsg.Time,
	}

	// Convert headers
	for _, header := range kafkaMsg.Headers {
		msg.Headers[header.Key] = string(header.Value)
	}

	return msg
}

// Close closes the consumer and releases resources
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Wait for ongoing processing to complete
	c.wg.Wait()

	var err error
	if c.reader != nil {
		err = c.reader.Close()
	}

	if c.dlqWriter != nil {
		dlqErr := c.dlqWriter.Close()
		if err == nil {
			err = dlqErr
		}
	}

	return err
}

// Stats returns consumer statistics
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// Lag returns the current consumer lag
func (c *Consumer) Lag() (int64, error) {
	stats := c.reader.Stats()
	return stats.Lag, nil
}
