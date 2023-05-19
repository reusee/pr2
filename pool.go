package pr2

import (
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	l         sync.Mutex
	newFunc   func(put PoolPutFunc) *T
	resetFunc func(*T)
	elems     atomic.Pointer[[]_PoolElem[T]]
	capacity  uint32
}

type PoolPutFunc = func() bool

type _PoolElem[T any] struct {
	refs   atomic.Int32
	put    func() bool
	incRef func()
	value  *T
}

func NewPool[T any](
	capacity uint32,
	newFunc func(put PoolPutFunc) *T,
	resetFunc func(*T),
) *Pool[T] {
	pool := &Pool[T]{
		capacity:  capacity,
		newFunc:   newFunc,
		resetFunc: resetFunc,
	}
	pool.allocElems()
	return pool
}

func (p *Pool[T]) allocElems() {
	cur := p.elems.Load()
	p.l.Lock()
	defer p.l.Unlock()
	if p.elems.Load() != cur {
		// refreshed
		return
	}
	elems := make([]_PoolElem[T], p.capacity)
	for i := uint32(0); i < p.capacity; i++ {
		i := i
		var ptr *T
		put := func() bool {
			if c := elems[i].refs.Add(-1); c == 0 {
				if p.resetFunc != nil {
					p.resetFunc(ptr)
				}
				return true
			} else if c < 0 {
				panic("bad put")
			}
			return false
		}
		ptr = p.newFunc(put)
		elems[i] = _PoolElem[T]{
			value: ptr,
			put:   put,
			incRef: func() {
				elems[i].refs.Add(1)
			},
		}
	}
	p.elems.Store(&elems)
}

func (p *Pool[T]) Get(ptr **T) (put func() bool) {
	put, _ = p.GetRC(ptr)
	return
}

func (p *Pool[T]) GetRC(ptr **T) (
	put func() bool,
	incRef func(),
) {

	for {
		elems := *p.elems.Load()
		for i := 0; i < 4; i++ {
			idx := fastrand() % p.capacity
			if elems[idx].refs.CompareAndSwap(0, 1) {
				*ptr = elems[idx].value
				put = elems[idx].put
				incRef = elems[idx].incRef
				return
			}
		}

		p.allocElems()
	}

}

func (p *Pool[T]) Getter() (
	get func(**T),
	putAll func(),
) {

	var l sync.Mutex
	var curPut func()

	get = func(ptr **T) {
		put := p.Get(ptr)
		l.Lock()
		if curPut != nil {
			cur := curPut
			newPut := func() {
				put()
				cur()
			}
			curPut = newPut
		} else {
			curPut = func() {
				put()
			}
		}
		l.Unlock()
	}

	putAll = func() {
		l.Lock()
		put := curPut
		curPut = nil
		l.Unlock()
		put()
	}

	return
}

func ResetSlice[T any](size int, capacity int) func(*[]T) {
	return func(ptr *[]T) {
		if capacity >= 0 {
			if len(*ptr) != size || cap(*ptr) != capacity {
				*ptr = (*ptr)[:size:capacity]
			}
		} else {
			if len(*ptr) != size {
				*ptr = (*ptr)[:size]
			}
		}
	}
}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32
