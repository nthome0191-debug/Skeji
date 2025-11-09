package mongo

import (
	"context"
	"fmt"
	apperrors "skeji/pkg/errors"

	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionFunc func(ctx mongo.SessionContext) error

type TransactionManager interface {
	ExecuteTransaction(ctx context.Context, fn TransactionFunc) error
}

type mongoTransactionManager struct {
	client *mongo.Client
}

func NewTransactionManager(client *mongo.Client) TransactionManager {
	return &mongoTransactionManager{
		client: client,
	}
}

func (m *mongoTransactionManager) ExecuteTransaction(ctx context.Context, fn TransactionFunc) error {
	session, err := m.client.StartSession()
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
