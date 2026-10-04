package repository

import "context"

type Transactional interface {
	InsideTx(ctx context.Context, fn func(ctx context.Context) error) error
}
