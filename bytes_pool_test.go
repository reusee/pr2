package pr

import (
	"encoding/binary"
	"testing"
)

func TestBytesPool(t *testing.T) {
	get := NewBytesPool(8, 8)
	for i := 0; i < 200; i++ {
		i := i
		go func() {
			for j := 0; j < 200; j++ {
				bs, put := get()
				defer put()
				binary.PutUvarint(bs, uint64(i))
			}
		}()
	}
}

func BenchmarkBytesPool(b *testing.B) {
	get := NewBytesPool(8, 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, put := get()
		put()
	}
}

func BenchmarkParallelBytesPool(b *testing.B) {
	get := NewBytesPool(8, 8)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, put := get()
			put()
		}
	})
}
