package pr

import (
	"context"
	"sync"
)

type Put = func(any) bool

type Wait = func(noMorePut bool) error

func Consume(
	ctx context.Context,
	numThread int,
	fn func(threadID int, value any) error,
) (
	put Put,
	wait Wait,
) {

	inCh := make(chan any)
	outCh := make(chan any)
	threadWaitGroup := new(sync.WaitGroup)
	errCh := make(chan error, 1)
	valueCond := sync.NewCond(new(sync.Mutex))
	numValue := 0

	threadWaitGroup.Add(1)
	go func() {
		defer threadWaitGroup.Done()
		var values []any
		n := 0
	loop:
		for {

			if len(values) > 0 {
				select {

				case outCh <- values[0]:
					values = values[1:]
					n++
					if n == 1024 {
						// avoid leak
						values = append(values[:0:0], values...)
						n = 0
					}

				case v, ok := <-inCh:
					if !ok {
						break loop
					}
					values = append(values, v)

				case <-ctx.Done():
					break loop

				}

			} else {
				select {

				case v, ok := <-inCh:
					if !ok {
						break loop
					}
					select {
					case outCh <- v:
					default:
						values = append(values, v)
					}

				case <-ctx.Done():
					break loop

				}
			}

		}

		for _, v := range values {
			outCh <- v
		}

		close(outCh)

	}()

	var putLock sync.RWMutex
	putClosed := false
	put = func(v any) bool {

		putLock.RLock()
		defer putLock.RUnlock()

		if putClosed {
			return false
		}

		if len(errCh) > 0 {
			return false
		}

		select {

		case inCh <- v:
			valueCond.L.Lock()
			numValue++
			valueCond.L.Unlock()
			valueCond.Signal()
			return true

		case <-ctx.Done():
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
			threadWaitGroup.Wait()
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
		}

		return nil
	}

	for i := 0; i < numThread; i++ {
		i := i
		threadWaitGroup.Add(1)

		go func() {
			defer threadWaitGroup.Done()

			for v := range outCh {
				func() {
					defer func() {
						valueCond.L.Lock()
						numValue--
						valueCond.L.Unlock()
						valueCond.Signal()
					}()

					if err := fn(i, v); err != nil {
						select {
						case errCh <- err:
							return
						default:
						}
					}
				}()
			}

		}()

	}

	return

}
