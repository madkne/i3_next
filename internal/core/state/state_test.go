package state

import (
	"sync"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestVersionMonotonic(t *testing.T) {
	s := New()
	v0 := s.Version()
	s1 := s.Update("a")
	s2 := s.Update("b")
	if s2.Version <= s1.Version || s1.Version <= v0 {
		t.Fatalf("versions not monotonic: %d %d %d", v0, s1.Version, s2.Version)
	}
	if s.Snapshot().Value != "b" {
		t.Fatal("value not updated")
	}
}

func TestSnapshotImmutable(t *testing.T) {
	s := NewState(map[string]int{"x": 1})
	snap := s.Snapshot()
	s.Update(map[string]int{"x": 2})
	if snap.Value.(map[string]int)["x"] != 1 {
		t.Fatal("snapshot mutated")
	}
}

func TestCompareAndSwap(t *testing.T) {
	s := NewState(0)
	snap := s.Snapshot()
	if _, ok := s.CompareAndSwap(snap.Version+1, 1); ok {
		t.Fatal("expected fail")
	}
	if ns, ok := s.CompareAndSwap(snap.Version, 1); !ok || ns.Value != 1 {
		t.Fatal("expected success")
	}
}

func BenchmarkStateUpdate(b *testing.B) {
	s := NewState(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Update(i)
		_ = s.Snapshot()
	}
}

func TestRace(t *testing.T) {
	s := NewState(0)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s.Update(i)
				_ = s.Snapshot()
			}
		}(i)
	}
	wg.Wait()
}
