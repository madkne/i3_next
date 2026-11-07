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
	ch, cancel := bus.Subscribe[int]("t")
	defer cancel()
	if err := bus.Publish("t", 42); err != nil {
		t.Fatal(err)
	}
	if v := <-ch; v != 42 {
		t.Fatalf("got %d", v)
	}
}

func TestEssentialQueueFull(t *testing.T) {
	bus := New(WithBufferSize(1))
	ch, cancel := bus.Subscribe[int]("t")
	defer cancel()
	_ = ch
	if err := bus.Publish("t", 1, Essential()); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish("t", 2, Essential()); err == nil {
		t.Fatal("expected error")
	}
}

func TestStateCoalesce(t *testing.T) {
	bus := New(WithBufferSize(1))
	ch, cancel := bus.Subscribe[int]("t")
	defer cancel()
	for i := 0; i < 5; i++ {
		if err := bus.PublishState("t", i); err != nil {
			t.Fatal(err)
		}
	}
	v := <-ch
	if v != 4 {
		t.Fatalf("got %d want 4", v)
	}
}

func TestPanicIsolation(t *testing.T) {
	bus := New()
	ch, cancel := bus.Subscribe[int]("t")
	defer cancel()
	c2 := bus.SubscribeFunc[int]("t", func(v int) {
		if v == 1 {
			panic("boom")
		}
	})
	defer c2()
	_ = bus.Publish("t", 1)
	_ = bus.Publish("t", 2)
	select {
	case <-ch:
	case <-time.After(time.Second):
	}
}

func TestPackageLevel(t *testing.T) {
	ch, cancel := Subscribe[int]("pkg")
	defer cancel()
	if err := Publish("pkg", 7); err != nil {
		t.Fatal(err)
	}
	if v := <-ch; v != 7 {
		t.Fatalf("got %d", v)
	}
}

func BenchmarkPublish(b *testing.B) {
	bus := New(WithBufferSize(1024))
	ch, cancel := bus.Subscribe[int]("b")
	defer cancel()
	go func() {
		for range ch {
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish("b", i)
	}
}
