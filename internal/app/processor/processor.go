package processor

import (
	"context"
	"sync"
)

type Processor interface {
	StartAsync(ctx context.Context, wg *sync.WaitGroup)
}
type ProcessorFunc func(context.Context, *sync.WaitGroup)

func (f ProcessorFunc) StartAsync(ctx context.Context, wg *sync.WaitGroup) { f(ctx, wg) }
