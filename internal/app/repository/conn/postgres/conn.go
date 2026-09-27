package rcpostgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/iFreezy/order-service/internal/app/config/section"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	maxOpenConns = 10

	pingTimeout = 2 * time.Second
)

type Client struct {
	db  *gorm.DB
	cfg section.RepositoryPostgres
}

func (c *Client) DB() *gorm.DB {
	return c.db
}

func (c *Client) GetDB(ctx context.Context) *gorm.DB {
	if tx := getTxFromCtx(ctx); tx != nil {
		return tx
	}
	return c.db
}

func (c *Client) InsideTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if getTxFromCtx(ctx) != nil {
		return fn(ctx)
	}
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctxWithTx(ctx, tx))
	})
}

func NewClient(ctx context.Context, cfg section.RepositoryPostgres) (*Client, error) {
	dsn := url.URL{
		Scheme:   "postgres",
		Host:     cfg.Address,
		User:     url.UserPassword(cfg.Username, cfg.Password),
		Path:     cfg.Name,
		RawQuery: url.Values{"sslmode": []string{"disable"}}.Encode(),
	}

	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err = sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Client{db: db, cfg: cfg}, nil
}

func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	if err = sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close postgres connection: %w", err)
	}

	return nil
}
