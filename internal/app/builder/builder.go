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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type Builder struct {
	cCtx     *cli.Context
	ctx      context.Context
	cancel   context.CancelCauseFunc
	wg       sync.WaitGroup
	err      error
	cfg      config.Config
	chErrors chan error
	signals  chan os.Signal

	connPostgres *rcpostgres.Client

	healthHandler rhandler.Health

	processors []processor.Processor
}

func NewBuilder(cCtx *cli.Context) *Builder {
	ctx, cancel := context.WithCancelCause(cCtx.Context)
	b := &Builder{
		cCtx:          cCtx,
		cancel:        cancel,
		chErrors:      make(chan error, 4096),
		signals:       make(chan os.Signal, 1),
		healthHandler: rhealth.NewHandler(),
	}
	b.ctx = processor.WithFailureHandler(ctx, func(err error) {
		select {
		case b.chErrors <- err:
		default:
		}
		cancel(err)
	})
	signal.Notify(b.signals, os.Interrupt, syscall.SIGTERM)
	b.wg.Add(2)
	go b.waitForSignal()
	go b.printErrors()
	return b
}
func (b *Builder) exec(fn func(*Builder), deps ...any) {
	if err := b.ctx.Err(); err != nil {
		b.err = err
		return
	}
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
		b.err = config.Load(config.LoadArgs{Output: b.cCtx.App.Writer, EnableSimpleLog: b.cCtx.Bool("no-json")})
		if b.err == nil {
			b.cfg = config.Root
		}
	})
}
func (b *Builder) BuildRepoConnPostgres() {
	b.exec(func(b *Builder) {
		b.connPostgres, b.err = rcpostgres.NewClient(b.ctx, b.cfg.Repository.Postgres)
	}, b.cfg)
}
func (b *Builder) BuildProcHttp() {
	b.exec(func(b *Builder) {
		b.processors = append(b.processors, rprocessor.NewHTTP(b.healthHandler, b.cfg.Processor.WebServer))
	}, b.cfg, b.healthHandler)
}
func (b *Builder) Run() error {
	defer b.stop()
	if err := b.ctx.Err(); err != nil {
		log.Info().Msg("Shutdown during initialization")
		return err
	}
	if b.err != nil {
		log.WithLevel(zerolog.FatalLevel).Err(b.err).Msg("Failed to initialize application")
		return b.err
	}
	log.Info().Msg("Application initialized")
	for _, proc := range b.processors {
		if b.ctx.Err() != nil {
			break
		}
		proc.StartAsync(b.ctx, &b.wg)
	}
	b.wg.Wait()
	log.Info().Msg("Application completed")
	if cause := context.Cause(b.ctx); !errors.Is(cause, context.Canceled) {
		return cause
	}
	return nil
}

func (b *Builder) waitForSignal() {
	defer b.wg.Done()
	select {
	case sig := <-b.signals:
		log.Info().Str("signal", sig.String()).Msg("Shutdown is requested")
		b.cancel(nil)
	case <-b.ctx.Done():
	}
}

func (b *Builder) printErrors() {
	defer b.wg.Done()
	for {
		select {
		case err := <-b.chErrors:
			log.Error().Err(err).Msg("Asynchronous error occurred")
		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Builder) stop() {
	b.cancel(nil)
	signal.Stop(b.signals)
	b.wg.Wait()
	if b.connPostgres != nil {
		if err := b.connPostgres.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database")
		}
	}
}
