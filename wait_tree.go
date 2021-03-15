package pr

import (
	"context"
	"sync"

	"github.com/reusee/e4"
)

func NewWaitTree(
	parentCtx context.Context,
	parentAdd func() func(),
	onCanceledError func(),
) (
	ctx context.Context,
	add func() func(),
	cancel func(),
	wait func(),
	done func(),
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
			if onCanceledError != nil {
				onCanceledError()
			}
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
	}

	done = func() {
		if parentDone != nil {
			parentDone()
		}
	}

	return
}
