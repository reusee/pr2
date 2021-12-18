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
	Value any
	Taken int32
	Put   func() bool
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
				if pool.LogCallers {
					pool.Callers[i] = nil
				}
				atomic.StoreInt32(&pool.pool[i].Taken, 0)
				return true
			},
		})
	}
	pool.Callers = make([][]byte, capacity)

	return pool
}

func (p *Pool) Get() (value any, put func() bool) {
	idx := atomic.AddInt32(&p.n, 1) % p.capacity
	if atomic.CompareAndSwapInt32(&p.pool[idx].Taken, 0, 1) {
		if p.LogCallers {
			stack := make([]byte, 8*1024)
			runtime.Stack(stack, false)
			p.Callers[idx] = stack
		}
		value = p.pool[idx].Value
		put = p.pool[idx].Put
	} else {
		value = p.newFunc()
		put = noopPut
	}
	return
}

func noopPut() bool {
	return false
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
