package rcpostgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/iFreezy/order-service/internal/app/config/section"
)

func TestTransactionsPostgres(t *testing.T) {
	address := os.Getenv("ORDER_TEST_POSTGRES_ADDRESS")
	if address == "" {
		t.Skip("set ORDER_TEST_POSTGRES_ADDRESS for disposable PostgreSQL")
	}
	client, err := NewClient(t.Context(), section.RepositoryPostgres{Address: address, Username: "order_test", Password: "order_test", Name: "order_test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := client.DB().WithContext(t.Context()).Exec("CREATE TABLE task3003_tx_probe (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.DB().WithContext(context.Background()).Exec("DROP TABLE task3003_tx_probe").Error; err != nil {
			t.Error(err)
		}
	})
	if client.GetDB(t.Context()) != client.DB() {
		t.Fatal("nontransactional context must use the pool")
	}
	insert := func(ctx context.Context, id int) error {
		return client.GetDB(ctx).WithContext(ctx).Exec("INSERT INTO task3003_tx_probe(id) VALUES (?)", id).Error
	}
	count := func(want int64) {
		t.Helper()
		var got int64
		if err := client.DB().WithContext(t.Context()).Table("task3003_tx_probe").Count(&got).Error; err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %d rows, want %d", got, want)
		}
	}
	err = client.InsideTx(t.Context(), func(ctx context.Context) error {
		if getTxFromCtx(ctx) == nil || client.GetDB(ctx) == client.DB() {
			t.Fatal("transaction missing from context")
		}
		return client.InsideTx(ctx, func(nested context.Context) error {
			if getTxFromCtx(nested) != getTxFromCtx(ctx) {
				t.Fatal("nested call must reuse transaction")
			}
			return insert(nested, 1)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	count(1)
	sentinel := errors.New("rollback requested")
	err = client.InsideTx(t.Context(), func(ctx context.Context) error {
		if err := insert(ctx, 2); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error lost: %v", err)
	}
	count(1)
	func() {
		defer func() {
			if recover() != sentinel {
				t.Fatal("panic was not propagated")
			}
		}()
		_ = client.InsideTx(t.Context(), func(ctx context.Context) error {
			if err := insert(ctx, 3); err != nil {
				t.Fatal(err)
			}
			panic(sentinel)
		})
	}()
	count(1)
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := client.InsideTx(canceled, func(context.Context) error { t.Fatal("callback should not run with a canceled context"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction: %v", err)
	}
}
