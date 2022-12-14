package pr2

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/reusee/e5"
)

func TestWaitGroup(t *testing.T) {

	t.Run("single", func(t *testing.T) {
		wg := NewWaitGroup(context.Background())
		n := 128
		var c int64
		for i := 0; i < n; i++ {
			wg.Go(func() {
				<-wg.Done()
				atomic.AddInt64(&c, 1)
			})
		}
		wg.Cancel()
		wg.Wait()
		if c != int64(n) {
			t.Fatal()
		}
	})

	t.Run("tree", func(t *testing.T) {
		wg := NewWaitGroup(context.Background())
		var c int64
		n := 128
		m := 8
		for i := 0; i < m; i++ {
			subWg := NewWaitGroup(wg)
			go func() {
				for i := 0; i < n; i++ {
					subWg.Go(func() {
						<-subWg.Done()
						atomic.AddInt64(&c, 1)
					})
				}
				subWg.Cancel()
				subWg.Wait()
			}()
		}
		wg.Wait()
		if c != int64(n*m) {
			t.Fatal()
		}
	})

	t.Run("cancel", func(t *testing.T) {
		var num int
		wg := NewWaitGroup(context.Background())
		wg.Go(func() {
			<-wg.Done()
			num++
		})
		wg.Cancel()
		err := wg.Err()
		if !errors.Is(err, context.Canceled) {
			t.Fatal()
		}
		func() {
			var err error
			defer func() {
				if err == nil {
					t.Fatal("shoule throw error")
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatal()
				}
				wg.Wait()
				if num != 1 {
					t.Fatal()
				}
			}()
			defer e5.Handle(&err)
			wg.Add()
		}()
	})

	t.Run("get", func(t *testing.T) {
		wg := NewWaitGroup(context.Background())
		wg2 := GetWaitGroup(wg)
		if wg2 != wg {
			t.Fatal()
		}

		wg2 = GetWaitGroup(context.Background())
		if wg2 != nil {
			t.Fatal()
		}
	})

}
