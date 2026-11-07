package state

import (
	"sync"
	"sync/atomic"
	"time"
)

// Version is a monotonic state version.
type Version uint64

// Snapshot is an immutable point-in-time view of state.
type Snapshot struct {
	Version   Version   `json:"version"`
	Value     any       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// Data is an alias for the snapshot's value field access.
func (s Snapshot) Data() any { return s.Value }

// State is a thread-safe, versioned state container.
type State struct {
	mu        sync.RWMutex
	version   Version
	current   Snapshot
	listeners []func(Snapshot)
}

// Store is an alias for State.
type Store = State

// New creates a new State with a nil initial value.
func New() *State { return NewState(nil) }

// NewState creates a new State with the given initial value.
func NewState(initial any) *State {
	s := &State{}
	s.current = Snapshot{Version: 0, Value: initial, CreatedAt: time.Now()}
	s.version = 0
	return s
}

// NewStore is an alias for NewState.
func NewStore(initial any) *State { return NewState(initial) }

// Snapshot returns the current immutable snapshot.
func (s *State) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Version returns the current monotonic version.
func (s *State) Version() Version {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Current is an alias for Snapshot.
func (s *State) Current() Snapshot { return s.Snapshot() }

// Get is an alias for Snapshot.
func (s *State) Get() Snapshot { return s.Snapshot() }

// Value returns the current value.
func (s *State) Value() any { return s.Snapshot().Value }

// Set updates the state value and increments the version.
func (s *State) Set(value any) Snapshot { return s.Update(value) }

// Update updates the state value and increments the version.
func (s *State) Update(value any) Snapshot {
	if s == nil {
		return Snapshot{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version++
	s.current = Snapshot{Version: s.version, Value: value, CreatedAt: time.Now()}
	s.notifyLocked(s.current)
	return s.current
}

// Mutate applies fn to the current value and stores the result.
func (s *State) Mutate(fn func(any) any) Snapshot {
	if s == nil || fn == nil {
		return s.Snapshot()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	newVal := fn(s.current.Value)
	s.version++
	s.current = Snapshot{Version: s.version, Value: newVal, CreatedAt: time.Now()}
	s.notifyLocked(s.current)
	return s.current
}

// CompareAndSwap updates the value only if the version matches.
func (s *State) CompareAndSwap(old Version, value any) (Snapshot, bool) {
	if s == nil {
		return Snapshot{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.version != old {
		return s.current, false
	}
	s.version++
	s.current = Snapshot{Version: s.version, Value: value, CreatedAt: time.Now()}
	s.notifyLocked(s.current)
	return s.current, true
}

// OnChange registers a listener for state changes.
func (s *State) OnChange(fn func(Snapshot)) {
	if s == nil || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

func (s *State) notifyLocked(snap Snapshot) {
	listeners := append([]func(Snapshot){}, s.listeners...)
	go func() {
		for _, fn := range listeners {
			func() {
				defer func() { _ = recover() }()
				fn(snap)
			}()
		}
	}()
}

// Reset resets state to initial with version 0.
func (s *State) Reset(initial any) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = 0
	s.current = Snapshot{Version: 0, Value: initial, CreatedAt: time.Now()}
}

// NextVersion returns the next monotonic version without updating state.
func (s *State) NextVersion() Version {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version + 1
}

// Atomic counter helper for global versions.
var globalVersion uint64

// NextGlobalVersion returns a process-global monotonic version.
func NextGlobalVersion() Version {
	return Version(atomic.AddUint64(&globalVersion, 1))
}

// Snapshotter is implemented by versioned state containers.
type Snapshotter interface {
	Snapshot() Snapshot
	Version() Version
}
