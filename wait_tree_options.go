package pr

import "time"

type WaitTreeOption interface {
	IsWaitTreeOption()
}

type Timeout time.Duration

func (_ Timeout) IsWaitTreeOption() {}
