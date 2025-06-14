package watcher

import (
	"sync"
	"time"
)

type debounce struct {
	after    time.Duration
	mu       *sync.Mutex
	timer    *time.Timer
	done     bool
	callback func()
}

func (d *debounce) reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.done {
		return
	}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.after, func() {
		d.callback()
	})
}

func (d *debounce) cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	d.done = true
}

// newDebounce creates a debounced instance that delays invoking functions given until after wait milliseconds have elapsed.
// Steal from: https://github.com/samber/lo
func newDebounce(duration time.Duration, f func()) (func(), func()) {
	d := &debounce{
		after:    duration,
		mu:       new(sync.Mutex),
		timer:    nil,
		done:     false,
		callback: f,
	}

	return func() {
		d.reset()
	}, d.cancel
}
