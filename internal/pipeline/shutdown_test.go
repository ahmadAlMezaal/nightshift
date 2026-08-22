package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDrainAndStop_CancelsBeforeWait(t *testing.T) {
	loopCtx, stop := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-loopCtx.Done()
	}()

	done := make(chan struct{})
	go func() {
		drainAndStop(stop, &wg)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drainAndStop deadlocked: stop() must be called before wg.Wait() " +
			"so context-bound child goroutines can leave the WaitGroup")
	}
}
