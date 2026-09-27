package builder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/iFreezy/order-service/internal/app/config"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
	rhealth "github.com/iFreezy/order-service/internal/app/handler/http/health"
	"github.com/iFreezy/order-service/internal/app/processor"
	rprocessor "github.com/iFreezy/order-service/internal/app/processor/http"
	rcpostgres "github.com/iFreezy/order-service/internal/app/repository/conn/postgres"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type Builder struct {
	cli *cli.Context
	cfg *config.Config
	err error

	connPostgres *rcpostgres.Client

	healthHandler rhandler.Health

	processors []processor.Processor
}

func NewBuilder(ctx *cli.Context) *Builder { return &Builder{cli: ctx} }
func (b *Builder) exec(fn func(*Builder), deps ...any) {
	if b.err != nil {
		return
	}
	for _, dep := range deps {
		if dep == nil || (reflect.ValueOf(dep).Kind() == reflect.Ptr && reflect.ValueOf(dep).IsNil()) {
			b.err = fmt.Errorf("required dependency %T is empty", dep)
			return
		}
	}
	fn(b)
}
func (b *Builder) BuildConfig() {
	b.exec(func(b *Builder) {
		b.err = config.Load(config.LoadArgs{Output: b.cli.App.Writer, EnableSimpleLog: b.cli.Bool("no-json")})
		b.cfg = &config.Root
	})
}
func (b *Builder) BuildRepoConnPostgres() {
	b.exec(func(b *Builder) {
		b.connPostgres, b.err = rcpostgres.NewClient(b.cli.Context, b.cfg.Repository.Postgres)
	}, b.cfg)
}
func (b *Builder) BuildProcHttp() {
	b.exec(func(b *Builder) {
		b.healthHandler = rhealth.NewHandler()
		b.processors = append(b.processors, rprocessor.NewHTTP(b.healthHandler, b.cfg.Processor.WebServer))
	}, b.cfg, b.connPostgres)
}
func (b *Builder) Run() error {
	if b.connPostgres != nil {
		defer func() {
			if err := b.connPostgres.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close database")
			}
		}()
	}
	if b.err != nil {
		log.Error().Err(b.err).Msg("Failed to initialize application")
		return b.err
	}
	ctx, cancel := context.WithCancelCause(b.cli.Context)
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	ctx = processor.WithFailureHandler(ctx, cancel)
	var wg sync.WaitGroup
	log.Info().Msg("Application initialized")
	for _, proc := range b.processors {
		proc.StartAsync(ctx, &wg)
	}
	select {
	case sig := <-signals:
		log.Info().Str("signal", sig.String()).Msg("Shutdown is requested")
		cancel(nil)
	case <-ctx.Done():
		log.Info().Msg("Shutdown is requested")
	}
	wg.Wait()
	log.Info().Msg("Application completed")
	if cause := context.Cause(ctx); !errors.Is(cause, context.Canceled) {
		return cause
	}
	return nil
}
