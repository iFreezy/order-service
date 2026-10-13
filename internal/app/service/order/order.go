package sorder

import (
	"context"
	"math"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
	"github.com/iFreezy/order-service/internal/app/repository"
	"github.com/iFreezy/order-service/internal/app/service"
)

type srv struct{ repoOrder repository.Order }

func NewService(repo repository.Order) service.Order { return &srv{repoOrder: repo} }
func (s *srv) Create(ctx context.Context, req entity.RequestOrderCreate) (entity.Order, error) {
	now := time.Now()
	guid, err := uuid.NewV4()
	if err != nil {
		return entity.Order{}, err
	}
	order := entity.Order{GUID: guid, UserGUID: req.UserGUID, Currency: req.Currency, Status: "pending", CreatedAt: now, UpdatedAt: now, Items: make([]entity.OrderItem, 0, len(req.Items))}
	for _, item := range req.Items {
		if item.Quantity <= 0 || item.Quantity > math.MaxInt32 || item.UnitPrice <= 0 || item.UnitPrice > (math.MaxInt64-order.TotalPrice)/int64(item.Quantity) {
			return entity.Order{}, entity.ErrIncorrectParameters
		}
		itemGUID, genErr := uuid.NewV4()
		if genErr != nil {
			return entity.Order{}, genErr
		}
		order.TotalPrice += int64(item.Quantity) * item.UnitPrice
		order.Items = append(order.Items, entity.OrderItem{GUID: itemGUID, OrderGUID: guid, ProductGUID: item.ProductGUID, Quantity: item.Quantity, UnitPrice: item.UnitPrice, CreatedAt: now, UpdatedAt: now})
	}
	if err = s.repoOrder.Create(ctx, order); err != nil {
		return entity.Order{}, err
	}
	return order, nil
}
func (s *srv) GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error) {
	return s.repoOrder.GetByGUID(ctx, guid)
}
func (s *srv) Update(ctx context.Context, guid uuid.UUID, req entity.RequestOrderUpdate) (entity.Order, error) {
	var order entity.Order
	err := s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		var err error
		order, err = s.repoOrder.GetByGUID(txCtx, guid)
		if err != nil {
			return err
		}
		order.Status = req.Status
		order.UpdatedAt = time.Now()
		return s.repoOrder.Update(txCtx, order)
	})
	return order, err
}
func (s *srv) Delete(ctx context.Context, guid uuid.UUID) error {
	return s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repoOrder.GetByGUID(txCtx, guid); err != nil {
			return err
		}
		return s.repoOrder.Delete(txCtx, guid)
	})
}
func (s *srv) List(ctx context.Context, req entity.RequestOrderList) ([]entity.Order, error) {
	return s.repoOrder.List(ctx, req.Status, req.UserGUID)
}
