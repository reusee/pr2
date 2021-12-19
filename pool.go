package pr

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type Pool struct {
	newFunc    func() any
	pool       []_PoolElem
	Callers    [][]byte
	capacity   int32
	n          int32
	LogCallers bool
}

type _PoolElem struct {
	Taken  uint32
	Refs   int32
	Value  any
	Put    func() bool
	IncRef func()
}

func NewPool(
	capacity int32,
	newFunc func() any,
) *Pool {

	pool := &Pool{
		capacity: capacity,
		newFunc:  newFunc,
	}

	for i := int32(0); i < capacity; i++ {
		i := i
		pool.pool = append(pool.pool, _PoolElem{
			Value: newFunc(),

			Put: func() bool {
				if c := atomic.AddInt32(&pool.pool[i].Refs, -1); c == 0 {
					if pool.LogCallers {
						pool.Callers[i] = nil
					}
					atomic.StoreUint32(&pool.pool[i].Taken, 0)
					return true
				} else if c < 0 {
					panic("bad put")
				}
				return false
			},

			IncRef: func() {
				atomic.AddInt32(&pool.pool[i].Refs, 1)
			},
		})
	}
	pool.Callers = make([][]byte, capacity)

	return pool
}

func (p *Pool) Get() (value any, put func() bool) {
	value, put, _ = p.GetRC()
	return
}

func (p *Pool) GetRC() (
	value any,
	put func() bool,
	incRef func(),
) {
	for i := 0; i < 4; i++ {
		idx := atomic.AddInt32(&p.n, 1) % p.capacity
		if atomic.CompareAndSwapUint32(&p.pool[idx].Taken, 0, 1) {
			if p.LogCallers {
				stack := make([]byte, 8*1024)
				runtime.Stack(stack, false)
				p.Callers[idx] = stack
			}
			value = p.pool[idx].Value
			put = p.pool[idx].Put
			incRef = p.pool[idx].IncRef
			p.pool[idx].Refs = 1
			return
		}
	}
	value = p.newFunc()
	put = noopPut
	incRef = noopIncRef
	return
}

func noopPut() bool {
	return false
}

func noopIncRef() {
}

func (p *Pool) Getter() (
	get func() any,
	putAll func(),
) {

	var l sync.Mutex
	curPut := func() {}

	get = func() any {
		v, put := p.Get()
		l.Lock()
		cur := curPut
		newPut := func() {
			put()
			cur()
		}
		curPut = newPut
		l.Unlock()
		return v
	}

	putAll = func() {
		l.Lock()
		put := curPut
		curPut = func() {}
		l.Unlock()
		put()
	}

	return
}
