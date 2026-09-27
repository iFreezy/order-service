package rprocessor

import (
	"context"
	"errors"
	"fmt"
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

const readHeaderTimeout = 10 * time.Second

type httpProc struct{ server http.Server }

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) processor.Processor {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(adaptRequestMiddleware(httph.NewErrorMiddleware()), mzerolog.NewMiddleware(mzerolog.WithSkipper(util.IsFilteredHttpRoute)), gin.Recovery())
	router.NoRoute(handleNotFound)
	vGenericRegHealthCheck(router, hHealth)
	for _, route := range router.Routes() {
		log.Info().Str("method", route.Method).Str("path", route.Path).Msg("Route registered")
	}
	return &httpProc{server: http.Server{Addr: fmt.Sprintf(":%d", cfg.ListenPort), Handler: router, ReadHeaderTimeout: readHeaderTimeout}}
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
	serveCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer stop()
		defer close(done)
		log.Info().Str("listen_addr", p.server.Addr).Msg("Listening of TCP addr for HTTP server has been started")
		if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server failed")
			processor.ReportFailure(ctx, err)
		}
	}()
	closer := processor.NewCloserContextFunc(10*time.Second, func(shutdownCtx context.Context) error {
		if err := p.server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("HTTP shutdown failed")
			if closeErr := p.server.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("HTTP close failed")
			}
		}
		<-done
		return nil
	})
	processor.WatchForShutdown(serveCtx, wg, closer)
}
