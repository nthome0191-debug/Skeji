package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a Kafka message with metadata
type Message struct {
	Key       string            // Partition key (e.g., user_phone_number, booking_id)
	Value     []byte            // Message payload (JSON-encoded)
	Headers   map[string]string // Message headers
	Topic     string            // Topic name
	Partition int               // Partition number (set by Kafka)
	Offset    int64             // Message offset (set by Kafka)
	Timestamp time.Time         // Message timestamp
}

// Header keys used across all microservices
const (
	HeaderEventID        = "event-id"
	HeaderEventType      = "event-type"
	HeaderCorrelationID  = "correlation-id"
	HeaderConversationID = "conversation-id"
	HeaderSchemaVersion  = "schema-version"
	HeaderSource         = "source"
	HeaderTimestamp      = "timestamp"
	HeaderRetryCount     = "retry-count"
	HeaderOriginalTopic  = "original-topic"
)

// MessageBuilder provides a fluent interface for building messages
type MessageBuilder struct {
	msg Message
}

// NewMessage creates a new MessageBuilder
func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		msg: Message{
			Headers:   make(map[string]string),
			Timestamp: time.Now(),
		},
	}
}

// WithKey sets the message key (for partition routing)
func (mb *MessageBuilder) WithKey(key string) *MessageBuilder {
	mb.msg.Key = key
	return mb
}

// WithValue sets the message value (will be JSON-encoded)
func (mb *MessageBuilder) WithValue(value interface{}) *MessageBuilder {
	data, err := json.Marshal(value)
	if err != nil {
		// Store error in context, will be handled by producer
		mb.msg.Value = nil
		return mb
	}
	mb.msg.Value = data
	return mb
}

// WithRawValue sets the message value directly (already encoded)
func (mb *MessageBuilder) WithRawValue(value []byte) *MessageBuilder {
	mb.msg.Value = value
	return mb
}

// WithHeader adds a custom header
func (mb *MessageBuilder) WithHeader(key, value string) *MessageBuilder {
	mb.msg.Headers[key] = value
	return mb
}

// WithEventID sets the event ID (generates UUID if not provided)
func (mb *MessageBuilder) WithEventID(eventID string) *MessageBuilder {
	if eventID == "" {
		eventID = uuid.New().String()
	}
	mb.msg.Headers[HeaderEventID] = eventID
	return mb
}

// WithEventType sets the event type
func (mb *MessageBuilder) WithEventType(eventType string) *MessageBuilder {
	mb.msg.Headers[HeaderEventType] = eventType
	return mb
}

// WithCorrelationID sets the correlation ID
func (mb *MessageBuilder) WithCorrelationID(correlationID string) *MessageBuilder {
	mb.msg.Headers[HeaderCorrelationID] = correlationID
	return mb
}

// WithConversationID sets the conversation ID
func (mb *MessageBuilder) WithConversationID(conversationID string) *MessageBuilder {
	mb.msg.Headers[HeaderConversationID] = conversationID
	return mb
}

// WithSchemaVersion sets the schema version
func (mb *MessageBuilder) WithSchemaVersion(version string) *MessageBuilder {
	mb.msg.Headers[HeaderSchemaVersion] = version
	return mb
}

// WithSource sets the source service
func (mb *MessageBuilder) WithSource(source string) *MessageBuilder {
	mb.msg.Headers[HeaderSource] = source
	return mb
}

// Build returns the constructed message
func (mb *MessageBuilder) Build() Message {
	// Ensure event ID exists
	if mb.msg.Headers[HeaderEventID] == "" {
		mb.msg.Headers[HeaderEventID] = uuid.New().String()
	}

	// Ensure timestamp header exists
	if mb.msg.Headers[HeaderTimestamp] == "" {
		mb.msg.Headers[HeaderTimestamp] = mb.msg.Timestamp.Format(time.RFC3339)
	}

	return mb.msg
}

// MessageHandler is the function signature for processing messages
// Return nil for successful processing, error for failure
type MessageHandler func(ctx context.Context, msg Message) error

// DecodeValue decodes the message value into the provided struct
func (m *Message) DecodeValue(v interface{}) error {
	return json.Unmarshal(m.Value, v)
}

// GetHeader retrieves a header value
func (m *Message) GetHeader(key string) (string, bool) {
	value, exists := m.Headers[key]
	return value, exists
}

// GetEventID returns the event ID header
func (m *Message) GetEventID() string {
	return m.Headers[HeaderEventID]
}

// GetCorrelationID returns the correlation ID header
func (m *Message) GetCorrelationID() string {
	return m.Headers[HeaderCorrelationID]
}

// GetConversationID returns the conversation ID header
func (m *Message) GetConversationID() string {
	return m.Headers[HeaderConversationID]
}

// GetEventType returns the event type header
func (m *Message) GetEventType() string {
	return m.Headers[HeaderEventType]
}

// GetRetryCount returns the retry count header as an integer
func (m *Message) GetRetryCount() int {
	if countStr, exists := m.Headers[HeaderRetryCount]; exists {
		var count int
		if err := json.Unmarshal([]byte(countStr), &count); err == nil {
			return count
		}
	}
	return 0
}

// IncrementRetryCount increments the retry count header
func (m *Message) IncrementRetryCount() {
	count := m.GetRetryCount() + 1
	m.Headers[HeaderRetryCount] = string(rune(count + '0'))
}
