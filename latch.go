package latch

import (
	"context"
	"sync"
	"time"
)

// NotifyingLatch is a concurrency safe latch.
type NotifyingLatch struct {
	onLock   chan struct{} // notifies consumers of lock events
	onUnlock chan struct{} // notifies consumers of unlock events
	mu       sync.RWMutex  // protects channel and status
	locked   bool          // current latch position
	tag      string        // helper for tracing
}

// NewLatch creates a new SafeLatch instance.
// Locked status indicates the protected resource is not available for consumption and
// clients should AwaitUnlock on it before resource access.
func NewLatch(locked bool, tag string) NotifyingLatch {
	if locked {
		return NotifyingLatch{
			onUnlock: make(chan struct{}),
			locked:   locked,
			tag:      tag,
		}
	} else {
		return NotifyingLatch{
			onLock: make(chan struct{}),
			locked: locked,
			tag:    tag,
		}
	}
}

// Lock closes the latch restricting access to the protected resource.
// idempotent: if already locked it does nothing
func (s *NotifyingLatch) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.locked {
		return
	}

	close(s.onLock)                  // raise locked event
	s.onUnlock = make(chan struct{}) // prepare unlock listeners
	s.locked = true
}

// Unlock opens the latch permitting access to the protected resource.
// idempotent: if already unlocked it does nothing
func (s *NotifyingLatch) Unlock() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.locked {
		return
	}

	close(s.onUnlock)              // raise unlocked event
	s.onLock = make(chan struct{}) // prepare lock listeners
	s.locked = false
}

// Locked returns true if the current status is locked.
func (s *NotifyingLatch) Locked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.locked
}

// Wait locks till the wantLocked status is raised, timeout expires or context is cancelled.
// Returns immediateley when the latch is already in the wantLocked status.
// Returns true if wantLocked was achieved or false on cancellations.
func (s *NotifyingLatch) Wait(ctx context.Context, wantLocked bool, timeout time.Duration) bool {
	var ch chan struct{}

	// we need a safe read of the latest knonw snap-shot for ch polling
	s.mu.RLock()
	if wantLocked {
		ch = s.onLock
	} else {
		ch = s.onUnlock
	}
	status := s.locked
	s.mu.RUnlock()

	if status != wantLocked {
		select {
		case <-ch:
			return true
		case <-time.After(timeout):
			return false
		case <-ctx.Done():
			return false
		}
	}
	return true
}
