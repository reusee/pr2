package pr

import "sync/atomic"

type Pool struct {
	pool     []*_PoolElem
	capacity int32
	n        int32
	newFunc  func() any
}

type _PoolElem struct {
	Value any
	Taken int32
}

func NewPool(
	capacity int32,
	newFunc func() any,
) *Pool {

	var pool []*_PoolElem
	for i := int32(0); i < capacity; i++ {
		pool = append(pool, &_PoolElem{
			Value: newFunc(),
		})
	}

	return &Pool{
		pool:     pool,
		capacity: capacity,
		newFunc:  newFunc,
	}

}

func (p *Pool) Get() (value any, put func()) {
	elem := p.pool[atomic.AddInt32(&p.n, 1)%p.capacity]
	if atomic.CompareAndSwapInt32(&elem.Taken, 0, 1) {
		value = elem.Value
		put = func() {
			atomic.StoreInt32(&elem.Taken, 0)
		}
	} else {
		value = p.newFunc()
		put = func() {}
	}
	return
}
