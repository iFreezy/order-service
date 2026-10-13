package mzerolog

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iFreezy/order-service/internal/pkg/http/httph"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type middleware struct {
	log     zerolog.Logger
	skipper func(r *http.Request) bool
}

func defaultSkipper(_ *http.Request) bool { return false }

func NewMiddleware(opts ...Option) gin.HandlerFunc {
	m := &middleware{log: log.Logger, skipper: defaultSkipper}
	for _, opt := range opts {
		opt(m)
	}
	return m.Callback
}

func (m *middleware) Callback(c *gin.Context) {
	start := time.Now()
	c.Next()
	err := httph.ErrorGet(c.Request)
	if m.skipper != nil && m.skipper(c.Request) {
		return
	}
	level := zerolog.DebugLevel
	var message strings.Builder
	message.Grow(48 + len(c.Request.RequestURI))
	message.WriteString(c.Request.Method)
	message.WriteByte(' ')
	message.WriteString(c.Request.RequestURI)
	if err != nil || c.Writer.Status() >= http.StatusInternalServerError {
		level = zerolog.ErrorLevel
		message.WriteString(" finished (or aborted) with error")
	} else {
		message.WriteString(" finished with no error")
	}
	m.log.WithLevel(level).Err(err).Ctx(c.Request.Context()).Dur("exec_time", time.Since(start)).Str("client_ip", c.ClientIP()).Int("status", c.Writer.Status()).Msg(message.String())
}
