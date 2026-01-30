package pr2

import (
	"context"
	"sync"
	"time"

	"github.com/reusee/e5"
)

const WaitGroupTheory = `
pr2.WaitGroup integrates sync.WaitGroup with context.Context.

1. Hierarchical structure: NewWaitGroup can derive from a parent WaitGroup (stored in context), forming a tree. The parent will wait for its children to finish.
2. Context synchronization: WaitGroup implements the context.Context interface. Cancellation is propagated down the hierarchy.
3. Safety: Add() and Go() operations are guarded by context cancellation to prevent starting new work after the group is shutting down.
`

type waitGroupKey struct{}

var WaitGroupKey = waitGroupKey{}

type WaitGroup struct {
	ctx    context.Context
	wg     *sync.WaitGroup
	cancel context.CancelFunc
}

func NewWaitGroup(ctx context.Context) *WaitGroup {

	if v := ctx.Value(WaitGroupKey); v != nil {
		// if there is parent wait group, derive from it
		parentWaitGroup := v.(*WaitGroup)
		parentWaitGroup.wg.Add(1)
		ctx, cancel := context.WithCancel(ctx)
		wg := &WaitGroup{
			wg:     new(sync.WaitGroup),
			cancel: cancel,
		}
		ctx = context.WithValue(ctx, WaitGroupKey, wg)
		wg.ctx = ctx
		go func() {
			<-ctx.Done()
			wg.wg.Wait()
			parentWaitGroup.wg.Done()
		}()
		return wg
	}

	// new root wait group
	ctx, cancel := context.WithCancel(ctx)
	wg := &WaitGroup{
		wg:     new(sync.WaitGroup),
		cancel: cancel,
	}
	ctx = context.WithValue(ctx, WaitGroupKey, wg)
	wg.ctx = ctx
	return wg
}

func GetWaitGroup(ctx context.Context) *WaitGroup {
	if v := ctx.Value(WaitGroupKey); v != nil {
		return v.(*WaitGroup)
	}
	return nil
}

func (w *WaitGroup) Cancel() {
	w.cancel()
}

func (w *WaitGroup) Add() (done func()) {
	select {
	case <-w.ctx.Done():
		e5.Throw(context.Canceled)
	default:
	}
	w.wg.Add(1)
	var doneOnce sync.Once
	return func() {
		doneOnce.Do(func() {
			w.wg.Done()
		})
	}
}

func (w *WaitGroup) Wait() {
	w.wg.Wait()
}

var _ context.Context = new(WaitGroup)

func (w *WaitGroup) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w *WaitGroup) Err() error {
	return w.ctx.Err()
}

func (w *WaitGroup) Deadline() (deadline time.Time, ok bool) {
	return w.ctx.Deadline()
}

func (w *WaitGroup) Value(key any) any {
	return w.ctx.Value(key)
}

func (w *WaitGroup) Go(fn func()) {
	done := w.Add()
	go func() {
		defer done()
		fn()
	}()
}
