package pr

import (
	"context"
	"sync"

	"github.com/reusee/e4"
)

func NewWaitTree(
	parentCtx context.Context,
	parentAdd func() func(),
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

	add = func() func() {
		select {
		case <-ctx.Done():
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
		if parentDone != nil {
			parentDone()
		}
	}

	return
}
