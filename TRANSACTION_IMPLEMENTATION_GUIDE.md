# MongoDB Transactions Implementation Guide

## Overview

Your codebase already has **transaction infrastructure** in place but it's **not yet being used**. Here's how to implement it.

---

## 🏗️ What You Already Have

### 1. Transaction Infrastructure (`internal/businessunits/repository/transactions.go`)

```go
type TransactionFunc func(ctx mongo.SessionContext) error

func (r *mongoBusinessUnitRepository) ExecuteTransaction(ctx context.Context, fn TransactionFunc) error {
    session, err := r.db.Client().StartSession()
    if err != nil {
        return fmt.Errorf("failed to start session: %w", err)
    }
    defer session.EndSession(ctx)

    _, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
        return nil, fn(sessCtx)
    })

    return err
}
```

**Key Features:**
- ✅ Automatic commit/rollback
- ✅ Automatic retry on transient errors
- ✅ Session management
- ✅ Well-documented

---

## ⚠️ Prerequisites

### MongoDB Must Be a Replica Set

Transactions **only work** on MongoDB replica sets, not standalone instances.

**Check your setup:**
```bash
# Connect to MongoDB
mongosh

# Check replica set status
rs.status()

# If standalone, convert to replica set:
# 1. Edit mongod.conf
replication:
  replSetName: "rs0"

# 2. Restart MongoDB
# 3. Initialize replica set
rs.initiate()
```

**In Kind/Kubernetes:**
Your `deployment/local/mongo/` setup should configure replica set. Check if it's already configured.

---

## 🎯 When to Use Transactions

### ✅ Use Transactions For:

1. **Multi-Collection Operations**
   ```
   Example: Create business unit + initial schedule
   ```

2. **Cascading Updates**
   ```
   Example: Update business unit location → Update all schedules
   ```

3. **Atomic Checks + Inserts**
   ```
   Example: Check for duplicate + insert (prevent race conditions)
   ```

### ❌ Don't Use Transactions For:

1. **Single Document Operations** (already atomic)
2. **Read-only queries** (unnecessary overhead)
3. **Operations with external API calls** (transactions should be fast)

---

## 📝 Implementation Examples

### Example 1: Simple Transaction (Single Collection)

**Scenario:** Prevent duplicate business units with same admin_phone

**Without Transaction (Race Condition):**
```go
func (s *businessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
    // Check if exists
    existing, _ := s.repo.FindByAdminPhone(ctx, bu.AdminPhone)
    if len(existing) > 0 {
        return errors.New("already exists")
    }

    // Race condition here! Another request could insert before us

    // Create
    return s.repo.Create(ctx, bu)
}
```

**With Transaction (Safe):**

#### Step 1: Add transaction method to repository

```go
// internal/businessunits/repository/business_unit.go

func (r *mongoBusinessUnitRepository) CreateWithDuplicateCheck(ctx context.Context, bu *model.BusinessUnit) error {
    return r.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
        // Check for duplicates INSIDE transaction
        filter := bson.M{"admin_phone": bu.AdminPhone}
        count, err := r.collection.CountDocuments(sessCtx, filter)
        if err != nil {
            return fmt.Errorf("failed to check for duplicates: %w", err)
        }

        if count > 0 {
            return businessunitserrors.ErrAlreadyExists
        }

        // Insert INSIDE same transaction
        bu.CreatedAt = time.Now()
        result, err := r.collection.InsertOne(sessCtx, bu)
        if err != nil {
            return fmt.Errorf("failed to create business unit: %w", err)
        }

        if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
            bu.ID = oid.Hex()
        }

        return nil
    })
}
```

#### Step 2: Update repository interface

```go
type BusinessUnitRepository interface {
    Create(ctx context.Context, bu *model.BusinessUnit) error
    CreateWithDuplicateCheck(ctx context.Context, bu *model.BusinessUnit) error  // Add this
    // ... rest of methods
}
```

#### Step 3: Use in service layer

