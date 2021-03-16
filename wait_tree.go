package pr

import (
	"context"
	"sync"

	"github.com/reusee/e4"
)

type WaitTree struct {
	Ctx             context.Context
	Cancel          func()
	parentDone      func()
	onCanceledError func()
	wg              sync.WaitGroup
}

func NewWaitTree(
	parent *WaitTree,
	onCanceledError func(),
) *WaitTree {
	tree := &WaitTree{
		onCanceledError: onCanceledError,
	}
	if parent != nil {
		tree.parentDone = parent.Add()
		ctx, cancel := context.WithCancel(parent.Ctx)
		tree.Ctx = ctx
		tree.Cancel = cancel
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		tree.Ctx = ctx
		tree.Cancel = cancel
	}
	return tree
}

func (t *WaitTree) Add() (done func()) {
	select {
	case <-t.Ctx.Done():
		if t.onCanceledError != nil {
			t.onCanceledError()
		}
		e4.Throw(context.Canceled)
	default:
	}
	t.wg.Add(1)
	var doneOnce sync.Once
	return func() {
		doneOnce.Do(func() {
			t.wg.Done()
		})
	}
}

func (t *WaitTree) Wait() {
	t.wg.Wait()
}

func (t *WaitTree) Done() {
	if t.parentDone != nil {
		t.parentDone()
	}
}
