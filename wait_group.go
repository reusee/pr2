package pr

import (
	"context"
	"sync"

	"github.com/reusee/e5"
)

type waitGroupKey struct{}

var WaitGroupKey = waitGroupKey{}

type WaitGroup struct {
	ctx       context.Context
	wg        *sync.WaitGroup
	cancelCtx context.CancelFunc
	parent    *WaitGroup
}

func newWaitGroup(
	parent *WaitGroup,
	ctx context.Context,
	cancel context.CancelFunc,
) *WaitGroup {
	return &WaitGroup{
		ctx:       ctx,
		wg:        new(sync.WaitGroup),
		cancelCtx: cancel,
		parent:    parent,
	}
}

func WithWaitGroup(
	ctx context.Context,
) (
	newCtx context.Context,
	wg *WaitGroup,
) {

	var parentWaitGroup *WaitGroup
	if v := ctx.Value(WaitGroupKey); v != nil {
		parentWaitGroup = v.(*WaitGroup)
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		parentWaitGroup = newWaitGroup(nil, ctx, cancel)
	}
	parentWaitGroup.wg.Add(1)

	newCtx, cancel := context.WithCancel(ctx)
	waitGroup := newWaitGroup(parentWaitGroup, newCtx, cancel)
	newCtx = context.WithValue(newCtx, WaitGroupKey, waitGroup)

	go func() {
		<-newCtx.Done()
		waitGroup.wg.Wait()
		parentWaitGroup.wg.Done()
	}()

	return newCtx, waitGroup
}

func (w *WaitGroup) Cancel() {
	w.cancelCtx()
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

func (w *WaitGroup) Go(fn func()) {
	done := w.Add()
	go func() {
		defer done()
		fn()
	}()
}

func (w *WaitGroup) Parent() *WaitGroup {
	return w.parent
}
