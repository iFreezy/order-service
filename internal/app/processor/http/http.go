package rprocessor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iFreezy/order-service/internal/app/config/section"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
	"github.com/iFreezy/order-service/internal/app/processor"
	"github.com/iFreezy/order-service/internal/app/util"
	"github.com/iFreezy/order-service/internal/pkg/http/httph"
	"github.com/iFreezy/order-service/internal/pkg/http/mzerolog"
	"github.com/rs/zerolog/log"
)

const (
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 5 * time.Second
)

type httpProc struct {
	addr   string
	server http.Server
}

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) processor.Processor {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(adaptRequestMiddleware(httph.NewErrorMiddleware()), mzerolog.NewMiddleware(mzerolog.WithSkipper(util.IsFilteredHttpRoute)), gin.Recovery())
	router.NoRoute(handleNotFound)
	vGenericRegHealthCheck(router, hHealth)
	for _, route := range router.Routes() {
		log.Info().Str("method", route.Method).Str("path", route.Path).Msg("Route registered")
	}
	addr := fmt.Sprintf(":%d", cfg.ListenPort)
	return &httpProc{addr: addr, server: http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: readHeaderTimeout}}
}

func adaptRequestMiddleware(m httph.Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		m(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}
func (p *httpProc) StartAsync(ctx context.Context, wg *sync.WaitGroup) {
	addr := p.addr
	if addr == "" {
		addr = p.server.Addr
	}
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Error().Err(err).Str("listen_addr", addr).Msg("Failed to start listening TCP addr for HTTP server")
		processor.ReportFailure(ctx, err)
		return
	}
	p.addr = l.Addr().String()
	log.Info().Str("listen_addr", p.addr).Msg("Listening of TCP addr for HTTP server has been started")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			log.Error().Err(err).Msg("HTTP server failed")
			processor.ReportFailure(ctx, err)
		}
	}()
	processor.WatchForShutdown(ctx, wg, processor.CloserFunc(l.Close))
	processor.WatchForShutdown(ctx, wg, processor.NewCloserContextFunc(shutdownTimeout, func(shutdownCtx context.Context) error {
		if err := p.server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("HTTP shutdown failed")
			if closeErr := p.server.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("HTTP close failed")
			}
			return err
		}
		return nil
	}))
}

func (p *httpProc) serve(l net.Listener) error { return p.server.Serve(l) }
