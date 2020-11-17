package pr

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func BenchmarkConsume(b *testing.B) {

	for i := 1; i < 16; i *= 2 {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			put, wait := Consume(ctx, i, func(i int, v any) error {
				return nil
			})
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				put(i)
			}
			if err := wait(true); err != nil {
				b.Fatal(err)
			}

		})
	}

}

func BenchmarkBaseline(b *testing.B) {
	for i := 1; i < 16; i *= 2 {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {

			wg := new(sync.WaitGroup)
			wg.Add(i)
			ch := make(chan int)
			for range make([]struct{}, i) {
				go func() {
					defer wg.Done()
					for range ch {
					}
				}()
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ch <- i
			}
			close(ch)
			wg.Wait()

		})
	}

}