```go
func (s *businessUnitService) Create(ctx context.Context, bu *model.BusinessUnit) error {
    s.sanitize(bu)
    s.applyDefaultsForNewCreatedBusiness(bu)

    if err := s.validator.Validate(bu); err != nil {
        return apperrors.Validation("validation failed", nil)
    }

    // Use transaction method instead
    if err := s.repo.CreateWithDuplicateCheck(ctx, bu); err != nil {
        if errors.Is(err, businessunitserrors.ErrAlreadyExists) {
            return apperrors.Validation("Business unit already exists", map[string]any{
                "admin_phone": bu.AdminPhone,
            })
        }
        return apperrors.Internal("Failed to create business unit", err)
    }

    return nil
}
```

---

### Example 2: Multi-Collection Transaction

**Scenario:** Create business unit + initial schedule atomically

#### Step 1: Add schedule collection to repository

```go
type mongoBusinessUnitRepository struct {
    cfg                *config.Config
    db                 *mongo.Database
    collection         *mongo.Collection
    scheduleCollection *mongo.Collection  // Add this
}

func NewMongoBusinessUnitRepository(cfg *config.Config) BusinessUnitRepository {
    db := cfg.Client.Mongo.Database(cfg.MongoDatabaseName)
    return &mongoBusinessUnitRepository{
        cfg:                cfg,
        db:                 db,
        collection:         db.Collection("Business_units"),
        scheduleCollection: db.Collection("Schedules"),  // Add this
    }
}
```

#### Step 2: Create transaction method

```go
func (r *mongoBusinessUnitRepository) CreateWithSchedule(
    ctx context.Context,
    bu *model.BusinessUnit,
    schedule *model.Schedule,
) error {
    return r.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
        // 1. Insert business unit
        bu.CreatedAt = time.Now()
        result, err := r.collection.InsertOne(sessCtx, bu)
        if err != nil {
            return fmt.Errorf("failed to create business unit: %w", err)
        }

        if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
            bu.ID = oid.Hex()
        }

        // 2. Insert schedule with business_id reference
        schedule.BusinessID = bu.ID
        schedule.CreatedAt = time.Now()
        _, err = r.scheduleCollection.InsertOne(sessCtx, schedule)
        if err != nil {
            return fmt.Errorf("failed to create schedule: %w", err)
        }

        // Both succeed or both rollback
        return nil
    })
}
```

---

### Example 3: Update with Cascade (Multi-Document)

**Scenario:** When business unit cities change, update all related schedules

```go
func (r *mongoBusinessUnitRepository) UpdateWithScheduleCascade(
    ctx context.Context,
    id string,
    bu *model.BusinessUnit,
    updatedCities []string,
) error {
    return r.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            return fmt.Errorf("%w: %s", businessunitserrors.ErrInvalidID, id)
        }

        // 1. Update business unit
        filter := bson.M{"_id": objectID}
        update := bson.M{"$set": bu}
        result, err := r.collection.UpdateOne(sessCtx, filter, update)
        if err != nil {
            return fmt.Errorf("failed to update business unit: %w", err)
        }

        if result.MatchedCount == 0 {
            return fmt.Errorf("%w: %s", businessunitserrors.ErrNotFound, id)
        }

        // 2. Update all schedules for this business unit
        scheduleFilter := bson.M{"business_id": id}
        scheduleUpdate := bson.M{
            "$set": bson.M{
                "valid_cities": updatedCities,
            },
        }
        _, err = r.scheduleCollection.UpdateMany(sessCtx, scheduleFilter, scheduleUpdate)
        if err != nil {
            return fmt.Errorf("failed to update related schedules: %w", err)
        }

        return nil
    })
}
```

---

## 🧪 Testing Transactions

### Test 1: Successful Transaction

```go
func TestCreateWithDuplicateCheck_Success(t *testing.T) {
    repo := setupTestRepository(t)
    ctx := context.Background()

    bu := &model.BusinessUnit{
        Name:       "Test Business",
        AdminPhone: "+972501234567",
        // ... other fields
    }

    err := repo.CreateWithDuplicateCheck(ctx, bu)
    assert.NoError(t, err)
    assert.NotEmpty(t, bu.ID)
}
```

### Test 2: Transaction Rollback

