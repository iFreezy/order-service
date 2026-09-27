package processor

import (
	"context"
	"io"
	"sync"
	"time"
)

type failureKey struct{}

func WithFailureHandler(ctx context.Context, cancel context.CancelCauseFunc) context.Context {
	return context.WithValue(ctx, failureKey{}, cancel)
}
func ReportFailure(ctx context.Context, err error) {
	if cancel, ok := ctx.Value(failureKey{}).(context.CancelCauseFunc); ok {
		cancel(err)
	}
}

type CloserFunc func() error

func (f CloserFunc) Close() error { return f() }
func NewCloserContextFunc(timeout time.Duration, fn func(context.Context) error) CloserFunc {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return fn(ctx)
	}
}
func WatchForShutdown(ctx context.Context, wg *sync.WaitGroup, closer io.Closer) {
	Wrap(context.Background(), wg, func(context.Context) { <-ctx.Done(); _ = closer.Close() })
}
func Wrap(ctx context.Context, wg *sync.WaitGroup, fn func(context.Context)) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		select {
		case <-ctx.Done():
			return
		default:
			fn(ctx)
		}
	}()
}
