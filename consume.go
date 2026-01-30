package pr2

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/reusee/e5"
)

const ConsumeTheory = `
pr2.Consume implements a robust producer-consumer pattern with an internal buffer.

1. Decoupling: Producers use put() to hand off work. Consumers (workers) process work in parallel.
2. Reliability: All items accepted by put() are guaranteed to be processed, even if the context is cancelled after acceptance.
3. Resource Management: numValue tracks the total number of items in the system (queued or being processed). wait() allows for flushing (waiting for numValue == 0) or complete shutdown (waiting for goroutines to exit).
4. Backpressure: backlogSize limits the internal queue, causing put() to block when the system is overloaded.
`

type Put[T any] func(T) bool

type Wait = func(noMorePut bool) error

type ConsumeOption interface {
	IsConsumeOption()
}

type BacklogSize int

func (BacklogSize) IsConsumeOption() {}

func Consume[T any](
	ctx context.Context,
	numThread int,
	fn func(threadID int, value T) error,
	options ...ConsumeOption,
) (
	put Put[T],
	wait Wait,
) {

	backlogSize := int(math.MaxInt32)

	for _, option := range options {
		switch option := option.(type) {
		case BacklogSize:
			backlogSize = int(option)
		default:
			panic(fmt.Errorf("unknown option: %T", option))
		}
	}

	inCh := make(chan T)
	outCh := make(chan T)
	errCh := make(chan error, 1)
	valueCond := sync.NewCond(new(sync.Mutex))
	numValue := 0
	wg := NewWaitGroup(ctx)

	wg.Go(func() {
		consumeBuffer(ctx, inCh, outCh, backlogSize)
	})

	var putLock sync.RWMutex
	putClosed := false
	put = func(v T) bool {

		putLock.RLock()
		defer putLock.RUnlock()

		if putClosed || len(errCh) > 0 {
			return false
		}

		valueCond.L.Lock()
		numValue++
		valueCond.L.Unlock()

		select {
		case inCh <- v:
			return true
		case <-ctx.Done():
			decrementNumValue(valueCond, &numValue)
			return false
		}
	}

	var closeOnce sync.Once
	closePut := func() {
		closeOnce.Do(func() {
			putLock.Lock()
			defer putLock.Unlock()
			putClosed = true
			close(inCh)
		})
	}

	wait = func(noMorePut bool) error {
		if noMorePut {
			closePut()
			wg.Wait()
		}
		valueCond.L.Lock()
		for numValue != 0 {
			valueCond.Wait()
		}
		valueCond.L.Unlock()
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	}

	for i := 0; i < numThread; i++ {
		i := i
		wg.Go(func() {
			consumeWorker(i, outCh, errCh, valueCond, &numValue, fn)
		})
	}

	return

}

func consumeBuffer[T any](ctx context.Context, inCh, outCh chan T, backlogSize int) {
	values := list.New()
loop:
	for {
		c := inCh
		if values.Len() > backlogSize {
			c = nil
		}
		if values.Len() > 0 {
			select {
			case outCh <- values.Front().Value.(T):
				values.Remove(values.Front())
			case v, ok := <-c:
				if !ok {
					break loop
				}
				values.PushBack(v)
			case <-ctx.Done():
				break loop
			}
		} else {
			select {
			case v, ok := <-c:
				if !ok {
					break loop
				}
				select {
				case outCh <- v:
				default:
					values.PushBack(v)
				}
			case <-ctx.Done():
				break loop
			}
		}
	}
	for elem := values.Front(); elem != nil; elem = elem.Next() {
		outCh <- elem.Value.(T)
	}
	close(outCh)
}

func consumeWorker[T any](
	id int,
	outCh chan T,
	errCh chan error,
	cond *sync.Cond,
	numValue *int,
	fn func(int, T) error,
) {
	for v := range outCh {
		err := func() (err error) {
			defer e5.Handle(&err)
			return fn(id, v)
		}()
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
		decrementNumValue(cond, numValue)
	}
}

func decrementNumValue(cond *sync.Cond, numValue *int) {
	cond.L.Lock()
	defer cond.L.Unlock()
	*numValue--
	if *numValue == 0 {
		cond.Signal()
	}
}