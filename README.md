# go-notifying-latch

`go-notifying-latch` is a Go library that provides a concurrency-safe latch mechanism. It allows you to control access to a shared resource by locking and unlocking the latch. Consumers can wait for the latch to be in a specific state (locked or unlocked) with a timeout and context support.

## Functionality

The `NotifyingLatch` struct provides the following methods:

*   `NewSafeLatch(...)`: Creates a new `NotifyingLatch` instance. The `locked` parameter indicates the initial state of the latch. The `tag` parameter is a helper for tracing.
*   `Lock()`: Closes the latch, signaling restricted access to the protected resource. It is idempotent, so if the latch is already locked, it does nothing.
*   `Unlock()`: Opens the latch, signaling permitted access to the protected resource. It is idempotent, so if the latch is already unlocked, it does nothing.
*   `Locked()`: Returns `true` if the latch is currently locked.
*   `Wait(...)`: Waits until the latch reaches the desired `wantLocked` state (either locked or unlocked), the timeout expires, or the context is canceled. It returns `true` if the desired state was achieved, or `false` if the wait was canceled or timed out.

## Example

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	latch "github.com/LucaWolf/go-notifying-latch"
)

func main() {
	// Create a new NotifyingLatch instance, initially locked.
	latch := latch.NewSafeLatch(true, "resource-latch")

	// Simulate a resource that requires setup.
	var resource struct {
		mu    sync.Mutex
		value int
	}

	// Simulate multiple consumers waiting for the resource.
	const numConsumers = 3
	var wg sync.WaitGroup
	wg.Add(numConsumers + 1)

	for i := 0; i < numConsumers; i++ {
		go func(consumerID int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Wait for the latch to be unlocked (resource available).
			if !latch.Wait(ctx, false, 5*time.Second) {
				fmt.Printf("Consumer %d: timed out waiting for resource up\\n", consumerID)
				return
			}

			// Access and modify the resource.
			resource.mu.Lock()
			resource.value++
			fmt.Printf("Consumer %d: accessed resource, value = %d\\n", consumerID, resource.value)
			resource.mu.Unlock()

			// Wait for the latch to be locked again (resource unavailable).
			if !latch.Wait(ctx, true, 5*time.Second) {
				fmt.Printf("Consumer %d: timed out waiting for resource down\\n", consumerID)
				return
			}
			fmt.Printf("Consumer %d: resource is now down\\n", consumerID)
		}(i)
	}

	// Simulate a producer setting up and tearing down the resource.
	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond) // Simulate resource setup
		fmt.Println("Resource is now available")
		latch.Unlock() // Resource is now fully available
		time.Sleep(200 * time.Millisecond) // Simulate resource hickup
		latch.Lock()
		fmt.Println("Resource is now unavailable")// Simulate resource teardown
		time.Sleep(1500 * time.Millisecond)
	}()

	wg.Wait()

	// Verify the final value.
	resource.mu.Lock()
	fmt.Printf("Final resource value: %d\\n", resource.value)
	resource.mu.Unlock()
}
```

## Usage

1.  Import the package:

    ```go
    import latch "github.com/LucaWolf/go-notifying-latch"
    ```

2.  Create a new `NotifyingLatch` instance:

    ```go
    latch := latch.NewSafeLatch(true, "my-latch")
    ```

3.  Use the `Lock()` and `Unlock()` methods to control access to the protected resource.

4.  Use the `Wait()` method to wait for the latch to be in a specific state.

## License

This library is licensed under the [MIT License](LICENSE).