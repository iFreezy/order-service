package rprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iFreezy/order-service/internal/app/config/section"
	"github.com/iFreezy/order-service/internal/app/entity"
	rhealth "github.com/iFreezy/order-service/internal/app/handler/http/health"
	"github.com/iFreezy/order-service/internal/app/processor"
	"github.com/iFreezy/order-service/internal/pkg/http/httph"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestMiddlewareChain(t *testing.T) {
	var output bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&output).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = previous })
	proc := NewHTTP(rhealth.NewHandler(), section.ProcessorWebServer{}).(*httpProc)
	router := proc.server.Handler.(*gin.Engine)
	router.GET("/failure", func(c *gin.Context) {
		httph.HandleError(c.Writer, c.Request, fmt.Errorf("private details: %w", entity.ErrAlreadyExists))
	})
	router.GET("/panic", func(*gin.Context) { panic("test recovery") })
	output.Reset()
	for _, tc := range []struct {
		path   string
		status int
		body   string
	}{
		{"/health", http.StatusOK, "ok"},
		{"/missing", http.StatusNotFound, `{"error":"route not found"}`},
		{"/failure", http.StatusConflict, `{"error":"already exists"}`},
		{"/panic", http.StatusInternalServerError, ""},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.status || w.Body.String() != tc.body {
			t.Fatalf("%s: %d %q", tc.path, w.Code, w.Body.String())
		}
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("health should be filtered: %s", output.String())
	}
	for i, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatal(err)
		}
		wantLevel := "error"
		if i == 0 {
			wantLevel = "debug"
		}
		if fields["level"] != wantLevel || fields["exec_time"] == nil || fields["client_ip"] == nil {
			t.Fatalf("unexpected log: %s", line)
		}
	}
	if !strings.Contains(output.String(), "private details") {
		t.Fatal("original wrapped error was lost from logs")
	}
}

func TestStartAsyncReportsBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	ctx = processor.WithFailureHandler(ctx, cancel)
	proc := &httpProc{server: http.Server{Addr: listener.Addr().String(), ReadHeaderTimeout: time.Second}}
	var wg sync.WaitGroup
	proc.StartAsync(ctx, &wg)
	select {
	case <-ctx.Done():
		if errors.Is(context.Cause(ctx), context.Canceled) {
			t.Fatal("bind failure was not propagated")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startup failure left the application running")
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not finish")
	}
}
