package latch

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSafeLatch_Wait_WithSetup(t *testing.T) {
	t.Run("multiple consumers waiting for resource", func(t *testing.T) {
		resource := struct {
			mu    sync.Mutex
			value int
		}{
			value: 0,
		}

		// initial resource requires setup, hence latch is "locked" created
		latch := NewLatch(true, "test-latch")
		const numConsumers = 5
		var wg sync.WaitGroup
		wg.Add(numConsumers + 1)

		// Consumer goroutines
		for i := 0; i < numConsumers; i++ {
			go func(consumerID int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if !latch.Wait(ctx, false, 5*time.Second) {
					t.Errorf("Consumer %d: timed out waiting for resosurce up", consumerID)
					return
				}
				// Access and modify the resource
				resource.mu.Lock()
				resource.value++
				resource.mu.Unlock()

				// await resource close down
				if !latch.Wait(ctx, true, 5*time.Second) {
					t.Errorf("Consumer %d: timed out waiting for resosurce down", consumerID)
					return
				}
			}(i)
		}

		// Producer goroutine
		go func() {
			defer wg.Done()
			time.Sleep(200 * time.Millisecond) // Simulate resource setup
			latch.Unlock()                     // Resource is now fully available
			time.Sleep(200 * time.Millisecond) // Simulate resource hickup
			latch.Lock()
			time.Sleep(1500 * time.Millisecond)
		}()

		wg.Wait()

		// Verify the final value
		resource.mu.Lock()
		if resource.value != numConsumers {
			t.Errorf("Expected resource value to be %d, got %d", numConsumers, resource.value)
		}
		resource.mu.Unlock()
	})
}

func TestSafeLatch_Wait_AlreadySetup(t *testing.T) {
	t.Run("multiple consumers waiting for resource", func(t *testing.T) {
		resource := struct {
			mu    sync.Mutex
			value int
		}{
			value: 0,
		}

		// initial resource ris up, hence latch is "unlocked" created
		latch := NewLatch(false, "test-latch")
		const numConsumers = 5
		var wg sync.WaitGroup
		wg.Add(numConsumers + 1)

		// Consumer goroutines
		for i := 0; i < numConsumers; i++ {
			go func(consumerID int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// already unlocked, these should return now
				now := time.Now()
				if !latch.Wait(ctx, false, 5*time.Second) {
					t.Errorf("Consumer %d: timed out waiting for resosurce up", consumerID)
					return
				}
				if time.Since(now) > 50*time.Millisecond {
					t.Errorf("Consumer %d: waitied too long for resosurce up", consumerID)
				}
				// Access and modify the resource
				resource.mu.Lock()
				resource.value++
				resource.mu.Unlock()

				// await resource close down
				if !latch.Wait(ctx, true, 5*time.Second) {
					t.Errorf("Consumer %d: timed out waiting for resosurce down", consumerID)
					return
				}
			}(i)
		}

		// Producer goroutine
		go func() {
			defer wg.Done()
			// already unlocked by creation... should be idempotent
			time.Sleep(200 * time.Millisecond)
			latch.Unlock()

			time.Sleep(200 * time.Millisecond)
			latch.Lock()
			// already locked... should be idempotent
			latch.Lock()
		}()

		wg.Wait()

		// Verify the final value
		resource.mu.Lock()
		if resource.value != numConsumers {
			t.Errorf("Expected resource value to be %d, got %d", numConsumers, resource.value)
		}
		resource.mu.Unlock()
	})
}
