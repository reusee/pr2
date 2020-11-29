package pr

import "sync/atomic"

func NewPool(
	capacity int32,
	newFunc func() any,
) (
	get func() (
		v any,
		put func(),
	),
) {

	type Elem struct {
		Value any
		Taken int32
	}
	var pool []*Elem
	for i := int32(0); i < capacity; i++ {
		pool = append(pool, &Elem{
			Value: newFunc(),
		})
	}

	var n int32
	get = func() (value any, put func()) {
		elem := pool[atomic.AddInt32(&n, 1)%capacity]
		if atomic.CompareAndSwapInt32(&elem.Taken, 0, 1) {
			value = elem.Value
			put = func() {
				atomic.StoreInt32(&elem.Taken, 0)
			}
		} else {
			value = newFunc()
			put = func() {}
		}
		return
	}

	return
}
