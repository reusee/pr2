package pr2

import (
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	newFunc  func(put PoolPutFunc) *T
	pool     []_PoolElem[T]
	capacity uint32
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
		capacity: capacity,
		newFunc:  newFunc,
	}

	for i := uint32(0); i < capacity; i++ {
		i := i
		var ptr *T
		put := func() bool {
			if c := pool.pool[i].refs.Add(-1); c == 0 {
				if resetFunc != nil {
					resetFunc(ptr)
				}
				return true
			} else if c < 0 {
				panic("bad put")
			}
			return false
		}
		ptr = newFunc(put)
		pool.pool = append(pool.pool, _PoolElem[T]{
			value: ptr,
			put:   put,
			incRef: func() {
				pool.pool[i].refs.Add(1)
			},
		})
	}

	return pool
}

func (p *Pool[T]) Get(ptr **T) (put func() bool) {
	put, _ = p.GetRC(ptr)
	return
}

func (p *Pool[T]) GetRC(ptr **T) (
	put func() bool,
	incRef func(),
) {

	for i := 0; i < 4; i++ {
		idx := fastrand() % p.capacity
		if p.pool[idx].refs.CompareAndSwap(0, 1) {
			*ptr = p.pool[idx].value
			put = p.pool[idx].put
			incRef = p.pool[idx].incRef
			return
		}
	}

	var refs atomic.Int32
	refs.Store(1)
	put = func() bool {
		if r := refs.Add(-1); r == 0 {
			return true
		} else if r < 0 {
			panic("bad put")
		}
		return false
	}
	incRef = func() {
		refs.Add(1)
	}
	value := p.newFunc(put)
	*ptr = value

	return
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

func ResetSlice[T any](size int) func(*[]T) {
	return func(ptr *[]T) {
		if len(*ptr) != size {
			*ptr = (*ptr)[:size]
		}
	}
}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32
