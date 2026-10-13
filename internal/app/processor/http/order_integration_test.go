package rprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/iFreezy/order-service/internal/app/config/section"
	"github.com/iFreezy/order-service/internal/app/entity"
	rhealth "github.com/iFreezy/order-service/internal/app/handler/http/health"
	horder "github.com/iFreezy/order-service/internal/app/handler/http/order"
	rcpostgres "github.com/iFreezy/order-service/internal/app/repository/conn/postgres"
	porder "github.com/iFreezy/order-service/internal/app/repository/order"
	sorder "github.com/iFreezy/order-service/internal/app/service/order"
)

func TestOrderCRUDPostgres(t *testing.T) {
	address := os.Getenv("ORDER_TEST_POSTGRES_ADDRESS")
	if address == "" {
		t.Skip("set ORDER_TEST_POSTGRES_ADDRESS for disposable PostgreSQL integration tests")
	}
	conn, err := rcpostgres.NewClient(t.Context(), section.RepositoryPostgres{Address: address, Username: "order_test", Password: "order_test", Name: "order_test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	repo := porder.NewRepo(conn)
	proc := NewHTTP(rhealth.NewHandler(), horder.NewHandler(sorder.NewService(repo)), section.ProcessorWebServer{}).(*httpProc)
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		proc.server.Handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	user := uuid.Must(uuid.NewV4())
	body := `{"user_guid":"` + user.String() + `","currency":"USD","items":[{"product_guid":"aaaa0000-aaaa-aaaa-aaaa-aaaaaaaaaaaa","quantity":2,"unit_price":1500},{"product_guid":"bbbb0000-bbbb-bbbb-bbbb-bbbbbbbbbbbb","quantity":1,"unit_price":3000}]}`
	created := request(http.MethodPost, "/v1/order/create", body, http.StatusCreated)
	var order entity.ResponseOrderGet
	if err := json.Unmarshal(created.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repo.Delete(context.Background(), order.GUID); err != nil && !errors.Is(err, entity.ErrNotFound) {
			t.Error(err)
		}
	})
	if order.TotalPrice != 6000 || len(order.Items) != 2 || order.Status != "pending" || !order.UpdatedAt.IsZero() {
		t.Fatalf("unexpected create: %s", created.Body)
	}
	path := "/v1/order/" + order.GUID.String()
	got := request(http.MethodGet, path, "", http.StatusOK)
	var loaded entity.ResponseOrderGet
	if err := json.Unmarshal(got.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Items) != 2 || loaded.UpdatedAt.IsZero() {
		t.Fatalf("missing preload/timestamp: %s", got.Body)
	}
	updated := request(http.MethodPatch, path, `{"status":"confirmed"}`, http.StatusOK)
	if err := json.Unmarshal(updated.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "confirmed" || !loaded.UpdatedAt.After(order.CreatedAt) || len(loaded.Items) != 2 {
		t.Fatalf("unexpected update: %s", updated.Body)
	}
	for _, filter := range []string{"", `{"status":"confirmed"}`, `{"user_guid":"` + user.String() + `"}`} {
		list := request(http.MethodPost, "/v1/order/list", filter, http.StatusOK)
		var result entity.ResponseOrderList
		if err := json.Unmarshal(list.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range result.Data {
			if item.GUID == order.GUID {
				found = true
			}
		}
		if !found {
			t.Fatalf("created order absent for filter %s: %s", filter, list.Body)
		}
	}
	empty := request(http.MethodPost, "/v1/order/list", `{"status":"pending","user_guid":"`+user.String()+`"}`, http.StatusOK)
	if strings.TrimSpace(empty.Body.String()) != `{"data":[]}` {
		t.Fatalf("empty list: %s", empty.Body)
	}
	for _, invalid := range []string{`{"currency":"US","items":[]}`, `{"currency":"USD","items":[{"product_guid":"00000000-0000-0000-0000-000000000000","quantity":1,"unit_price":1}]}`, strings.Replace(body, "\"quantity\":2", "\"quantity\":0", 1), strings.Replace(body, "\"unit_price\":1500", "\"unit_price\":9223372036854775807", 1), "{", "null"} {
		request(http.MethodPost, "/v1/order/create", invalid, http.StatusBadRequest)
	}
	request(http.MethodGet, "/v1/order/not-a-uuid", "", http.StatusBadRequest)
	request(http.MethodPatch, path, `{}`, http.StatusBadRequest)
	request(http.MethodPost, "/v1/order/list", `{"user_guid":"invalid"}`, http.StatusBadRequest)
	deleted := request(http.MethodDelete, path, "", http.StatusOK)
	if deleted.Body.Len() != 0 {
		t.Fatal("delete must have an empty body")
	}
	var count int64
	if err := conn.DB().Model(&entity.OrderItem{}).Where("order_guid = ?", order.GUID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("cascade: count=%d err=%v", count, err)
	}
	request(http.MethodGet, path, "", http.StatusNotFound)
	request(http.MethodDelete, path, "", http.StatusNotFound)
	request(http.MethodPatch, path, `{"status":"confirmed"}`, http.StatusNotFound)
	request(http.MethodGet, "/health", "", http.StatusOK)
	request(http.MethodGet, "/unknown", "", http.StatusNotFound)

	bad := entity.Order{GUID: uuid.Must(uuid.NewV4()), Currency: "USD", Status: "pending", Items: []entity.OrderItem{{GUID: uuid.Must(uuid.NewV4()), Quantity: 1 << 32, UnitPrice: 1}}}
	if err := repo.Create(t.Context(), bad); err == nil {
		t.Fatal("expected failing item insert")
	}
	if _, err := repo.GetByGUID(t.Context(), bad.GUID); !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("parent not rolled back: %v", err)
	}
	sentinel := errors.New("rollback")
	err = repo.InsideTx(t.Context(), func(ctx context.Context) error {
		return repo.InsideTx(ctx, func(nested context.Context) error {
			if conn.GetDB(nested) != conn.GetDB(ctx) {
				t.Fatal("nested transaction changed connection")
			}
			if err := conn.GetDB(nested).WithContext(nested).Create(&entity.Order{GUID: bad.GUID, Currency: "USD", Status: "pending"}).Error; err != nil {
				return err
			}
			return sentinel
		})
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := repo.GetByGUID(t.Context(), bad.GUID); !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("transaction not rolled back: %v", err)
	}
}
