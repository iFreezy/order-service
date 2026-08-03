package rprocessor

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iFreezy/order-service/internal/app/config/section"
	rhandler "github.com/iFreezy/order-service/internal/app/handler/http"
)

// readHeaderTimeout bounds how long the server waits for request headers,
// protecting it from Slowloris-style connection exhaustion.
const readHeaderTimeout = 10 * time.Second

type httpProc struct {
	server http.Server
	addr   string
}

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) *httpProc {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	// Recovery turns a panic in a handler into a 500 instead of a dead process.
	router.Use(gin.Recovery())
	router.NoRoute(handleNotFound)

	vGenericRegHealthCheck(router, hHealth)

	for _, route := range router.Routes() {
		log.Printf("Route registered: %s %s", route.Method, route.Path)
	}

	addr := fmt.Sprintf(":%d", cfg.ListenPort)

	return &httpProc{
		server: http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
		},
		addr: addr,
	}
}

func (p *httpProc) Serve() error {
	log.Printf("Starting HTTP server on %s", p.addr)

	return p.server.ListenAndServe()
}
