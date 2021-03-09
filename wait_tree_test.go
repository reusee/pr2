package pr

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/reusee/e4"
)

func TestWaitTree(t *testing.T) {

	t.Run("single", func(t *testing.T) {
		ctx, add, cancel, wait := NewWaitTree(context.Background(), nil)
		n := 128
		var c int64
		for i := 0; i < n; i++ {
			done := add()
			go func() {
				defer done()
				<-ctx.Done()
				atomic.AddInt64(&c, 1)
			}()
		}
		cancel()
		wait()
		if c != int64(n) {
			t.Fatal()
		}
	})

	t.Run("tree", func(t *testing.T) {
		ctx, add, cancel, wait := NewWaitTree(context.Background(), nil)
		var c int64
		n := 128
		m := 8
		for i := 0; i < m; i++ {
			ctx1, add1, cancel1, wait1 := NewWaitTree(ctx, add)
			go func() {
				for i := 0; i < n; i++ {
					done := add1()
					go func() {
						defer done()
						<-ctx1.Done()
						atomic.AddInt64(&c, 1)
					}()
				}
				cancel1()
				wait1()
			}()
		}
		_ = cancel
		wait()
		if c != int64(n*m) {
			t.Fatal()
		}
	})

	t.Run("tree2", func(t *testing.T) {
		n := int64(1024)
		var c int64
		var fork func(context.Context, func() func()) func()
		fork = func(ctx context.Context, add func() func()) func() {
			if atomic.AddInt64(&n, -1) <= 0 {
				return nil
			}
			ctx1, add1, cancel1, wait1 := NewWaitTree(ctx, add)
			for i, j := 0, rand.Intn(8); i < j; i++ {
				fork(ctx1, add1)
			}
			go func() {
				_ = cancel1
				wait1()
				atomic.AddInt64(&c, 1)
			}()
			return wait1
		}
		wait := fork(context.Background(), nil)
		wait()
		if c != 1023 {
			t.Fatal()
		}
	})

	t.Run("cancel", func(t *testing.T) {
		_, add, cancel, _ := NewWaitTree(context.Background(), nil)
		cancel()
		func() {
			var err error
			defer func() {
				if err == nil {
					t.Fatal("shoule throw error")
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatal()
				}
			}()
			defer e4.Handle(&err)
			add()
		}()
	})

}
