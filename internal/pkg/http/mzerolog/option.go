package mzerolog

import (
	"net/http"

	"github.com/rs/zerolog"
)

type Option func(*middleware)

func WithLogger(logger zerolog.Logger) Option {
	return func(m *middleware) { m.log = logger }
}

func WithSkipper(skipper func(r *http.Request) bool) Option {
	return func(m *middleware) { m.skipper = skipper }
}
