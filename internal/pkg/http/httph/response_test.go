package httph

import (
	"errors"
	"fmt"
	"github.com/iFreezy/order-service/internal/app/entity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleError(t *testing.T) {
	for _, tc := range []struct {
		err     error
		status  int
		message string
	}{
		{fmt.Errorf("repository details: %w", entity.ErrNotFound), http.StatusNotFound, "not found"},
		{errors.New("password=secret"), http.StatusInternalServerError, "internal server error"},
	} {
		r := ErrorPrepare(httptest.NewRequest(http.MethodGet, "/", nil))
		w := httptest.NewRecorder()
		HandleError(w, r, tc.err)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.message) || strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "repository details") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
		}
		if ErrorGet(r) != tc.err || ErrorGetStatusCode(r) != tc.status {
			t.Fatal("full error/status was not retained for logging")
		}
	}
}
