package process

import (
	"context"
	"sync"
)

var (
	mu  sync.RWMutex
	ctx = context.Background()
)

func SetContext(next context.Context) {
	mu.Lock()
	defer mu.Unlock()

	if next == nil {
		ctx = context.Background()
		return
	}
	ctx = next
}

func Context() context.Context {
	mu.RLock()
	defer mu.RUnlock()
	return ctx
}
