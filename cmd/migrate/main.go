package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	mongoMigration "skeji/internal/migrations/mongo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is required")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	fmt.Printf("Connected to MongoDB: %s\n", mongoURI)

	if err := mongoMigration.RunMigration(ctx, client); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully.")
}
