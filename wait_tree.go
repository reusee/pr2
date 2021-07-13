package pr

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reusee/e4"
)

type WaitTree struct {
	Ctx    context.Context
	Cancel func()
	wg     sync.WaitGroup

	sync.Mutex
	traces map[string]int
}

var traceWaitTree = func() int {
	env := os.Getenv("TRACE_WAIT_TREE")
	if len(env) == 0 {
		return 0
	}
	n, err := strconv.Atoi(env)
	if err != nil {
		panic("bad TRACE_WAIT_TREE value")
	}
	return n
}()

func NewWaitTree(
	parent *WaitTree,
) *WaitTree {
	tree := &WaitTree{
		traces: make(map[string]int),
	}
	if parent != nil {
		parentDone := parent.Add()
		ctx, cancel := context.WithCancel(parent.Ctx)
		tree.Ctx = ctx
		tree.Cancel = cancel
		go func() {
			<-ctx.Done()
			tree.Wait()
			parentDone()
		}()
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		tree.Ctx = ctx
		tree.Cancel = cancel
	}
	return tree
}

func NewRootWaitTree(
	ctx context.Context,
) *WaitTree {
	tree := &WaitTree{
		traces: make(map[string]int),
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	tree.Ctx = ctx
	tree.Cancel = cancel
	return tree
}

var errStacktrace = errors.New("stack trace")

func (t *WaitTree) Add() (done func()) {
	select {
	case <-t.Ctx.Done():
		e4.Throw(context.Canceled)
	default:
	}
	t.wg.Add(1)
	var stack string
	if traceWaitTree > 0 {
		stack = e4.WrapStacktrace(errStacktrace).Error()
		t.Lock()
		t.traces[stack]++
		t.Unlock()
	}
	var doneOnce sync.Once
	return func() {
		doneOnce.Do(func() {
			t.wg.Done()
			if traceWaitTree > 0 {
				t.Lock()
				t.traces[stack]--
				t.Unlock()
			}
		})
	}
}

func (t *WaitTree) Wait() {
	if traceWaitTree > 0 {
		var ok int64
		time.AfterFunc(time.Second*time.Duration(traceWaitTree), func() {
			if atomic.CompareAndSwapInt64(&ok, 0, 1) {
				t.Lock()
				for stack, n := range t.traces {
					if n == 0 {
						continue
					}
					pt("WAIT TREE BLOCKING: %s\n", stack)
				}
				t.Unlock()
			}
		})
		t.wg.Wait()
		atomic.StoreInt64(&ok, 1)
	} else {
		t.wg.Wait()
	}
}

func (t *WaitTree) Go(fn func()) {
	done := t.Add()
	go func() {
		defer done()
		fn()
	}()
}
