package testutil

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DefaultMongoURI      = "mongodb://localhost:27017"
	DefaultDatabaseName  = "skeji"
	ConnectionTimeout    = 10 * time.Second
	BusinessUnitsCollection = "Business_units"
)

// MongoHelper provides MongoDB test utilities
type MongoHelper struct {
	Client   *mongo.Client
	Database *mongo.Database
	DBName   string
}

// NewMongoHelper creates a new MongoDB test helper
func NewMongoHelper(t *testing.T, mongoURI, dbName string) *MongoHelper {
	t.Helper()

	if mongoURI == "" {
		mongoURI = DefaultMongoURI
	}
	if dbName == "" {
		dbName = DefaultDatabaseName
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping MongoDB: %v", err)
	}

	t.Log("Connected to MongoDB successfully")

	return &MongoHelper{
		Client:   client,
		Database: client.Database(dbName),
		DBName:   dbName,
	}
}

// Close closes MongoDB connection
func (m *MongoHelper) Close(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.Client.Disconnect(ctx); err != nil {
		t.Logf("warning: failed to disconnect from MongoDB: %v", err)
	}
}

// CleanDatabase drops all collections to ensure clean state
func (m *MongoHelper) CleanDatabase(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := m.Database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to list collections: %v", err)
	}

	for _, collName := range collections {
		// Skip system collections and migrations
		if collName == "_migrations" || collName == "system.indexes" {
			continue
		}

		if err := m.Database.Collection(collName).Drop(ctx); err != nil {
			t.Fatalf("failed to drop collection %s: %v", collName, err)
		}
		t.Logf("Dropped collection: %s", collName)
	}
}

// CleanCollection removes all documents from a specific collection
func (m *MongoHelper) CleanCollection(t *testing.T, collectionName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.Database.Collection(collectionName).DeleteMany(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to clean collection %s: %v", collectionName, err)
	}
	t.Logf("Cleaned %d documents from collection: %s", result.DeletedCount, collectionName)
}

// CountDocuments returns the number of documents in a collection
func (m *MongoHelper) CountDocuments(t *testing.T, collectionName string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := m.Database.Collection(collectionName).CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to count documents in %s: %v", collectionName, err)
	}
	return count
}

// GetCollection returns a collection for direct access
func (m *MongoHelper) GetCollection(collectionName string) *mongo.Collection {
	return m.Database.Collection(collectionName)
}
