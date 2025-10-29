package repository

// TODO: Implement data access layer for Business Units
// Responsibilities:
// - MongoDB CRUD operations
// - Query methods (FindByID, FindAll, Search, etc.)
// - Use models from pkg/model/business_unit.go
// - No business logic here - pure data access

import (
	"context"
	"errors"
	"fmt"
	"skeji/pkg/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DB_NAME         = "skeji"
	COLLECTION_NAME = "Business_units"
)

type mongoBusinessUnitRepository struct {
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
	FindByCity(ctx context.Context, city string) ([]*model.BusinessUnit, error)
	Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error)

	Count(ctx context.Context) (int64, error)
}

func NewMongoBusinessUnitRepository(client *mongo.Client) BusinessUnitRepository {
	db := client.Database(DB_NAME)
	return &mongoBusinessUnitRepository{
		db:         db,
		collection: db.Collection(COLLECTION_NAME),
	}
}

func (r *mongoBusinessUnitRepository) Create(ctx context.Context, bu *model.BusinessUnit) error {
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
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}
	filter := bson.M{"_id": objectID}

	var bu model.BusinessUnit
	err = r.collection.FindOne(ctx, filter).Decode(&bu)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("business unit not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find business unit: %w", err)
	}
	return &bu, nil
}

func (r *mongoBusinessUnitRepository) FindAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, error) {
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
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
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
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update business unit: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("business unit not found: %s", id)
	}

	return nil
}

func (r *mongoBusinessUnitRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete business unit: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("business unit not found: %s", id)
	}

	return nil
}

func (r *mongoBusinessUnitRepository) Search(ctx context.Context, cities []string, labels []string) ([]*model.BusinessUnit, error) {
	filter := bson.M{}
	if len(cities) > 0 {
		filter["cities"] = bson.M{"$in": cities}
	}
	if len(labels) > 0 {
		filter["labels"] = bson.M{"$in": labels}
	}

	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}})

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

func (r *mongoBusinessUnitRepository) FindByCity(ctx context.Context, city string) ([]*model.BusinessUnit, error) {
	filter := bson.M{"cities": city}

	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find business units in city [%s]: %w", city, err)
	}
	defer cursor.Close(ctx)

	var businessUnits []*model.BusinessUnit
	if err = cursor.All(ctx, &businessUnits); err != nil {
		return nil, fmt.Errorf("failed to decode business units: %w", err)
	}
	return businessUnits, nil
}

func (r *mongoBusinessUnitRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count business units: %w", err)
	}
	return count, nil
}
