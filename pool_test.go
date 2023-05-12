package pr2

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestBytesPool(t *testing.T) {
	pool := NewPool(8, func(_ PoolPutFunc) *[]byte {
		bs := make([]byte, 8)
		return &bs
	})
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var bs *[]byte
				put := pool.Get(&bs)
				defer put()
				binary.PutUvarint(*bs, uint64(i))
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

func TestBytesPoolRC(t *testing.T) {
	pool := NewPool(8, func(_ PoolPutFunc) *[]byte {
		bs := make([]byte, 8)
		return &bs
	})
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var bs *[]byte
				put, inc := pool.GetRC(&bs)
				defer put()
				nRef := rand.Intn(16)
				for i := 0; i < nRef; i++ {
					inc()
				}
				defer func() {
					for i := 0; i < nRef; i++ {
						put()
					}
				}()
				binary.PutUvarint(*bs, uint64(i))
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

func TestBytesPoolRCOverload(t *testing.T) {
	pool := NewPool(1, func(put PoolPutFunc) int {
		return 1
	})
	var i int
	pool.GetRC(&i)
	var j int
	put, inc := pool.GetRC(&j)
	inc()
	if put() {
		t.Fatal()
	}
	if !put() {
		t.Fatal()
	}
}

func BenchmarkBytesPool(b *testing.B) {
	pool := NewPool(8, func(_ PoolPutFunc) *[]byte {
		bs := make([]byte, 8)
		return &bs
	})
	var v *[]byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		put := pool.Get(&v)
		put()
	}
}

func BenchmarkParallelBytesPool(b *testing.B) {
	pool := NewPool(1024, func(_ PoolPutFunc) *[]byte {
		bs := make([]byte, 8)
		return &bs
	})
	var v *[]byte
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			put := pool.Get(&v)
			put()
		}
	})
}

func TestGetter(t *testing.T) {
	pool := NewPool(8, func(_ PoolPutFunc) *[]byte {
		bs := make([]byte, 8)
		return &bs
	})
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			get, put := pool.Getter()
			defer put()
			for j := 0; j < 200; j++ {
				var v *[]byte
				get(&v)
				binary.PutUvarint(*v, uint64(i))
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

func TestPoolBadPut(t *testing.T) {
	pool := NewPool(1, func(put PoolPutFunc) int {
		return 1
	})
	var i int
	put := pool.Get(&i)
	put()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal()
			}
			if fmt.Sprintf("%v", p) != "bad put" {
				t.Fatal()
			}
		}()
		put()
	}()
}

func TestPoolBadPutRC(t *testing.T) {
	pool := NewPool(1, func(put PoolPutFunc) int {
		return 1
	})
	var j int
	pool.Get(&j)
	var i int
	put := pool.Get(&i)
	put()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal()
			}
			if fmt.Sprintf("%v", p) != "bad put" {
				t.Fatal()
			}
		}()
		put()
	}()
}
