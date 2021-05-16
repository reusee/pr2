package pr

import (
	"encoding/binary"
	"sync"
	"testing"
)

func TestBytesPool(t *testing.T) {
	pool := NewPool(8, func() any {
		bs := make([]byte, 8)
		return &bs
	})
	pool.LogCallers = true
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				bs, put := pool.Get()
				defer put()
				binary.PutUvarint(*bs.(*[]byte), uint64(i))
			}
		}()
	}
	wg.Wait()
	for _, caller := range pool.Callers {
		if len(caller) > 0 {
			t.Fatalf("not put: %s\n", caller)
		}
	}
}

func BenchmarkBytesPool(b *testing.B) {
	pool := NewPool(8, func() any {
		bs := make([]byte, 8)
		return &bs
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, put := pool.Get()
		put()
	}
}

func BenchmarkParallelBytesPool(b *testing.B) {
	pool := NewPool(8, func() any {
		bs := make([]byte, 8)
		return &bs
	})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, put := pool.Get()
			put()
		}
	})
}

func TestGetter(t *testing.T) {
	pool := NewPool(8, func() any {
		bs := make([]byte, 8)
		return &bs
	})
	pool.LogCallers = true
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			get, put := pool.Getter()
			defer put()
			for j := 0; j < 200; j++ {
				v := get()
				binary.PutUvarint(*v.(*[]byte), uint64(i))
			}
		}()
	}
	wg.Wait()
	for _, caller := range pool.Callers {
		if len(caller) > 0 {
			t.Fatalf("not put: %s\n", caller)
		}
	}
}
