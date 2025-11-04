package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"skeji/internal/migrations/mongo/validators"
)

const DatabaseName = "skeji"

var (
	BusinessUnitsIndexes = []mongo.IndexModel{
		{Keys: bson.D{{Key: "admin_phone", Value: 1}}},
		{Keys: bson.D{{Key: "maintainers", Value: 1}}},
		{Keys: bson.D{
			{Key: "cities", Value: 1},
			{Key: "labels", Value: 1},
			{Key: "priority", Value: -1},
		}},
	}

	SchedulesIndexes = []mongo.IndexModel{
		{Keys: bson.D{{Key: "_id", Value: 1}}},
		{Keys: bson.D{
			{Key: "business_id", Value: 1},
			{Key: "city", Value: 1},
		}},
	}

	BookingsIndexes = []mongo.IndexModel{
		{Keys: bson.D{
			{Key: "business_id", Value: 1},
			{Key: "schedule_id", Value: 1},
			{Key: "start_time", Value: 1},
			{Key: "end_time", Value: 1},
		}},
		{Keys: bson.D{
			{Key: "participants", Value: 1},
			{Key: "start_time", Value: 1},
		}},
	}
)

func RunMigration(ctx context.Context, client *mongo.Client) error {
	db := client.Database(DatabaseName)
	fmt.Printf("🚀 Running Skeji Mongo migrations on database: %s\n", DatabaseName)

	collections := map[string]struct {
		Indexes   []mongo.IndexModel
		Validator bson.M
	}{
		"Business_units": {
			Indexes:   BusinessUnitsIndexes,
			Validator: validators.BusinessUnitValidator,
		},
		"Schedules": {
			Indexes:   SchedulesIndexes,
			Validator: validators.ScheduleValidator,
		},
		"Bookings": {
			Indexes:   BookingsIndexes,
			Validator: validators.BookingValidator,
		},
	}

	for name, def := range collections {
		fmt.Printf("\n🔸 Migrating collection: %s\n", name)

		if err := ensureCollection(ctx, db, name, def.Validator); err != nil {
			return fmt.Errorf("❌ failed to ensure collection %s: %w", name, err)
		}
		if err := ensureIndexes(ctx, db, name, def.Indexes); err != nil {
			return fmt.Errorf("❌ failed to ensure indexes for %s: %w", name, err)
		}
		if err := logMigration(ctx, db, name); err != nil {
			fmt.Printf("⚠️  Warning: failed to log migration for %s: %v\n", name, err)
		}
	}

	fmt.Println("\n✅ All migrations applied successfully.")
	return nil
}

func ensureCollection(ctx context.Context, db *mongo.Database, name string, validator bson.M) error {
	existing, err := db.ListCollectionNames(ctx, bson.D{{Key: "name", Value: name}})
	if err != nil {
		return err
	}

	if len(existing) == 0 {
		fmt.Printf("🆕 Creating collection: %s\n", name)
		opts := options.CreateCollection().
			SetValidator(validator).
			SetValidationLevel("strict")

		if err := db.CreateCollection(ctx, name, opts); err != nil {
			return fmt.Errorf("failed creating %s: %w", name, err)
		}
	} else {
		fmt.Printf("ℹ️  Collection %s already exists — updating validator if needed\n", name)

		command := bson.D{
			{Key: "collMod", Value: name},
			{Key: "validator", Value: validator},
			{Key: "validationLevel", Value: "strict"},
		}

		collCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := db.RunCommand(collCtx, command).Err(); err != nil {
			fmt.Printf("⚠️  Warning: failed updating validator for %s: %v\n", name, err)
		}
	}

	return nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database, name string, models []mongo.IndexModel) error {
	coll := db.Collection(name)
	if len(models) == 0 {
		return nil
	}

	_, err := coll.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("failed creating indexes for %s: %w", name, err)
	}

	fmt.Printf("📚 Ensured indexes for %s\n", name)
	return nil
}

func logMigration(ctx context.Context, db *mongo.Database, name string) error {
	meta := db.Collection("_migrations")
	_, err := meta.UpdateOne(
		ctx,
		bson.M{"collection": name},
		bson.M{"$set": bson.M{"applied_at": time.Now()}},
		options.Update().SetUpsert(true),
	)
	return err
}
