package eventbus

import (
	"testing"
	"time"
)

func TestLoad10k100Subs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	bus := New(WithBufferSize(1024))
	const subs = 100
	cancels := make([]func(), subs)
	for i := 0; i < subs; i++ {
		ch, c := bus.Subscribe[int]("load")
		cancels[i] = c
		go func(ch <-chan int) {
			for range ch {
			}
		}(ch)
	}

	start := time.Now()
	const n = 10000
	for i := 0; i < n; i++ {
		_ = bus.Publish("load", i)
	}
	elapsed := time.Since(start)
	t.Logf("published %d events to %d subs in %v", n, subs, elapsed)

	// allow drain
	time.Sleep(100 * time.Millisecond)
	for _, c := range cancels {
		c()
	}
}
