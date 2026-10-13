package repository

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
)

type Transactional interface {
	InsideTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Order interface {
	Transactional
	Create(ctx context.Context, order entity.Order) error
	GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error)
	Update(ctx context.Context, order entity.Order) error
	Delete(ctx context.Context, guid uuid.UUID) error
	List(ctx context.Context, status *string, userGUID *uuid.UUID) ([]entity.Order, error)
}
