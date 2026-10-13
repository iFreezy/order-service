package porder

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/entity"
	"github.com/iFreezy/order-service/internal/app/repository"
	rcpostgres "github.com/iFreezy/order-service/internal/app/repository/conn/postgres"
	"gorm.io/gorm"
)

type repoPg struct{ conn *rcpostgres.Client }

func NewRepo(conn *rcpostgres.Client) repository.Order { return &repoPg{conn: conn} }
func (r *repoPg) InsideTx(ctx context.Context, fn func(context.Context) error) error {
	return r.conn.InsideTx(ctx, fn)
}
func (r *repoPg) Create(ctx context.Context, order entity.Order) error {
	return r.conn.GetDB(ctx).WithContext(ctx).Create(&order).Error
}
func (r *repoPg) GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error) {
	var order entity.Order
	err := r.conn.GetDB(ctx).WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Where("guid = ?", guid).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return order, entity.ErrNotFound
	}
	return order, err
}
func (r *repoPg) Update(ctx context.Context, order entity.Order) error {
	result := r.conn.GetDB(ctx).WithContext(ctx).Model(&entity.Order{}).Where("guid = ?", order.GUID).Updates(map[string]any{"status": order.Status, "updated_at": order.UpdatedAt})
	return affectedError(result)
}
func (r *repoPg) Delete(ctx context.Context, guid uuid.UUID) error {
	return affectedError(r.conn.GetDB(ctx).WithContext(ctx).Where("guid = ?", guid).Delete(&entity.Order{}))
}
func affectedError(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrNotFound
	}
	return nil
}
func (r *repoPg) List(ctx context.Context, status *string, userGUID *uuid.UUID) ([]entity.Order, error) {
	orders := make([]entity.Order, 0)
	db := r.conn.GetDB(ctx).WithContext(ctx)
	if status != nil {
		db = db.Where("status = ?", *status)
	}
	if userGUID != nil {
		db = db.Where("user_guid = ?", *userGUID)
	}
	err := db.Order("id ASC").Find(&orders).Error
	return orders, err
}
