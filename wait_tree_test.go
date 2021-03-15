package pr

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/reusee/e4"
)

func TestWaitTree(t *testing.T) {

	t.Run("single", func(t *testing.T) {
		ctx, add, cancel, wait, done := NewWaitTree(context.Background(), nil, nil)
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
		done()
		if c != int64(n) {
			t.Fatal()
		}
	})

	t.Run("tree", func(t *testing.T) {
		ctx, add, cancel, wait, done := NewWaitTree(context.Background(), nil, nil)
		var c int64
		n := 128
		m := 8
		for i := 0; i < m; i++ {
			ctx1, add1, cancel1, wait1, done1 := NewWaitTree(ctx, add, nil)
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
				done1()
			}()
		}
		_ = cancel
		wait()
		done()
		if c != int64(n*m) {
			t.Fatal()
		}
	})

	t.Run("cancel", func(t *testing.T) {
		var num int
		_, add, cancel, _, _ := NewWaitTree(context.Background(), nil, func() {
			num++
		})
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
				if num != 1 {
					t.Fatal()
				}
			}()
			defer e4.Handle(&err)
			add()
		}()
	})

}