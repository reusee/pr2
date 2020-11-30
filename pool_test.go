package pr

import (
	"encoding/binary"
	"testing"
)

func TestBytesPool(t *testing.T) {
	pool := NewPool(8, func() any {
		bs := make([]byte, 8)
		return &bs
	})
	for i := 0; i < 200; i++ {
		i := i
		go func() {
			for j := 0; j < 200; j++ {
				bs, put := pool.Get()
				defer put()
				binary.PutUvarint(*bs.(*[]byte), uint64(i))
			}
		}()
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
