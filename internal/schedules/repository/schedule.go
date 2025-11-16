package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	scheduleserrors "skeji/internal/schedules/errors"
	"skeji/pkg/config"
	mongotx "skeji/pkg/db/mongo"
	"skeji/pkg/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionName = "Schedules"
)

type mongoScheduleRepository struct {
	cfg        *config.Config
	db         *mongo.Database
	collection *mongo.Collection
	txManager  mongotx.TransactionManager
}

type ScheduleRepository interface {
	Create(ctx context.Context, sc *model.Schedule) error
	FindByID(ctx context.Context, id string) (*model.Schedule, error)
	FindAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, error)
	Update(ctx context.Context, id string, sc *model.Schedule) (*mongo.UpdateResult, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, businessId string, city string) ([]*model.Schedule, error)
	Count(ctx context.Context) (int64, error)
	ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error
}

func NewMongoScheduleRepository(cfg *config.Config) ScheduleRepository {
	db := cfg.Client.Mongo.Database(cfg.MongoDatabaseName)
	return &mongoScheduleRepository{
		cfg:        cfg,
		db:         db,
		collection: db.Collection(CollectionName),
		txManager:  mongotx.NewTransactionManager(cfg.Client.Mongo),
	}
}

// withTimeout wraps the context with a timeout if not already in a transaction.
// When inside a transaction (SessionContext), returns the original context unchanged
// with a no-op cancel function, as we cannot wrap SessionContext without breaking
// transaction semantics.
func (r *mongoScheduleRepository) withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.(mongo.SessionContext); ok {
		// Inside transaction - cannot wrap SessionContext, return no-op cancel
		return ctx, func() {}
	}

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return context.WithTimeout(ctx, timeout)
	}

	remaining := time.Until(deadline)
	if remaining > timeout {
		return context.WithTimeout(ctx, remaining)
	}

	return context.WithTimeout(ctx, timeout)
}

func (r *mongoScheduleRepository) Create(ctx context.Context, sc *model.Schedule) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	sc.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	result, err := r.collection.InsertOne(ctx, sc)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		sc.ID = oid.Hex()
	}
	return nil
}

func (r *mongoScheduleRepository) FindByID(ctx context.Context, id string) (*model.Schedule, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", scheduleserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}

	var sc model.Schedule
	err = r.collection.FindOne(ctx, filter).Decode(&sc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: %s", scheduleserrors.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to find schedule: %w", err)
	}

	return &sc, nil
}

func (r *mongoScheduleRepository) FindAll(ctx context.Context, limit int, offset int) ([]*model.Schedule, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer cursor.Close(ctx)

	var schedules []*model.Schedule
	if err = cursor.All(ctx, &schedules); err != nil {
		return nil, fmt.Errorf("failed to decode schedules: %w", err)
	}
	return schedules, nil
}

func (r *mongoScheduleRepository) Update(ctx context.Context, id string, sc *model.Schedule) (*mongo.UpdateResult, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", scheduleserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"business_id":                  sc.BusinessID,
			"name":                         sc.Name,
			"city":                         sc.City,
			"address":                      sc.Address,
			"start_of_day":                 sc.StartOfDay,
			"end_of_day":                   sc.EndOfDay,
			"working_days":                 sc.WorkingDays,
			"default_meeting_duration_min": sc.DefaultMeetingDurationMin,
			"default_break_duration_min":   sc.DefaultBreakDurationMin,
			"max_participants_per_slot":    sc.MaxParticipantsPerSlot,
			"exceptions":                   sc.Exceptions,
			"time_zone":                    sc.TimeZone,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("%w: %s", scheduleserrors.ErrNotFound, id)
	}

	return result, nil
}

func (r *mongoScheduleRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("%w: %s", scheduleserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("%w: %s", scheduleserrors.ErrNotFound, id)
	}
	return nil
}

// escapeRegexSpecialChars escapes special regex characters to prevent ReDoS attacks
func escapeRegexSpecialChars(s string) string {
	// Escape special regex characters: . * + ? ^ $ ( ) [ ] { } | \
	specialChars := regexp.MustCompile(`[.*+?^$()[\]{}|\\]`)
	return specialChars.ReplaceAllStringFunc(s, func(match string) string {
		return "\\" + match
	})
}

func (r *mongoScheduleRepository) Search(ctx context.Context, businessId string, city string) ([]*model.Schedule, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{}
	if businessId != "" {
		filter["business_id"] = businessId
	}
	if city != "" {
		// Escape special regex characters to prevent ReDoS attacks
		escapedCity := escapeRegexSpecialChars(city)
		filter["city"] = bson.M{"$regex": escapedCity, "$options": "i"}
	}

	const maxSearchResults = 1000
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}). //todo: why we need this? necessary? would mongo search work without it?
		SetLimit(maxSearchResults)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search schedules: %w", err)
	}
	defer cursor.Close(ctx)

	var schedules []*model.Schedule
	if err = cursor.All(ctx, &schedules); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return schedules, nil
}

func (r *mongoScheduleRepository) Count(ctx context.Context) (int64, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count schedules: %w", err)
	}
	return count, nil
}

func (r *mongoScheduleRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return r.txManager.ExecuteTransaction(ctx, fn)
}
