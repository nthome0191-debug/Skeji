package repository

import (
	"context"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BookingLockRepository provides operations for advisory locks
type BookingLockRepository interface {
	Create(ctx context.Context, lock *model.BookingLock) (*model.BookingLock, error)
	Delete(ctx context.Context, lockID string) error
}

type mongoBookingLockRepository struct {
	collection *mongo.Collection
}

func NewBookingLockRepository(cfg *config.Config) BookingLockRepository {
	db := cfg.Client.Mongo.Client.Database(cfg.MongoDatabaseName)
	return &mongoBookingLockRepository{
		collection: db.Collection("Booking_locks"),
	}
}

// Returns duplicate key error if lock already exists
func (r *mongoBookingLockRepository) Create(ctx context.Context, lock *model.BookingLock) (*model.BookingLock, error) {
	lock.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, lock)
	if err != nil {
		return nil, err
	}

	return lock, nil
}

// Delete removes an advisory lock
func (r *mongoBookingLockRepository) Delete(ctx context.Context, lockID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": lockID})
	return err
}
