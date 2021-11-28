package pr

import (
	"context"
	"time"
)

type WaitTreeOption interface {
	IsWaitTreeOption()
}

type Timeout time.Duration

func (_ Timeout) IsWaitTreeOption() {}

type ID string

func (_ ID) IsWaitTreeOption() {}

type Trace bool

func (_ Trace) IsWaitTreeOption() {}

type BackgroundCtx func() context.Context

func (_ BackgroundCtx) IsWaitTreeOption() {}
