package kafka

// This file provides example code snippets for microservice integration.
// Copy and adapt these examples for your specific services.

import (
	"context"
	"log"
	"os"
	"os/signal"
	kafka_config "skeji/pkg/kafka/config"
	"syscall"
)

// ExampleProducer demonstrates how to create and use a Kafka producer
func ExampleProducer() {
	// 1. Load Kafka config from environment
	cfg := kafka_config.Load()

	// 2. Create producer with topic and DLQ
	producer, err := NewProducer(cfg, "my-topic", "dlq-my-service")
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	// 3. Build and publish a message
	msg := NewMessage().
		WithKey("entity-123").
		WithValue(map[string]interface{}{
			"field1": "value1",
			"field2": 123,
		}).
		WithEventType("entity.created").
		WithSource("my-service").
		WithCorrelationID("corr-abc-123").
		Build()

	// 4. Publish
	ctx := context.Background()
	if err := producer.Publish(ctx, msg); err != nil {
		log.Printf("Failed to publish: %v", err)
	} else {
		log.Println("Message published successfully!")
	}
}

// ExampleConsumer demonstrates how to create and use a Kafka consumer
func ExampleConsumer() {
	// 1. Load Kafka config from environment
	cfg := kafka_config.Load()

	// 2. Define message handler
	handler := func(ctx context.Context, msg Message) error {
		log.Printf("Processing message: key=%s event_id=%s", msg.Key, msg.GetEventID())

		// Decode message value
		var payload map[string]interface{}
		if err := msg.DecodeValue(&payload); err != nil {
			// Permanent error - send to DLQ
			return NewPermanentError("invalid message format", err)
		}

		// Process message
		log.Printf("Payload: %+v", payload)

		// Success
		return nil
	}

	// 3. Create consumer with topic, group ID, DLQ, and handler
	consumer, err := NewConsumer(
		cfg,
		"my-topic",          // Topic to consume from
		"my-consumer-group", // Consumer group ID
		"dlq-my-service",    // DLQ topic
		handler,             // Message handler
	)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	// 4. Start consuming with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down consumer...")
		cancel()
	}()

	log.Println("Consumer started, waiting for messages...")
	if err := consumer.Start(ctx); err != nil && err != context.Canceled {
		log.Printf("Consumer error: %v", err)
	}
}

// ExampleBookingsService shows a complete example for the Bookings Service
func ExampleBookingsService() {
	// Define domain types
	type BookingCommand struct {
		Action string                 `json:"action"`
		Params map[string]interface{} `json:"params"`
	}

	// Load Kafka config
	cfg := kafka_config.Load()

	// Consumer: Receive commands from Pipeline Executor
	handler := func(ctx context.Context, msg Message) error {
		var cmd BookingCommand
		if err := msg.DecodeValue(&cmd); err != nil {
			return NewPermanentError("invalid command format", err)
		}

		correlationID := msg.GetCorrelationID()
		log.Printf("Processing command: %s (correlation_id=%s)", cmd.Action, correlationID)

		// Process command
		var result interface{}
		var err error

		switch cmd.Action {
		case "create_booking":
			result, err = createBooking(ctx, cmd.Params)
		case "cancel_booking":
			result, err = cancelBooking(ctx, cmd.Params)
		default:
			return NewBusinessError("unknown action: "+cmd.Action, nil)
		}

		if err != nil {
			// Send error response back to Pipeline Executor
			sendResponse(ctx, cfg, correlationID, nil, err)
			return NewTransientError("command processing failed", err)
		}

		// Send success response back to Pipeline Executor
		sendResponse(ctx, cfg, correlationID, result, nil)

		// Publish domain event to Notifier
		publishBookingEvent(ctx, cfg, cmd.Action, result)

		return nil
	}

	consumer, _ := NewConsumer(
		cfg,
		"pipeline-executor-to-bookings",
		"bookings-service-consumer-group",
		"dlq-domain-services",
		handler,
	)

	// Start consuming
	log.Println("Bookings service started...")
	consumer.Start(context.Background())
}

// Helper functions for BookingsService example
func createBooking(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Your business logic here
	return map[string]interface{}{
		"booking_id": "booking-12345",
		"status":     "created",
	}, nil
}

func cancelBooking(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Your business logic here
	return map[string]interface{}{
		"booking_id": "booking-12345",
		"status":     "cancelled",
	}, nil
}

func sendResponse(ctx context.Context, cfg *kafka_config.Config, correlationID string, result interface{}, err error) {
	// Send response back to Pipeline Executor
	producer, _ := NewProducer(cfg, "bookings-to-pipeline-executor", "")
	defer producer.Close()

	response := map[string]interface{}{
		"correlation_id": correlationID,
		"result":         result,
		"error":          nil,
	}
	if err != nil {
		response["error"] = err.Error()
	}

	msg := NewMessage().
		WithKey(correlationID).
		WithCorrelationID(correlationID).
		WithValue(response).
		WithSource("bookings-service").
		Build()

	producer.Publish(ctx, msg)
}

func publishBookingEvent(ctx context.Context, cfg *kafka_config.Config, action string, result interface{}) {
	// Publish domain event to Notifier
	producer, _ := NewProducer(cfg, "domain-events-to-notifier", "")
	defer producer.Close()

	msg := NewMessage().
		WithKey("booking-event").
		WithValue(result).
		WithEventType("booking." + action).
		WithSource("bookings-service").
		Build()

	producer.Publish(ctx, msg)
}
