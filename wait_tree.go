package pr

import (
	"context"
	"sync"

	"github.com/reusee/e4"
)

func NewWaitTree(
	parentCtx context.Context,
	parentAdd func() func(),
	finally func(),
) (
	ctx context.Context,
	add func() func(),
	cancel func(),
	wait func(),
) {

	var parentDone func()
	if parentAdd != nil {
		parentDone = parentAdd()
	}

	ctx, cancel = context.WithCancel(parentCtx)

	wg := new(sync.WaitGroup)

	var runFinallyOnce sync.Once

	add = func() func() {
		select {
		case <-ctx.Done():
			runFinallyOnce.Do(func() {
				if finally != nil {
					finally()
				}
			})
			e4.Throw(context.Canceled)
		default:
		}
		wg.Add(1)
		var doneOnce sync.Once
		return func() {
			doneOnce.Do(func() {
				wg.Done()
			})
		}
	}

	wait = func() {
		wg.Wait()
		runFinallyOnce.Do(func() {
			if finally != nil {
				finally()
			}
		})
		if parentDone != nil {
			parentDone()
		}
	}

	return
}
