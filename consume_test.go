package pr

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reusee/e5"
)

func TestConsume(
	t *testing.T,
) {

	t.Run("normal", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		if c != 8 {
			t.Fatalf("got %d", c)
		}
		if c != numPut {
			t.Fatalf("got %d, %d", c, numPut)
		}
	})

	t.Run("multiple wait", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)

		var numPut int64
		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		if err := wait(false); err != nil {
			t.Fatal(err)
		}
		if c != 8 {
			t.Fatalf("got %d", c)
		}
		if c != numPut {
			t.Fatalf("got %d, %d", c, numPut)
		}

		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		if err := wait(false); err != nil {
			t.Fatal(err)
		}
		if c != 16 {
			t.Fatalf("got %d", 16)
		}
		if c != numPut {
			t.Fatalf("got %d, %d", c, numPut)
		}
	})

	t.Run("cancel before put", func(t *testing.T) {
		var c int64
		ctx, wg := WithWaitGroup(context.Background())
		put, wait := Consume(
			ctx,
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		wg.Cancel()
		var numPut int
		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatalf("got %d", c)
		}
		if numPut != 0 {
			t.Fatalf("got %d", numPut)
		}
	})

	t.Run("close before put", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatalf("got %d", c)
		}
		if c != numPut {
			t.Fatalf("got %d", numPut)
		}
	})

	t.Run("concurrent put and close", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		wg := new(sync.WaitGroup)
		for i := 0; i < 128; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				if put(int64(i)) {
					atomic.AddInt64(&numPut, 1)
				}
			}()
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		if a, b := atomic.LoadInt64(&numPut), atomic.LoadInt64(&c); a != b {
			t.Fatalf("got %d, %d", a, b)
		}
	})

	t.Run("concurrent put and cancel", func(t *testing.T) {
		ctx, waitGroup := WithWaitGroup(context.Background())
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(_ int, _ int64) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		wg := new(sync.WaitGroup)
		for i := 0; i < 128; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				if put(int64(i)) {
					atomic.AddInt64(&numPut, 1)
				}
			}()
		}
		waitGroup.Cancel()
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		if a, b := atomic.LoadInt64(&numPut), atomic.LoadInt64(&c); a != b {
			t.Fatalf("got %d, %d", a, b)
		}
	})

	t.Run("fn error", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, v int64) error {
				atomic.AddInt64(&c, 1)
				if v == 3 {
					return errors.New("foo")
				}
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		err := wait(true)
		if err == nil {
			t.Fatal()
		}
		if err.Error() != "foo" {
			t.Fatalf("got %v", err)
		}
		if numPut != c {
			t.Fatalf("got %d, %d", numPut, c)
		}
		time.Sleep(time.Millisecond * 10)
	})

	t.Run("multiple fn error", func(t *testing.T) {
		var c int64
		put, wait := Consume(
			context.Background(),
			128,
			func(_ int, v int64) error {
				atomic.AddInt64(&c, 1)
				if v > 1 {
					return errors.New("foo")
				}
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 128; i++ {
			if put(int64(i)) {
				numPut++
			}
		}
		err := wait(true)
		if err == nil {
			t.Fatal()
		}
		if err.Error() != "foo" {
			t.Fatalf("got %v", err)
		}
		if numPut != c {
			t.Fatalf("got %d, %d", numPut, c)
		}
	})

	t.Run("put in func", func(t *testing.T) {
		var c int64
		var put Put[int64]
		put, wait := Consume(
			context.Background(),
			8,
			func(_ int, n int64) error {
				atomic.AddInt64(&c, 1)
				if n == 0 {
					return nil
				}
				put(n - 1)
				return nil
			},
		)
		put(2048)
		if err := wait(false); err != nil {
			t.Fatal(err)
		}
		if c != 2049 {
			t.Fatalf("got %d", c)
		}
	})

}

func TestConsumeE4Error(t *testing.T) {
	put, wait := Consume(context.Background(), 8, func(_ int, v func() error) error {
		return v()
	})
	put(func() error {
		return func() error {
			e5.Check(io.EOF)
			return nil
		}()
	})
	err := wait(true)
	if !errors.Is(err, io.EOF) {
		t.Fatal()
	}
}
