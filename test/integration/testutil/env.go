package testutil

import (
	"fmt"
	"os"
	"testing"
)

type TestEnv struct {
	MongoURI     string
	DatabaseName string
	ServerURL    string
	ServerPort   string
}

func NewTestEnv() *TestEnv {
	mongoURI := getEnv("TEST_MONGO_URI", DefaultMongoURI)
	dbName := getEnv("TEST_DB_NAME", DefaultDatabaseName)
	serverPort := getEnv("TEST_SERVER_PORT", "8080")
	serverURL := getEnv("TEST_SERVER_URL", fmt.Sprintf("http://localhost:%s", serverPort))

	return &TestEnv{
		MongoURI:     mongoURI,
		DatabaseName: dbName,
		ServerURL:    serverURL,
		ServerPort:   serverPort,
	}
}

func (e *TestEnv) Setup(t *testing.T) (*MongoHelper, *Client) {
	t.Helper()

	mongo := NewMongoHelper(t, e.MongoURI, e.DatabaseName)
	mongo.CleanDatabase(t)

	client := NewClient(e.ServerURL)
	client.WaitForHealthy(t, DefaultHealthCheckTimeout)

	return mongo, client
}

func (e *TestEnv) Cleanup(t *testing.T, mongo *MongoHelper) {
	t.Helper()

	if mongo != nil {
		mongo.CleanDatabase(t)
		mongo.Close(t)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

const (
	DefaultHealthCheckTimeout = 30 * ConnectionTimeout
)
