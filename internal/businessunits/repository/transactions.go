package repository

import (
	"context"
	mongotx "skeji/pkg/db/mongo"
)

func (r *mongoBusinessUnitRepository) ExecuteTransaction(ctx context.Context, fn mongotx.TransactionFunc) error {
	return r.txManager.ExecuteTransaction(ctx, fn)
}