```go
func TestCreateWithDuplicateCheck_Duplicate(t *testing.T) {
    repo := setupTestRepository(t)
    ctx := context.Background()

    bu1 := &model.BusinessUnit{
        Name:       "Test Business",
        AdminPhone: "+972501234567",
    }

    // First insert succeeds
    err := repo.CreateWithDuplicateCheck(ctx, bu1)
    assert.NoError(t, err)

    // Second insert should fail (transaction rolls back)
    bu2 := &model.BusinessUnit{
        Name:       "Another Business",
        AdminPhone: "+972501234567", // Same phone
    }

    err = repo.CreateWithDuplicateCheck(ctx, bu2)
    assert.Error(t, err)
    assert.True(t, errors.Is(err, businessunitserrors.ErrAlreadyExists))

    // Verify only one exists
    all, _ := repo.FindByAdminPhone(ctx, "+972501234567")
    assert.Len(t, all, 1)
}
```

### Test 3: Multi-Collection Rollback

```go
func TestCreateWithSchedule_RollbackOnError(t *testing.T) {
    repo := setupTestRepository(t)
    ctx := context.Background()

    bu := &model.BusinessUnit{
        Name:       "Test Business",
        AdminPhone: "+972501234567",
    }

    // Schedule with invalid data (will fail)
    schedule := &model.Schedule{
        City: "", // Invalid: empty city
    }

    err := repo.CreateWithSchedule(ctx, bu, schedule)
    assert.Error(t, err)

    // Verify business unit was NOT created (transaction rolled back)
    all, _ := repo.FindByAdminPhone(ctx, "+972501234567")
    assert.Len(t, all, 0)
}
```

---

## 🚀 Implementation Checklist

### Phase 1: Setup
- [ ] Verify MongoDB is replica set (not standalone)
- [ ] Test transaction infrastructure works
- [ ] Add error types (ErrAlreadyExists, etc.)

### Phase 2: Identify Use Cases
- [ ] List all multi-collection operations
- [ ] Identify race condition risks
- [ ] Document cascade requirements

### Phase 3: Implement
- [ ] Add transaction methods to repository
- [ ] Update repository interface
- [ ] Update service layer to use transactions
- [ ] Add comprehensive tests

### Phase 4: Monitor
- [ ] Add transaction logging
- [ ] Monitor transaction duration
- [ ] Track rollback rates
- [ ] Alert on excessive retries

---

## 📊 Performance Considerations

### Transaction Overhead
- ~30% slower than non-transactional operations
- Only use when consistency is critical

### Timeout
- Default: 60 seconds
- Keep transactions short (<1 second ideal)
- Avoid network calls inside transactions

### Retry Logic
- MongoDB auto-retries on transient errors
- Max 120 seconds of total retry time
- Don't implement your own retry on top

---

## 🔧 Common Patterns

### Pattern 1: Check + Create
```go
// Check if exists, create if not (atomic)
ExecuteTransaction(ctx, func(sessCtx) {
    if exists { return ErrAlreadyExists }
    create()
})
```

### Pattern 2: Update + Cascade
```go
// Update main record + update related records
ExecuteTransaction(ctx, func(sessCtx) {
    updateMain()
    updateRelated()
})
```

### Pattern 3: Conditional Update
```go
// Update only if condition met
ExecuteTransaction(ctx, func(sessCtx) {
    record := find()
    if !condition(record) { return ErrConditionFailed }
    update(record)
})
```

---

## 🎯 Next Steps

1. **Verify MongoDB Replica Set**
   ```bash
   kubectl exec -it -n mongo <pod> -- mongosh --eval "rs.status()"
   ```

2. **Start with Simple Transaction**
   - Implement `CreateWithDuplicateCheck`
   - Test thoroughly
   - Monitor performance

3. **Add More Complex Transactions**
   - Multi-collection operations
   - Cascade updates
   - Conditional logic

4. **Monitor and Tune**
   - Track transaction duration
   - Optimize slow operations
   - Adjust timeouts if needed

---

## 📚 Resources

- [MongoDB Transactions Docs](https://www.mongodb.com/docs/manual/core/transactions/)
- [Go Driver Transactions](https://www.mongodb.com/docs/drivers/go/current/fundamentals/transactions/)
- Your existing docs: `internal/businessunits/repository/transactions.go`
