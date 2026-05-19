package data

import "context"

type TransactionManager interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type NoopTransactionManager struct{}

func (NoopTransactionManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }
