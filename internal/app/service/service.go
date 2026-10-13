package service

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
)

type Order interface {
	Create(ctx context.Context, req entity.RequestOrderCreate) (entity.Order, error)
	GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error)
	Update(ctx context.Context, guid uuid.UUID, req entity.RequestOrderUpdate) (entity.Order, error)
	Delete(ctx context.Context, guid uuid.UUID) error
	List(ctx context.Context, req entity.RequestOrderList) ([]entity.Order, error)
}
