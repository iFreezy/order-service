package sorder

import (
	"context"
	"math"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
	"github.com/iFreezy/order-service/internal/app/repository"
)

type recordingRepo struct {
	repository.Order
	created *entity.Order
}

func (r *recordingRepo) Create(_ context.Context, order entity.Order) error {
	r.created = &order
	return nil
}

func TestCreate(t *testing.T) {
	r := &recordingRepo{}
	req := entity.RequestOrderCreate{Currency: "USD", Items: []entity.RequestOrderItemCreate{
		{ProductGUID: uuid.Must(uuid.NewV4()), Quantity: 2, UnitPrice: 1500},
		{ProductGUID: uuid.Must(uuid.NewV4()), Quantity: 1, UnitPrice: 3000},
	}}
	order, err := NewService(r).Create(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if order.TotalPrice != 6000 || order.Status != "pending" || order.GUID == uuid.Nil || r.created == nil {
		t.Fatalf("unexpected order: %+v", order)
	}
	for _, item := range order.Items {
		if item.GUID == uuid.Nil || item.OrderGUID != order.GUID || !item.CreatedAt.Equal(order.CreatedAt) || !item.UpdatedAt.Equal(order.UpdatedAt) {
			t.Fatalf("inconsistent item: %+v", item)
		}
	}
	req.Items[0].UnitPrice = math.MaxInt64
	r.created = nil
	if _, err = NewService(r).Create(t.Context(), req); err == nil || r.created != nil {
		t.Fatal("overflow must be rejected before persistence")
	}
}
