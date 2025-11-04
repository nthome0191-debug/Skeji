package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// TransactionFunc defines a function that executes within a MongoDB transaction
type TransactionFunc func(ctx mongo.SessionContext) error

// ExecuteTransaction runs a function within a MongoDB transaction with automatic retry logic
//
// MongoDB transactions provide ACID guarantees for multi-document operations.
// Use transactions when you need to:
// - Update multiple collections atomically
// - Ensure consistency across related documents
// - Prevent partial updates on failure
//
// Current Implementation Status: Foundation Only
// This function provides the infrastructure for transactions but is not yet
// integrated into the service layer. Future work should:
// 1. Identify operations that require transactions (e.g., creating business + schedule)
// 2. Update service methods to use ExecuteTransaction
// 3. Handle transaction-specific errors appropriately
// 4. Add transaction monitoring and logging
//
// Example Usage (Future):
//
//	err := r.ExecuteTransaction(ctx, func(sessCtx mongo.SessionContext) error {
//	    if err := r.collection.InsertOne(sessCtx, doc1); err != nil {
//	        return err // Transaction will abort
//	    }
//	    if err := r.otherCollection.InsertOne(sessCtx, doc2); err != nil {
//	        return err // Transaction will abort
//	    }
//	    return nil // Transaction will commit
//	})
//
// Transaction Considerations:
// - Transactions have a 60-second timeout by default
// - Write conflicts will cause automatic retries
// - Transactions require MongoDB replica set (not supported in standalone)
// - Avoid long-running operations inside transactions
func (r *mongoBusinessUnitRepository) ExecuteTransaction(ctx context.Context, fn TransactionFunc) error {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// WithTransaction handles commit/abort and retries on transient errors
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// Transaction Strategy Documentation
//
// When to Use Transactions:
// -------------------------
// 1. Creating Business Unit with Related Entities
//    - Business unit + initial schedule setup
//    - Business unit + admin user record
//    Problem: If business unit creation succeeds but schedule fails, we have orphaned data
//
// 2. Updating Business Unit with Cascading Changes
//    - Business unit update + schedule location updates
//    - Business unit deletion + cleanup of related schedules/bookings
//    Problem: Partial updates could leave data in inconsistent state
//
// 3. Multi-Step Validation and Creation
//    - Check for duplicates + create new record
//    Problem: Race condition between check and insert
//
// When NOT to Use Transactions:
// ----------------------------
// 1. Single Document Operations
//    - Simple CRUD on one collection
//    Reason: MongoDB operations are atomic at document level
//
// 2. Read-Only Operations
//    - Queries and searches
//    Reason: No consistency concerns, transactions add overhead
//
// 3. Operations with External Services
//    - Calling external APIs within transaction
//    Reason: Transactions should be short-lived; external calls are unpredictable
//
// Current State: No Transactions
// ------------------------------
// As of now, all repository operations are single-document and do not use transactions.
// This is acceptable for the current MVP phase but should be revisited when:
// - Adding multi-collection operations
// - Implementing cascade deletes
// - Adding complex business workflows
//
// Performance Considerations:
// --------------------------
// - Transactions have ~30% overhead compared to non-transactional operations
// - Use only when consistency requirements justify the cost
// - Consider eventual consistency patterns for non-critical operations
//
// Testing Strategy:
// ----------------
// When implementing transaction support, ensure tests cover:
// 1. Happy path: successful transaction commit
// 2. Error path: transaction abort and rollback
// 3. Retry logic: transient errors and automatic retry
// 4. Timeout: transaction exceeding time limit
// 5. Concurrent access: multiple transactions competing for same data
