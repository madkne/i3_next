package eventbus

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestSubscribeReceive(t *testing.T) {
	bus := New()
	ch, cancel := Subscribe[int](bus, "t")
	defer cancel()
	if err := Publish(bus, "t", 42); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-ch:
		if v != 42 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestEssentialQueueFull(t *testing.T) {
	bus := New(WithBufferSize(1))
	ch, cancel := Subscribe[int](bus, "t")
	defer cancel()
	_ = ch
	if err := Publish(bus, "t", 1, Essential()); err != nil {
		t.Fatal(err)
	}
	if err := Publish(bus, "t", 2, Essential()); err == nil {
		t.Fatal("expected error")
	}
}

func TestStateCoalesce(t *testing.T) {
	bus := New(WithBufferSize(1))
	ch, cancel := Subscribe[int](bus, "t")
	defer cancel()
	for i := 0; i < 5; i++ {
		if err := PublishState(bus, "t", i); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case v := <-ch:
		if v != 4 {
			t.Fatalf("got %d want 4", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestPanicIsolation(t *testing.T) {
	bus := New()
	ch, cancel := Subscribe[int](bus, "t")
	defer cancel()
	c2 := SubscribeFunc[int](bus, "t", func(v int) {
		if v == 1 {
			panic("boom")
		}
	})
	defer c2()
	_ = Publish(bus, "t", 1)
	_ = Publish(bus, "t", 2)
	select {
	case <-ch:
	case <-time.After(time.Second):
	}
}

func TestPackageLevel(t *testing.T) {
	ch, cancel := Subscribe[int](DefaultBus, "pkg")
	defer cancel()
	if err := Publish(DefaultBus, "pkg", 7); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-ch:
		if v != 7 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestCancelDouble(t *testing.T) {
	bus := New()
	_, cancel := Subscribe[int](bus, "d")
	cancel()
	cancel() // must not panic
	bus.Close()
	bus.Close() // must not panic
}

func BenchmarkPublish(b *testing.B) {
	bus := New(WithBufferSize(1024))
	ch, cancel := Subscribe[int](bus, "b")
	defer cancel()
	go func() {
		for range ch {
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Publish(bus, "b", i)
	}
}
