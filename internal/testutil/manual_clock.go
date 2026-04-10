package testutil

import (
	"sync"
	"time"
)

type ManualClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*manualWaiter
}

type manualWaiter struct {
	at time.Time
	ch chan time.Time
}

func NewManualClock(start time.Time) *ManualClock {
	return &ManualClock{now: start}
}

func (c *ManualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *ManualClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- c.now
		return ch
	}

	c.waiters = append(c.waiters, &manualWaiter{
		at: c.now.Add(d),
		ch: ch,
	})
	return ch
}

func (c *ManualClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)

	var remaining []*manualWaiter
	for _, waiter := range c.waiters {
		if !waiter.at.After(c.now) {
			waiter.ch <- c.now
			close(waiter.ch)
			continue
		}
		remaining = append(remaining, waiter)
	}
	c.waiters = remaining
	c.mu.Unlock()
}
