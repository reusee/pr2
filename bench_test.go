package pr

import (
	"context"
	"runtime"
	"testing"
)

func BenchmarkConsume(b *testing.B) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	put, wait := Consume(ctx, runtime.NumCPU(), func(i int, v any) error {
		return nil
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		put(i)
	}
	if err := wait(true); err != nil {
		b.Fatal(err)
	}

}
