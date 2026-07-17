package godoors

import (
	"testing"
	"time"
)

// withIdle sets Idle/IdleAction for a test and restores them afterward.
func withIdle(t *testing.T, seconds int, action func()) {
	t.Helper()
	origIdle, origAction := Idle, IdleAction
	t.Cleanup(func() { Idle, IdleAction = origIdle, origAction })
	Idle, IdleAction = seconds, action
}

func TestIdleActionOverrideFires(t *testing.T) {
	fired := make(chan struct{}, 1)
	withIdle(t, 1, func() { fired <- struct{}{} })

	stop := startIdleTimer()
	defer stop()

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("custom IdleAction did not fire")
	}
}

func TestIdleTimerStopPreventsAction(t *testing.T) {
	fired := make(chan struct{}, 1)
	withIdle(t, 1, func() { fired <- struct{}{} })

	stop := startIdleTimer()
	stop() // cancel before it can fire

	select {
	case <-fired:
		t.Fatal("IdleAction fired after the timer was stopped")
	case <-time.After(1500 * time.Millisecond):
		// expected: no fire
	}
}

func TestIdleDisabledNeverFires(t *testing.T) {
	fired := make(chan struct{}, 1)
	withIdle(t, 0, func() { fired <- struct{}{} })

	stop := startIdleTimer()
	defer stop()

	select {
	case <-fired:
		t.Fatal("IdleAction fired even though Idle was 0")
	case <-time.After(200 * time.Millisecond):
		// expected: disabled
	}
}
