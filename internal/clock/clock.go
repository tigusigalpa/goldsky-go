// Package clock provides injectable time and sleeping primitives so that retry
// and backoff behavior can be made deterministic in tests without real delays.
package clock

import (
	"context"
	"time"
)

// Clock returns the current time. Production code uses the system clock; tests
// inject a fake to advance time deterministically.
type Clock interface {
	Now() time.Time
}

// Sleeper blocks the calling goroutine for the supplied duration, honouring
// context cancellation. Tests inject a fake that records requested sleeps
// without blocking.
type Sleeper interface {
	Sleep(ctx context.Context, d time.Duration) error
}

// SystemClock is the Clock backed by time.Now.
type SystemClock struct{}

// Now returns the current system time.
func (SystemClock) Now() time.Time { return time.Now() }

// SystemSleeper is the Sleeper backed by time.Timer, context-aware.
type SystemSleeper struct{}

// Sleep blocks for d or until ctx is cancelled.
func (SystemSleeper) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// FakeClock is a Clock whose current time is controlled by Advance.
type FakeClock struct {
	t time.Time
}

// NewFakeClock returns a FakeClock set to t.
func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{t: t} }

// Now returns the clock's current time.
func (c *FakeClock) Now() time.Time { return c.t }

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

// FakeSleeper is a Sleeper that records requested sleeps instead of blocking.
type FakeSleeper struct {
	Sleeps []time.Duration
}

// Sleep records the requested duration without blocking. It returns ctx.Err()
// if ctx is already done, so tests can exercise cancellation.
func (s *FakeSleeper) Sleep(ctx context.Context, d time.Duration) error {
	s.Sleeps = append(s.Sleeps, d)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
