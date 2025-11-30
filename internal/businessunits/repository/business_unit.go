package repository

import (
	"context"
	"errors"
	"fmt"
	businessunitserrors "skeji/internal/businessunits/errors"
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
	CollectionName = "Business_units"
)

type mongoBusinessUnitRepository struct {
	cfg        *config.Config
	db         *mongo.Database
	collection *mongo.Collection
	txManager  mongotx.TransactionManager
}

type BusinessUnitRepository interface {
	Create(ctx context.Context, bu *model.BusinessUnit) error
	FindByID(ctx context.Context, id string) (*model.BusinessUnit, error)
	FindAll(ctx context.Context, limit int, offset int64) ([]*model.BusinessUnit, error)
	Update(ctx context.Context, id string, bu *model.BusinessUnit) (*mongo.UpdateResult, error)
	Delete(ctx context.Context, id string) error

	GetByPhone(ctx context.Context, phone string, cities []string, labels []string, limit int, offset int64) ([]*model.BusinessUnit, error)
	CountByPhone(ctx context.Context, phone string, cities []string, labels []string) (int64, error)
	SearchByCityLabelPairs(ctx context.Context, pairs []string, limit int, offset int64) ([]*model.BusinessUnit, error)
	CountByCityLabelPairs(ctx context.Context, pairs []string) (int64, error)
	Count(ctx context.Context) (int64, error)

	ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error
}

func NewMongoBusinessUnitRepository(cfg *config.Config) BusinessUnitRepository {
	db := cfg.Client.Mongo.Client.Database(cfg.MongoDatabaseName)
	return &mongoBusinessUnitRepository{
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
func (r *mongoBusinessUnitRepository) withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.(mongo.SessionContext); ok {
		// Inside transaction - cannot wrap SessionContext, return no-op cancel
		return ctx, func() {}
	}

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return context.WithTimeout(ctx, timeout)
	}

	remaining := time.Until(deadline)
	// Use the shorter of remaining time or requested timeout
	if remaining < timeout {
		return context.WithTimeout(ctx, remaining)
	}

	return context.WithTimeout(ctx, timeout)
}

func (r *mongoBusinessUnitRepository) Create(ctx context.Context, bu *model.BusinessUnit) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	bu.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	result, err := r.collection.InsertOne(ctx, bu)
	if err != nil {
		return fmt.Errorf("failed to create business unit: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		bu.ID = oid.Hex()
	}

	return nil
}

func (r *mongoBusinessUnitRepository) FindByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", businessunitserrors.ErrInvalidID, id)
	}
	filter := bson.M{"_id": objectID}

	var bu model.BusinessUnit
	err = r.collection.FindOne(ctx, filter).Decode(&bu)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: %s", businessunitserrors.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to find business unit: %w", err)
	}
	return &bu, nil
}

func (r *mongoBusinessUnitRepository) FindAll(ctx context.Context, limit int, offset int64) ([]*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "priority", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query business units: %w", err)
	}
	defer cursor.Close(ctx)

	var businessUnits []*model.BusinessUnit
	if err = cursor.All(ctx, &businessUnits); err != nil {
		return nil, fmt.Errorf("failed to decode business units: %w", err)
	}

	return businessUnits, nil
}

func (r *mongoBusinessUnitRepository) Update(ctx context.Context, id string, bu *model.BusinessUnit) (*mongo.UpdateResult, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", businessunitserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"name":             bu.Name,
			"cities":           bu.Cities,
			"labels":           bu.Labels,
			"admin_phone":      bu.AdminPhone,
			"maintainers":      bu.Maintainers,
			"priority":         bu.Priority,
			"time_zone":        bu.TimeZone,
			"website_urls":     bu.WebsiteURLs,
			"city_label_pairs": bu.CityLabelPairs,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update business unit: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("%w: %s", businessunitserrors.ErrNotFound, id)
	}

	return result, nil
}

func (r *mongoBusinessUnitRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("%w: %s", businessunitserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete business unit: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("%w: %s", businessunitserrors.ErrNotFound, id)
	}

	return nil
}

func (r *mongoBusinessUnitRepository) SearchByCityLabelPairs(ctx context.Context, pairs []string, limit int, offset int64) ([]*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{"city_label_pairs": bson.M{"$in": pairs}}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "priority", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find business units by city_label_pairs: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*model.BusinessUnit
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode business units: %w", err)
	}

	return results, nil
}

func (r *mongoBusinessUnitRepository) CountByCityLabelPairs(ctx context.Context, pairs []string) (int64, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{"city_label_pairs": bson.M{"$in": pairs}}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count business units by city_label_pairs: %w", err)
	}
	return count, nil
}

func (r *mongoBusinessUnitRepository) GetByPhone(
	ctx context.Context,
	phone string,
	cities []string,
	labels []string,
	limit int,
	offset int64,
) ([]*model.BusinessUnit, error) {

	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"admin_phone": phone},
			{fmt.Sprintf("maintainers.%s", phone): bson.M{"$exists": true}},
		},
	}

	if len(cities) > 0 {
		filter["cities"] = bson.M{"$in": cities}
	}

	if len(labels) > 0 {
		filter["labels"] = bson.M{"$in": labels}
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "priority", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find business units for phone [%s]: %w", phone, err)
	}
	defer cursor.Close(ctx)

	var businessUnits []*model.BusinessUnit
	if err := cursor.All(ctx, &businessUnits); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return businessUnits, nil
}

func (r *mongoBusinessUnitRepository) CountByPhone(
	ctx context.Context,
	phone string,
	cities []string,
	labels []string,
) (int64, error) {

	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"admin_phone": phone},
			{fmt.Sprintf("maintainers.%s", phone): bson.M{"$exists": true}},
		},
	}

	if len(cities) > 0 {
		filter["cities"] = bson.M{"$in": cities}
	}

	if len(labels) > 0 {
		filter["labels"] = bson.M{"$in": labels}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count business units for phone [%s]: %w", phone, err)
	}

	return count, nil
}

func (r *mongoBusinessUnitRepository) Count(ctx context.Context) (int64, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count business units: %w", err)
	}
	return count, nil
}

func (r *mongoBusinessUnitRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return r.txManager.ExecuteTransaction(ctx, fn)
}
