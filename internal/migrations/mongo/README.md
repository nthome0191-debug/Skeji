# 🧭 MongoDB Migration & Schema Evolution Guide

This document explains how Skeji manages **MongoDB schema validation, index setup, and forward-safe data evolution** in both local and production environments.

MongoDB in Skeji is **schema-validated** using `$jsonSchema` validators, but remains flexible enough for incremental evolution without downtime.

---

## ⚙️ Overview

Each Skeji collection is defined with:

- **Indexes** — declared in [`migrate.go`](./migrate.go)
- **Validators** — JSON schemas in [`validators/`](./validators)
- **Migrations** — applied via [`RunMigration()`](./migrate.go)

Migrations are:
- **Idempotent** – safe to re-run multiple times  
- **Non-destructive** – never drop data  
- **Version-tracked** – logged in the `_migrations` collection  
- **Automatic** – executed via `make migrate` or CI/CD

---

## 🧩 Folder structure

```
internal/migrations/mongo/
├── migrate.go                 # Main migration logic
├── validators/
│   ├── business_unit.go
│   ├── schedule.go
│   └── booking.go
└── README.md                  # (this file)
```

---

## 🚀 Running Migrations

To apply or verify all schemas and indexes locally:

```bash
make migrate
```

This will:
1. Ensure all collections exist (`Business_units`, `Schedules`, `Bookings`)
2. Apply `$jsonSchema` validators
3. Apply all indexes
4. Log the run in `_migrations`

Safe to run repeatedly.

---

## 🧠 Adding a New Field to an Existing Collection (Zero Downtime)

When Skeji evolves and a new field must be added (for example, `payment_status` in `Bookings`),  
follow this **three-step safe process**.

### Step 1️⃣ – Update the Validator (Field Optional)

In `validators/booking.go`, add the new field to `properties`, **but not to `required`:**

```go
"payment_status": bson.M{"bsonType": "string"},
```

Then re-run migration:

```bash
make migrate
```

✅ This updates the collection schema to *allow* the new field without breaking existing data.

---

### Step 2️⃣ – Backfill Existing Documents

Create a short Go script under `cmd/backfill/` (or use the helper below):

```go
filter := bson.M{"payment_status": bson.M{"$exists": false}}
update := bson.M{"$set": bson.M{"payment_status": "unpaid"}}

result, err := db.Collection("Bookings").UpdateMany(ctx, filter, update)
if err != nil {
    log.Fatalf("backfill failed: %v", err)
}
fmt.Printf("✅ Backfilled %d existing documents\n", result.ModifiedCount)
```

Run it locally or as a one-off CI job.

✅ All existing rows now include the new field.

---

### Step 3️⃣ – Enforce as Required (After Backfill)

Once every document includes the new field,  
add it to the `required` array in the validator:

```go
"required": [..., "payment_status"]
```

Then run migration again:

```bash
make migrate
```

✅ The validator is now strict — future inserts/updates must include this field.

---

## 🧰 (Optional) Helper for Automated Backfills

You can use a common helper for backfilling new fields:

```go
func BackfillField(ctx context.Context, db *mongo.Database, collName, field string, defaultValue any) error {
    filter := bson.M{field: bson.M{"$exists": false}}
    update := bson.M{"$set": bson.M{field: defaultValue}}
    result, err := db.Collection(collName).UpdateMany(ctx, filter, update)
    if err != nil {
        return fmt.Errorf("failed to backfill %s.%s: %w", collName, field, err)
    }
    fmt.Printf("Backfilled %d docs with %s=%v\n", result.ModifiedCount, field, defaultValue)
    return nil
}
```

Usage example:
```go
BackfillField(ctx, db, "Bookings", "payment_status", "unpaid")
```

---

## 🧱 Adding a New Collection

1. Create a new validator file in `validators/`
   ```bash
   touch internal/migrations/mongo/validators/notifications.go
   ```
2. Define its schema (JSON schema with required + optional fields)
3. Add its indexes and validator to `RunMigration()` in `migrate.go`
4. Run:
   ```bash
   make migrate
   ```

---

## 🧩 Versioning (optional but recommended)

If you plan frequent schema updates:
- Create folders like `v1/`, `v2/` under `internal/migrations/mongo/`
- Log migration version numbers in `_migrations`
- CI/CD can then apply only newer migrations automatically

---

## 🧠 Key Principles

| Rule | Why |
|------|-----|
| **Never make new fields required immediately** | Prevents validator rejections for existing data |
| **Backfill with defaults before enforcing** | Keeps data consistent |
| **Always re-run migrations after schema change** | Ensures validators and indexes are synced |
| **Never drop or rename collections** | Avoids production data loss |
| **All migrations must be idempotent** | Safe for retries & CI/CD pipelines |

---

## 🧰 Quick Commands

| Task | Command |
|------|----------|
| Run all migrations | `make migrate` |
| Drop local kind cluster | `make kind-down` |
| Spin up local environment | `make local-up` |
| Verify Mongo collections | `kubectl exec -it -n mongo <pod> -- mongo skeji --eval 'db.getCollectionNames()'` |
| Check validator | `db.getCollectionInfos({name: "Bookings"})[0].options.validator` |

---

## 🧩 Example Migration Lifecycle

| Action | Description |
|--------|--------------|
| Add a new feature field | Update validator, make it optional |
| Backfill old docs | Script or helper to set default value |
| Enforce field required | Update validator, re-run migration |
| Add a new service | Create new collection + validator + indexes |
| Rename field | Add new field, backfill, deprecate old, remove in next version |

---

## 👩‍💻 Maintainers Notes

- Always test migrations on **staging** with a copy of production data before rollout.
- Avoid using `$unset` in live environments.
- Review schema diffs in code review — especially `required` and index definitions.
- The `_migrations` collection acts as an audit trail.

---

## 🧾 Example `_migrations` entry

```json
{
  "_id": "671e6c7a5d1234",
  "collection": "Bookings",
  "applied_at": "2025-10-27T14:52:12Z"
}
```

---

### 🧩 Summary

Skeji migrations are:
- **Idempotent**
- **Audited**
- **Safe for production**
- **Schema-validated**

Always evolve schema in **two phases**:  
_add → backfill → enforce_.
