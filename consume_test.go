package pr

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestConsume(
	t *testing.T,
) {

	t.Run("normal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(i) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal()
		}
		if c != 8 {
			t.Fatal()
		}
		if c != numPut {
			t.Fatal()
		}
	})

	t.Run("cancel before put", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		cancel()
		var numPut int
		for i := 0; i < 8; i++ {
			if put(i) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatal()
		}
		if numPut != 0 {
			t.Fatal()
		}
	})

	t.Run("close before put", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		if err := wait(true); err != nil {
			t.Fatal()
		}
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(i) {
				numPut++
			}
		}
		if err := wait(true); err != nil {
			t.Fatal()
		}
		if c != 0 {
			t.Fatal()
		}
		if c != numPut {
			t.Fatal()
		}
	})

	t.Run("concurrent put and close", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 128; i++ {
			i := i
			go func() {
				if put(i) {
					atomic.AddInt64(&numPut, 1)
				}
			}()
		}
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
		if numPut != c {
			t.Fatal()
		}
	})

	t.Run("concurrent put and cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 128; i++ {
			i := i
			go func() {
				if put(i) {
					atomic.AddInt64(&numPut, 1)
				}
			}()
		}
		cancel()
		if err := wait(true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fn error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				if v.(int) == 3 {
					return errors.New("foo")
				}
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 8; i++ {
			if put(i) {
				numPut++
			}
		}
		err := wait(true)
		if err == nil || err.Error() != "foo" {
			t.Fatal()
		}
		if numPut != c {
			t.Fatal()
		}
		time.Sleep(time.Millisecond * 10)
	})

	t.Run("multiple fn error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		put, wait := Consume(
			ctx,
			128,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				if v.(int) > 1 {
					return errors.New("foo")
				}
				return nil
			},
		)
		var numPut int64
		for i := 0; i < 128; i++ {
			if put(i) {
				numPut++
			}
		}
		err := wait(true)
		if err == nil || err.Error() != "foo" {
			t.Fatal()
		}
		if numPut != c {
			t.Fatal()
		}
	})

	t.Run("put in func", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c int64
		var put Put
		put, wait := Consume(
			ctx,
			8,
			func(i int, v any) error {
				atomic.AddInt64(&c, 1)
				n := v.(int)
				if n == 0 {
					return nil
				}
				put(n - 1)
				return nil
			},
		)
		put(2048)
		if err := wait(false); err != nil {
			t.Fatal()
		}
		if c != 2049 {
			t.Fatal()
		}
	})

}
