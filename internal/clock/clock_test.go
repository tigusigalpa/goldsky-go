package clock

import (
	"context"
	"testing"
	"time"
)

func TestSystemClockNow(t *testing.T) {
	before := time.Now()
	got := (SystemClock{}).Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %s, want between %s and %s", got, before, after)
	}
}

func TestSystemSleeper(t *testing.T) {
	s := SystemSleeper{}
	if err := s.Sleep(context.Background(), 0); err != nil {
		t.Fatalf("zero sleep: %v", err)
	}
	if err := s.Sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("timed sleep: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Sleep(ctx, time.Hour); err != context.Canceled {
		t.Fatalf("cancelled sleep = %v, want context.Canceled", err)
	}
}

func TestFakeClock(t *testing.T) {
	start := time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC)
	c := NewFakeClock(start)
	c.Advance(5 * time.Second)
	if got := c.Now(); got != start.Add(5*time.Second) {
		t.Fatalf("Now() = %s", got)
	}
}

func TestFakeSleeper(t *testing.T) {
	s := &FakeSleeper{}
	if err := s.Sleep(context.Background(), 2*time.Second); err != nil {
		t.Fatalf("Sleep: %v", err)
	}
	if len(s.Sleeps) != 1 || s.Sleeps[0] != 2*time.Second {
		t.Fatalf("sleeps = %v", s.Sleeps)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Sleep(ctx, time.Second); err != context.Canceled {
		t.Fatalf("cancelled Sleep = %v", err)
	}
}
