package common

import (
	"context"
	"testing"
	"time"

	"skeji/pkg/client"
	"skeji/pkg/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	DefaultMongoURI         = "mongodb://localhost:27017"
	DefaultDatabaseName     = "skeji_test"
	ConnectionTimeout       = 10 * time.Second
	BusinessUnitsCollection = "Business_units"
)

type MongoHelper struct {
	Client   *mongo.Client
	Database *mongo.Database
	DBName   string
}

func NewMongoHelper(t *testing.T, mongoURI, dbName string) *MongoHelper {
	t.Helper()

	if mongoURI == "" {
		mongoURI = DefaultMongoURI
	}
	if dbName == "" {
		dbName = DefaultDatabaseName
	}

	testLogger := logger.New(logger.Config{
		Service: "test",
		Level:   "debug",
	})

	prodClient := client.NewClient()
	prodClient.SetMongo(testLogger, mongoURI, ConnectionTimeout)

	return &MongoHelper{
		Client:   prodClient.Mongo,
		Database: prodClient.Mongo.Database(dbName),
		DBName:   dbName,
	}
}

func (m *MongoHelper) Close(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.Client.Disconnect(ctx); err != nil {
		t.Logf("warning: failed to disconnect from MongoDB: %v", err)
	}
}

func (m *MongoHelper) CleanDatabase(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := m.Database.ListCollectionNames(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("failed to list collections: %v", err)
	}

	for _, collName := range collections {
		if collName == "_migrations" || collName == "system.indexes" {
			continue
		}

		if err := m.Database.Collection(collName).Drop(ctx); err != nil {
			t.Fatalf("failed to drop collection %s: %v", collName, err)
		}
		t.Logf("Dropped collection: %s", collName)
	}
}

func (m *MongoHelper) CleanCollection(t *testing.T, collectionName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.Database.Collection(collectionName).DeleteMany(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("failed to clean collection %s: %v", collectionName, err)
	}
	t.Logf("Cleaned %d documents from collection: %s", result.DeletedCount, collectionName)
}

func (m *MongoHelper) CountDocuments(t *testing.T, collectionName string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := m.Database.Collection(collectionName).CountDocuments(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("failed to count documents in %s: %v", collectionName, err)
	}
	return count
}

func (m *MongoHelper) GetCollection(collectionName string) *mongo.Collection {
	return m.Database.Collection(collectionName)
}
