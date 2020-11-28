package pr

import "sync/atomic"

func NewBytesPool(size int64, capacity int32) (
	get func() (
		bs []byte,
		put func(),
	),
) {

	type Elem struct {
		Bytes []byte
		Taken int32
	}
	var pool []*Elem
	for i := int32(0); i < capacity; i++ {
		bs := make([]byte, size)
		pool = append(pool, &Elem{
			Bytes: bs,
		})
	}

	var n int32
	get = func() (bs []byte, put func()) {
		elem := pool[atomic.AddInt32(&n, 1)%capacity]
		if atomic.CompareAndSwapInt32(&elem.Taken, 0, 1) {
			bs = elem.Bytes
			put = func() {
				atomic.StoreInt32(&elem.Taken, 0)
			}
		} else {
			bs = make([]byte, size)
			put = func() {}
		}
		return
	}

	return
}
