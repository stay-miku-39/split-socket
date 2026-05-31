package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

type Timer struct {
	closed    chan struct{}
	reset     chan time.Duration
	timer     *time.Timer
	callback  func()
	fired     atomic.Bool
	closedFlg atomic.Bool
	once      sync.Once
}

func NewTimer(callback func()) *Timer {
	t := &Timer{
		closed:   make(chan struct{}),
		reset:    make(chan time.Duration),
		timer:    time.NewTimer(time.Hour),
		callback: callback,
	}
	go t.run()
	return t
}

func (t *Timer) run() {
	for {
		select {
		case <-t.closed:
			return
		case d := <-t.reset:
			t.timer.Stop()
			select {
			case <-t.timer.C:
			default:
			}
			t.timer.Reset(d)
		case <-t.timer.C:
			t.callback()
			t.fired.Store(true)
			t.Close()
			return
		}
	}
}

func (t *Timer) Reset(d time.Duration) {
	if t.IsClosed() {
		return
	}
	t.reset <- d
}

func (t *Timer) Close() {
	t.once.Do(func() {
		close(t.closed)
		close(t.reset)
		t.timer.Stop()
		t.closedFlg.Store(true)
	})
}

func (t *Timer) HasFired() bool {
	return t.fired.Load()
}

func (t *Timer) IsClosed() bool {
	return t.closedFlg.Load()
}
