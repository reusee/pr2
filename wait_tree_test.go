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
		tree := NewWaitTree(nil, nil)
		n := 128
		var c int64
		for i := 0; i < n; i++ {
			done := tree.Add()
			go func() {
				defer done()
				<-tree.Ctx.Done()
				atomic.AddInt64(&c, 1)
			}()
		}
		tree.Cancel()
		tree.Wait()
		tree.Done()
		if c != int64(n) {
			t.Fatal()
		}
	})

	t.Run("tree", func(t *testing.T) {
		tree := NewWaitTree(nil, nil)
		var c int64
		n := 128
		m := 8
		for i := 0; i < m; i++ {
			tree1 := NewWaitTree(tree, nil)
			go func() {
				for i := 0; i < n; i++ {
					done := tree1.Add()
					go func() {
						defer done()
						<-tree1.Ctx.Done()
						atomic.AddInt64(&c, 1)
					}()
				}
				tree1.Cancel()
				tree1.Wait()
				tree1.Done()
			}()
		}
		tree.Wait()
		tree.Done()
		if c != int64(n*m) {
			t.Fatal()
		}
	})

	t.Run("cancel", func(t *testing.T) {
		var num int
		tree := NewWaitTree(nil, func() {
			num++
		})
		tree.Cancel()
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
			tree.Add()
		}()
	})

}
