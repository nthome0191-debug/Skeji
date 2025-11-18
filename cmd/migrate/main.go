package main

import (
	"context"
	"fmt"
	"log"
	"time"

	mongoMigration "skeji/internal/migrations/mongo"
	"skeji/pkg/config"
)

const JobName = "mongo-migration"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cfg := config.Load(JobName)
	cfg.SetMongo()
	cfg.Log.Info("Starting Mongo migration job")
	defer cfg.GracefulShutdown()
	migrateMongo(ctx, cfg)
	fmt.Println("Migration completed successfully.")
}

func migrateMongo(ctx context.Context, cfg *config.Config) {
	if err := mongoMigration.RunMigration(ctx, cfg.Client.Mongo.Client); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}
