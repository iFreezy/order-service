package builder

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/iFreezy/order-service/internal/app/processor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func newTestBuilder(t *testing.T, parent context.Context) (*Builder, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	app := &cli.App{Writer: &output}
	cCtx := cli.NewContext(app, flag.NewFlagSet("test", flag.ContinueOnError), nil)
	cCtx.Context = parent
	return NewBuilder(cCtx), &output
}

func TestNewBuilderInitializesLifecycleAndHealth(t *testing.T) {
	b, _ := newTestBuilder(t, t.Context())
	defer b.stop()
	if b.ctx == nil || b.chErrors == nil || cap(b.chErrors) != 4096 || b.healthHandler == nil {
		t.Fatal("builder lifecycle or health handler was not initialized")
	}
}

func TestBuildProcHTTPDoesNotRequirePostgres(t *testing.T) {
	b, _ := newTestBuilder(t, t.Context())
	defer b.stop()
	b.cfg.Processor.WebServer.ListenPort = 0
	b.BuildProcHttp()
	if b.err != nil || len(b.processors) != 1 {
		t.Fatalf("HTTP processor needs only config and health: err=%v processors=%d", b.err, len(b.processors))
	}
}

func TestShutdownDuringInitialization(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	b, _ := newTestBuilder(t, parent)
	cancel()
	var output bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&output)
	defer func() { log.Logger = previous }()
	b.BuildConfig()
	if !errors.Is(b.err, context.Canceled) {
		t.Fatalf("initialization continued after cancellation: %v", b.err)
	}
	if err := b.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v", err)
	}
	if !strings.Contains(output.String(), "Shutdown during initialization") {
		t.Fatalf("missing initialization shutdown log: %s", output.String())
	}
}

func TestFailedInitializationLogsFatalAndCleansUp(t *testing.T) {
	b, _ := newTestBuilder(t, t.Context())
	b.err = errors.New("postgres unavailable")
	var output bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&output)
	defer func() { log.Logger = previous }()
	if err := b.Run(); err == nil {
		t.Fatal("expected initialization failure")
	}
	if !strings.Contains(output.String(), `"level":"fatal"`) || !strings.Contains(output.String(), "Failed to initialize application") {
		t.Fatalf("fatal startup log missing: %s", output.String())
	}
}

func TestSignalStopsApplication(t *testing.T) {
	b, _ := newTestBuilder(t, t.Context())
	started := make(chan struct{})
	b.processors = append(b.processors, processor.ProcessorFunc(func(ctx context.Context, wg *sync.WaitGroup) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			close(started)
			<-ctx.Done()
		}()
	}))
	result := make(chan error, 1)
	go func() { result <- b.Run() }()
	<-started
	b.signals <- os.Signal(syscall.SIGTERM)
	if err := <-result; err != nil {
		t.Fatalf("signal shutdown failed: %v", err)
	}
}

func TestSignalDuringInitialization(t *testing.T) {
	b, _ := newTestBuilder(t, t.Context())
	b.signals <- os.Signal(syscall.SIGINT)
	<-b.ctx.Done()
	b.BuildConfig()
	if err := b.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled initialization, got %v", err)
	}
}

func TestBuildConfigAndPostgresFailure(t *testing.T) {
	t.Setenv("APP_MONITOR_LOG_LEVEL", "error")
	t.Setenv("APP_REPOSITORY_POSTGRES_ADDRESS", "127.0.0.1:1")
	t.Setenv("APP_REPOSITORY_POSTGRES_USERNAME", "order_test")
	t.Setenv("APP_REPOSITORY_POSTGRES_PASSWORD", "order_test")
	t.Setenv("APP_REPOSITORY_POSTGRES_NAME", "order_test")
	b, _ := newTestBuilder(t, t.Context())
	b.BuildConfig()
	if b.err != nil {
		t.Fatalf("config load failed: %v", b.err)
	}
	b.BuildRepoConnPostgres()
	if b.err == nil {
		t.Fatal("postgres connection unexpectedly succeeded")
	}
	if err := b.Run(); err == nil {
		t.Fatal("Run did not propagate startup failure")
	}
}
