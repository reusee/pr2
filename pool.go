package pr2

import (
	"runtime"
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	newFunc    func(put PoolPutFunc) T
	pool       []_PoolElem[T]
	Callers    [][]byte
	capacity   uint32
	LogCallers bool
}

type PoolPutFunc = func() bool

type _PoolElem[T any] struct {
	Refs   atomic.Int32
	Put    func() bool
	IncRef func()
	Value  T
}

func NewPool[T any](
	capacity uint32,
	newFunc func(put PoolPutFunc) T,
) *Pool[T] {

	pool := &Pool[T]{
		capacity: capacity,
		newFunc:  newFunc,
	}

	for i := uint32(0); i < capacity; i++ {
		i := i
		put := func() bool {
			if c := pool.pool[i].Refs.Add(-1); c == 0 {
				if pool.LogCallers {
					pool.Callers[i] = nil
				}
				return true
			} else if c < 0 {
				panic("bad put")
			}
			return false
		}
		pool.pool = append(pool.pool, _PoolElem[T]{
			Value: newFunc(put),

			Put: put,

			IncRef: func() {
				pool.pool[i].Refs.Add(1)
			},
		})
	}
	pool.Callers = make([][]byte, capacity)

	return pool
}

func (p *Pool[T]) Get(ptr *T) (put func() bool) {
	put, _ = p.GetRC(ptr)
	return
}

func (p *Pool[T]) GetRC(ptr *T) (
	put func() bool,
	incRef func(),
) {

	for i := 0; i < 4; i++ {
		idx := fastrand() % p.capacity
		if p.pool[idx].Refs.CompareAndSwap(0, 1) {
			if p.LogCallers {
				stack := make([]byte, 8*1024)
				runtime.Stack(stack, false)
				p.Callers[idx] = stack
			}
			*ptr = p.pool[idx].Value
			put = p.pool[idx].Put
			incRef = p.pool[idx].IncRef
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
	get func(*T),
	putAll func(),
) {

	var l sync.Mutex
	var curPut func()

	get = func(ptr *T) {
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

//go:linkname fastrand runtime.fastrand
func fastrand() uint32
