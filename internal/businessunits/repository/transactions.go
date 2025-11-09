package repository

import (
	"context"
	"fmt"
	apperrors "skeji/pkg/errors"

	"go.mongodb.org/mongo-driver/mongo"
)

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

	if err != nil {
		if apperrors.IsAppError(err) {
			return err
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}
