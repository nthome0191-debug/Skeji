package repository

import (
	"context"
	"errors"
	"fmt"
	bookingserrors "skeji/internal/bookings/errors"
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
	CollectionName = "Bookings"
)

type mongoBookingRepository struct {
	cfg        *config.Config
	db         *mongo.Database
	collection *mongo.Collection
	txManager  mongotx.TransactionManager
}

type BookingRepository interface {
	Create(ctx context.Context, booking *model.Booking) error
	FindByID(ctx context.Context, id string) (*model.Booking, error)
	FindAll(ctx context.Context, limit int, offset int64) ([]*model.Booking, error)
	Update(ctx context.Context, id string, booking *model.Booking) (*mongo.UpdateResult, error)
	Delete(ctx context.Context, id string) error
	FindByBusinessAndSchedule(ctx context.Context, businessID string, scheduleID string, startTime *time.Time, endTime *time.Time, limit int, offset int64) ([]*model.Booking, error)
	CountByBusinessAndSchedule(ctx context.Context, businessID string, scheduleID string, startTime *time.Time, endTime *time.Time) (int64, error)
	Count(ctx context.Context) (int64, error)
	ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error
}

func NewMongoBookingRepository(cfg *config.Config) BookingRepository {
	db := cfg.Client.Mongo.Client.Database(cfg.MongoDatabaseName)
	return &mongoBookingRepository{
		cfg:        cfg,
		db:         db,
		collection: db.Collection(CollectionName),
		txManager:  mongotx.NewTransactionManager(cfg.Client.Mongo.Client),
	}
}

// withTimeout wraps the context with a timeout if not already in a transaction.
// When inside a transaction (SessionContext), returns the original context unchanged
// with a no-op cancel function, as we cannot wrap SessionContext without breaking
// transaction semantics.
func (r *mongoBookingRepository) withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.(mongo.SessionContext); ok {
		// Inside transaction - cannot wrap SessionContext, return no-op cancel
		return ctx, func() {}
	}

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return context.WithTimeout(ctx, timeout)
	}

	remaining := time.Until(deadline)
	if remaining < timeout {
		return context.WithTimeout(ctx, remaining)
	}

	return context.WithTimeout(ctx, timeout)
}

func (r *mongoBookingRepository) Create(ctx context.Context, booking *model.Booking) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	booking.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	result, err := r.collection.InsertOne(ctx, booking)
	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		booking.ID = oid.Hex()
	}
	return nil
}

func (r *mongoBookingRepository) FindByID(ctx context.Context, id string) (*model.Booking, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", bookingserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}

	var booking model.Booking
	err = r.collection.FindOne(ctx, filter).Decode(&booking)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, bookingserrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find booking: %w", err)
	}

	return &booking, nil
}

func (r *mongoBookingRepository) FindAll(ctx context.Context, limit int, offset int64) ([]*model.Booking, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: 1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find bookings: %w", err)
	}
	defer cursor.Close(ctx)

	var bookings []*model.Booking
	if err = cursor.All(ctx, &bookings); err != nil {
		return nil, fmt.Errorf("failed to decode bookings: %w", err)
	}

	return bookings, nil
}

func (r *mongoBookingRepository) Update(ctx context.Context, id string, booking *model.Booking) (*mongo.UpdateResult, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", bookingserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"service_label": booking.ServiceLabel,
			"start_time":    booking.StartTime,
			"end_time":      booking.EndTime,
			"capacity":      booking.Capacity,
			"participants":  booking.Participants,
			"status":        booking.Status,
			"managed_by":    booking.ManagedBy,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, bookingserrors.ErrNotFound
	}

	return result, nil
}

func (r *mongoBookingRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("%w: %s", bookingserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete booking: %w", err)
	}

	if result.DeletedCount == 0 {
		return bookingserrors.ErrNotFound
	}

	return nil
}

func (r *mongoBookingRepository) FindByBusinessAndSchedule(
	ctx context.Context,
	businessID string,
	scheduleID string,
	startTime, endTime *time.Time,
	limit int, offset int64,
) ([]*model.Booking, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := r.buildSearchFilter(businessID, scheduleID, startTime, endTime)

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "start_time", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find bookings: %w", err)
	}
	defer cursor.Close(ctx)

	var bookings []*model.Booking
	if err = cursor.All(ctx, &bookings); err != nil {
		return nil, fmt.Errorf("failed to decode bookings: %w", err)
	}

	return bookings, nil
}

func (r *mongoBookingRepository) CountByBusinessAndSchedule(
	ctx context.Context,
	businessID string,
	scheduleID string,
	startTime, endTime *time.Time,
) (int64, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := r.buildSearchFilter(businessID, scheduleID, startTime, endTime)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings by search: %w", err)
	}
	return count, nil
}

func (r *mongoBookingRepository) buildSearchFilter(businessID string, scheduleID string, startTime, endTime *time.Time) bson.M {
	filter := bson.M{
		"business_id": businessID,
		"schedule_id": scheduleID,
	}

	if startTime != nil || endTime != nil {
		timeFilters := bson.M{}
		if startTime != nil && endTime != nil {
			timeFilters = bson.M{
				"start_time": bson.M{"$lt": *endTime},
				"end_time":   bson.M{"$gt": *startTime},
			}
		} else if startTime != nil {
			timeFilters = bson.M{
				"end_time": bson.M{"$gt": *startTime},
			}
		} else if endTime != nil {
			timeFilters = bson.M{
				"start_time": bson.M{"$lt": *endTime},
			}
		}

		filter["$and"] = []bson.M{timeFilters}
	}

	return filter
}

func (r *mongoBookingRepository) Count(ctx context.Context) (int64, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings: %w", err)
	}

	return count, nil
}

func (r *mongoBookingRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return r.txManager.ExecuteTransaction(ctx, fn)
}
