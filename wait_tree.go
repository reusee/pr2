package pr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/reusee/e4"
)

type WaitTree struct {
	ID     string
	Ctx    context.Context
	Cancel func()
	wg     sync.WaitGroup

	traceEnabled bool
	l            sync.Mutex
	traces       map[string]int
}

func NewWaitTree(
	parent *WaitTree,
	options ...WaitTreeOption,
) *WaitTree {

	var timeout time.Duration
	var id ID
	var traceEnabled bool
	for _, option := range options {
		switch option := option.(type) {
		case Timeout:
			timeout = time.Duration(option)
		case ID:
			id = option
		case Trace:
			traceEnabled = bool(option)
		default:
			panic(fmt.Errorf("unknown option: %T", option))
		}
	}

	tree := &WaitTree{
		ID:           string(id),
		traceEnabled: traceEnabled,
		traces:       make(map[string]int),
	}
	var ctx context.Context
	var cancel context.CancelFunc
	if parent != nil {
		parentDone := parent.Add()
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(parent.Ctx, timeout)
		} else {
			ctx, cancel = context.WithCancel(parent.Ctx)
		}
		tree.Ctx = ctx
		tree.Cancel = cancel
		go func() {
			<-ctx.Done()
			tree.Wait()
			parentDone()
		}()
	} else {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		tree.Ctx = ctx
		tree.Cancel = cancel
	}
	return tree
}

func NewRootWaitTree(
	options ...WaitTreeOption,
) *WaitTree {
	return NewWaitTree(nil, options...)
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
	if t.traceEnabled {
		stack = e4.WrapStacktrace(errStacktrace).Error()
		t.l.Lock()
		t.traces[stack]++
		t.l.Unlock()
	}
	var doneOnce sync.Once
	return func() {
		doneOnce.Do(func() {
			t.wg.Done()
			if t.traceEnabled {
				t.l.Lock()
				t.traces[stack]--
				t.l.Unlock()
			}
		})
	}
}

func (t *WaitTree) Wait() {
	t.wg.Wait()
}

func (t *WaitTree) Go(fn func()) {
	done := t.Add()
	go func() {
		defer done()
		fn()
	}()
}

func (t *WaitTree) PrintTraces(w io.Writer) {
	t.l.Lock()
	defer t.l.Unlock()
	for trace, num := range t.traces {
		fmt.Fprintf(w, "<%d> %s\n", num, trace)
	}
}
