package kafka_middleware

import (
	"context"
	"log"
	"time"

	"skeji/pkg/kafka"
)

// LoggingProducerMiddleware logs message publishing operations
func LoggingProducerMiddleware() kafka.ProducerMiddleware {
	return func(ctx context.Context, msg kafka.Message, next func(ctx context.Context, msg kafka.Message) error) error {
		start := time.Now()

		log.Printf(
			"[KAFKA PRODUCER] Publishing message | topic=%s key=%s event_id=%s correlation_id=%s",
			msg.Topic,
			msg.Key,
			msg.GetEventID(),
			msg.GetCorrelationID(),
		)

		err := next(ctx, msg)

		duration := time.Since(start)

		if err != nil {
			log.Printf(
				"[KAFKA PRODUCER] Failed to publish message | topic=%s key=%s event_id=%s correlation_id=%s duration=%s error=%v",
				msg.Topic,
				msg.Key,
				msg.GetEventID(),
				msg.GetCorrelationID(),
				duration,
				err,
			)
		} else {
			log.Printf(
				"[KAFKA PRODUCER] Successfully published message | topic=%s key=%s event_id=%s correlation_id=%s duration=%s",
				msg.Topic,
				msg.Key,
				msg.GetEventID(),
				msg.GetCorrelationID(),
				duration,
			)
		}

		return err
	}
}

// LoggingConsumerMiddleware logs message consumption operations
func LoggingConsumerMiddleware() kafka.ConsumerMiddleware {
	return func(ctx context.Context, msg kafka.Message, next kafka.MessageHandler) error {
		start := time.Now()

		log.Printf(
			"[KAFKA CONSUMER] Processing message | topic=%s partition=%d offset=%d key=%s event_id=%s correlation_id=%s",
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.GetEventID(),
			msg.GetCorrelationID(),
		)

		err := next(ctx, msg)

		duration := time.Since(start)

		if err != nil {
			log.Printf(
				"[KAFKA CONSUMER] Failed to process message | topic=%s partition=%d offset=%d key=%s event_id=%s correlation_id=%s duration=%s error=%v",
				msg.Topic,
				msg.Partition,
				msg.Offset,
				msg.Key,
				msg.GetEventID(),
				msg.GetCorrelationID(),
				duration,
				err,
			)
		} else {
			log.Printf(
				"[KAFKA CONSUMER] Successfully processed message | topic=%s partition=%d offset=%d key=%s event_id=%s correlation_id=%s duration=%s",
				msg.Topic,
				msg.Partition,
				msg.Offset,
				msg.Key,
				msg.GetEventID(),
				msg.GetCorrelationID(),
				duration,
			)
		}

		return err
	}
}
