package httph

import (
	"context"
	"net/http"
)

type errorKey struct{}
type errorState struct {
	err    error
	status int
}
type Middleware = func(http.Handler) http.Handler

func ErrorPrepare(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), errorKey{}, &errorState{}))
}
func ErrorApply(r *http.Request, err error) {
	if state, ok := r.Context().Value(errorKey{}).(*errorState); ok {
		state.err = err
	}
}
func ErrorGet(r *http.Request) error {
	if state, ok := r.Context().Value(errorKey{}).(*errorState); ok {
		return state.err
	}
	return nil
}
func ErrorApplyStatusCode(r *http.Request, status int) {
	if state, ok := r.Context().Value(errorKey{}).(*errorState); ok {
		state.status = status
	}
}
func ErrorGetStatusCode(r *http.Request) int {
	if state, ok := r.Context().Value(errorKey{}).(*errorState); ok {
		return state.status
	}
	return 0
}
func NewErrorMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, ErrorPrepare(r)) })
	}
}
