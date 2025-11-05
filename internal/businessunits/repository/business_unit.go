package repository

import (
	"context"
	"errors"
	"fmt"
	businessunitserrors "skeji/internal/businessunits/errors"
	"skeji/pkg/config"
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
}

type BusinessUnitRepository interface {
	Create(ctx context.Context, bu *model.BusinessUnit) error
	FindByID(ctx context.Context, id string) (*model.BusinessUnit, error)
	FindAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error)
	Update(ctx context.Context, id string, bu *model.BusinessUnit) error
	Delete(ctx context.Context, id string) error

	FindByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error)
	Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error)

	Count(ctx context.Context) (int64, error)
}

func NewMongoBusinessUnitRepository(cfg *config.Config) BusinessUnitRepository {
	db := cfg.Client.Mongo.Database(cfg.MongoDatabaseName)
	return &mongoBusinessUnitRepository{
		cfg:        cfg,
		db:         db,
		collection: db.Collection(CollectionName),
	}
}

func (r *mongoBusinessUnitRepository) withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
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

func (r *mongoBusinessUnitRepository) Create(ctx context.Context, bu *model.BusinessUnit) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	bu.CreatedAt = time.Now()
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

func (r *mongoBusinessUnitRepository) FindAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	if limit == 0 {
		limit = 10
	}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "priority", Value: -1}})
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

func (r *mongoBusinessUnitRepository) Update(ctx context.Context, id string, bu *model.BusinessUnit) error {
	ctx, cancel := r.withTimeout(ctx, r.cfg.WriteTimeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("%w: %s", businessunitserrors.ErrInvalidID, id)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"name":        bu.Name,
			"cities":      bu.Cities,
			"labels":      bu.Labels,
			"admin_phone": bu.AdminPhone,
			"maintainers": bu.Maintainers,
			"priority":    bu.Priority,
			"time_zone":   bu.TimeZone,
			"website_url": bu.WebsiteURL,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update business unit: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: %s", businessunitserrors.ErrNotFound, id)
	}

	return nil
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

func (r *mongoBusinessUnitRepository) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{}
	if len(cities) > 0 {
		filter["cities"] = bson.M{"$in": cities}
	}
	if len(labels) > 0 {
		filter["labels"] = bson.M{"$in": labels}
	}

	const maxSearchResults = 1000
	opts := options.Find().
		SetSort(bson.D{{Key: "priority", Value: -1}}).
		SetLimit(maxSearchResults)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search business units: %w", err)
	}
	defer cursor.Close(ctx)

	var businessUnits []*model.BusinessUnit
	if err = cursor.All(ctx, &businessUnits); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return businessUnits, nil
}

func (r *mongoBusinessUnitRepository) FindByAdminPhone(ctx context.Context, phone string) ([]*model.BusinessUnit, error) {
	ctx, cancel := r.withTimeout(ctx, r.cfg.ReadTimeout)
	defer cancel()

	filter := bson.M{"admin_phone": phone}

	cursor, err := r.collection.Find(ctx, filter, options.Find())
	if err != nil {
		return nil, fmt.Errorf("failed to find business units for phone [%s]: %w", phone, err)
	}
	defer cursor.Close(ctx)

	var businessUnits []*model.BusinessUnit
	if err = cursor.All(ctx, &businessUnits); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}
	return businessUnits, nil
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
