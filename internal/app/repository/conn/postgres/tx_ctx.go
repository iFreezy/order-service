package rcpostgres

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

func getTxFromCtx(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(txKey{}).(*gorm.DB)
	return tx
}

func ctxWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
