package kafka_middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"skeji/pkg/kafka"
)

// Metrics holds Kafka operation metrics
type Metrics struct {
	// Producer metrics
	MessagesPublished       int64
	MessagesPublishedFailed int64
	PublishDurationTotal    int64 // Nanoseconds

	// Consumer metrics
	MessagesConsumed       int64
	MessagesConsumedFailed int64
	ConsumeDurationTotal   int64 // Nanoseconds

	mu sync.RWMutex
}

// Global metrics instance
var globalMetrics = &Metrics{}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return globalMetrics
}

// Reset resets all metrics (useful for testing)
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.MessagesPublished, 0)
	atomic.StoreInt64(&m.MessagesPublishedFailed, 0)
	atomic.StoreInt64(&m.PublishDurationTotal, 0)
	atomic.StoreInt64(&m.MessagesConsumed, 0)
	atomic.StoreInt64(&m.MessagesConsumedFailed, 0)
	atomic.StoreInt64(&m.ConsumeDurationTotal, 0)
}

// GetPublishRate returns messages published per second
func (m *Metrics) GetPublishRate(duration time.Duration) float64 {
	published := atomic.LoadInt64(&m.MessagesPublished)
	return float64(published) / duration.Seconds()
}

// GetConsumeRate returns messages consumed per second
func (m *Metrics) GetConsumeRate(duration time.Duration) float64 {
	consumed := atomic.LoadInt64(&m.MessagesConsumed)
	return float64(consumed) / duration.Seconds()
}

// GetAvgPublishDuration returns average publish duration
func (m *Metrics) GetAvgPublishDuration() time.Duration {
	published := atomic.LoadInt64(&m.MessagesPublished)
	if published == 0 {
		return 0
	}
	total := atomic.LoadInt64(&m.PublishDurationTotal)
	return time.Duration(total / published)
}

// GetAvgConsumeDuration returns average consume duration
func (m *Metrics) GetAvgConsumeDuration() time.Duration {
	consumed := atomic.LoadInt64(&m.MessagesConsumed)
	if consumed == 0 {
		return 0
	}
	total := atomic.LoadInt64(&m.ConsumeDurationTotal)
	return time.Duration(total / consumed)
}

// MetricsProducerMiddleware tracks producer metrics
func MetricsProducerMiddleware() kafka.ProducerMiddleware {
	return func(ctx context.Context, msg kafka.Message, next func(ctx context.Context, msg kafka.Message) error) error {
		start := time.Now()

		err := next(ctx, msg)

		duration := time.Since(start)
		atomic.AddInt64(&globalMetrics.PublishDurationTotal, int64(duration))

		if err != nil {
			atomic.AddInt64(&globalMetrics.MessagesPublishedFailed, 1)
		} else {
			atomic.AddInt64(&globalMetrics.MessagesPublished, 1)
		}

		return err
	}
}

// MetricsConsumerMiddleware tracks consumer metrics
func MetricsConsumerMiddleware() kafka.ConsumerMiddleware {
	return func(ctx context.Context, msg kafka.Message, next kafka.MessageHandler) error {
		start := time.Now()

		err := next(ctx, msg)

		duration := time.Since(start)
		atomic.AddInt64(&globalMetrics.ConsumeDurationTotal, int64(duration))

		if err != nil {
			atomic.AddInt64(&globalMetrics.MessagesConsumedFailed, 1)
		} else {
			atomic.AddInt64(&globalMetrics.MessagesConsumed, 1)
		}

		return err
	}
}

// PrintMetrics prints current metrics to stdout (useful for debugging)
func (m *Metrics) PrintMetrics() {
	published := atomic.LoadInt64(&m.MessagesPublished)
	publishedFailed := atomic.LoadInt64(&m.MessagesPublishedFailed)
	consumed := atomic.LoadInt64(&m.MessagesConsumed)
	consumedFailed := atomic.LoadInt64(&m.MessagesConsumedFailed)

	println("=== Kafka Metrics ===")
	println("Producer:")
	println("  Published:", published)
	println("  Failed:", publishedFailed)
	println("  Avg Duration:", m.GetAvgPublishDuration().String())
	println("Consumer:")
	println("  Consumed:", consumed)
	println("  Failed:", consumedFailed)
	println("  Avg Duration:", m.GetAvgConsumeDuration().String())
	println("====================")
}
