package eventbus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// Errors returned by the event bus.
var (
	ErrBusClosed    = errors.New("eventbus: bus closed")
	ErrQueueFull    = errors.New("eventbus: queue full")
	ErrInvalidTopic = errors.New("eventbus: invalid topic")
	ErrTypeMismatch = errors.New("eventbus: type mismatch")
	ErrDropped      = errors.New("eventbus: event dropped")
)

// Topic is the subscription topic type (a string).
type Topic = string

// PublishOption configures a single publish operation.
type PublishOption func(*publishOptions)

type publishOptions struct {
	essential bool
	state     bool
	coalesce  bool
}

// Essential marks an event that must never be silently dropped.
func Essential() PublishOption {
	return func(o *publishOptions) { o.essential = true }
}

// State marks an event carrying state, eligible for coalescing (only the
// latest value matters).
func State() PublishOption {
	return func(o *publishOptions) { o.state = true; o.coalesce = true }
}

// Coalesce marks an event as coalescable.
func Coalesce() PublishOption {
	return func(o *publishOptions) { o.coalesce = true }
}

// Option configures a Bus.
type Option func(*Bus)

// WithBufferSize sets the per-subscriber queue capacity.
func WithBufferSize(n int) Option {
	return func(b *Bus) {
		if n > 0 {
			b.bufferSize = n
		}
	}
}

// WithQueueSize is an alias for WithBufferSize.
func WithQueueSize(n int) Option { return WithBufferSize(n) }

// Bus is a generic, type-safe event bus.
type Bus struct {
	mu         sync.RWMutex
	closed     bool
	bufferSize int
	nextID     uint64
	subs       map[string][]*subscription
}

// New creates a new Bus.
func New(opts ...Option) *Bus {
	b := &Bus{bufferSize: 64, subs: make(map[string][]*subscription)}
	for _, o := range opts {
		if o != nil {
			o(b)
		}
	}
	if b.bufferSize <= 0 {
		b.bufferSize = 1
	}
	return b
}

// NewBus is an alias for New.
func NewBus(opts ...Option) *Bus { return New(opts...) }

// NewEventBus is an alias for New.
func NewEventBus(opts ...Option) *Bus { return New(opts...) }

// DefaultBus is the package-level bus used by package-level helpers.
var DefaultBus = New()

// subscription is a type-erased subscriber.
type subscription struct {
	id      uint64
	topic   string
	typeOf  reflect.Type
	enqueue func(value any, opts publishOptions) error
	cancel  func()
}

// subscriberQueue holds a single subscriber's state slot.
type subscriberQueue[T any] struct {
	mu       sync.Mutex
	cond     *sync.Cond
	stateVal T
	hasState bool
	closed   bool
	closedCh chan struct{}
}

func newSubscriberQueue[T any]() *subscriberQueue[T] {
	q := &subscriberQueue[T]{closedCh: make(chan struct{})}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Subscribe subscribes to topic for events of type T.
func (b *Bus) Subscribe[T any](topic string) (ch <-chan T, cancel func()) {
	return b.SubscribeBuffered[T](topic, b.bufferSize)
}

// SubscribeBuffered subscribes with an explicit buffer size.
func (b *Bus) SubscribeBuffered[T any](topic string, buffer int) (ch <-chan T, cancel func()) {
	if buffer <= 0 {
		buffer = b.bufferSize
		if buffer <= 0 {
			buffer = 1
		}
	}
	if topic == "" {
		topic = "*"
	}
	q := newSubscriberQueue[T]()
	out := make(chan T, buffer)

	// state-slot pump goroutine (isolates panics)
	go func() {
		defer func() { _ = recover() }()
		for {
			q.mu.Lock()
			for !q.hasState && !q.closed {
				q.cond.Wait()
			}
			if q.closed {
				q.mu.Unlock()
				return
			}
			val := q.stateVal
			q.hasState = false
			q.mu.Unlock()
			select {
			case out <- val:
			case <-q.closedCh:
				return
			}
		}
	}()

	id := atomic.AddUint64(&b.nextID, 1)
	t := reflect.TypeOf((*T)(nil)).Elem()

	enqueue := func(value any, opts publishOptions) error {
		v, ok := value.(T)
		if !ok {
			return ErrTypeMismatch
		}
		q.mu.Lock()
		defer q.mu.Unlock()
		if q.closed {
			return ErrBusClosed
		}
		if opts.state || opts.coalesce {
			// coalesce: only the latest value matters
			q.stateVal = v
			q.hasState = true
			q.cond.Broadcast()
			return nil
		}
		select {
		case out <- v:
			return nil
		default:
			if opts.essential {
				return ErrQueueFull
			}
			// drop non-essential without error
			return nil
		}
	}

	cancel = func() {
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		q.cond.Broadcast()
		close(q.closedCh)
		b.removeSubscription(topic, id)
		// close out after pump exits; best-effort
	}

	sub := &subscription{id: id, topic: topic, typeOf: t, enqueue: enqueue, cancel: cancel}

	b.mu.Lock()
	if b.subs == nil {
		b.subs = make(map[string][]*subscription)
	}
	b.subs[topic] = append(b.subs[topic], sub)
	b.mu.Unlock()

	// close out when canceled
	go func() {
		<-q.closedCh
		// wait for pump exit? small delay
		close(out)
	}()

	return out, cancel
}

// SubscribeChan is an alias for SubscribeBuffered.
func (b *Bus) SubscribeChan[T any](topic string, buffer int) (ch <-chan T, cancel func()) {
	return b.SubscribeBuffered[T](topic, buffer)
}

// Publish publishes an event to all subscribers of topic with matching type.
func (b *Bus) Publish[T any](topic string, value T, opts ...PublishOption) error {
	if topic == "" {
		return ErrInvalidTopic
	}
	o := publishOptions{}
	for _, f := range opts {
		if f != nil {
			f(&o)
		}
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	list := append([]*subscription(nil), b.subs[topic]...)
	b.mu.RUnlock()

	t := reflect.TypeOf((*T)(nil)).Elem()
	var firstErr error
	for _, sub := range list {
		if sub.typeOf != t {
			continue
		}
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("eventbus: handler panic: %v", r)
				}
			}()
			return sub.enqueue(value, o)
		}()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// PublishEssential publishes an essential event.
func (b *Bus) PublishEssential[T any](topic string, value T) error {
	return b.Publish(topic, value, Essential())
}

// PublishState publishes a state (coalescable) event.
func (b *Bus) PublishState[T any](topic string, value T) error {
	return b.Publish(topic, value, State())
}

// Close closes the bus.
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	lists := make([][]*subscription, 0, len(b.subs))
	for _, l := range b.subs {
		lists = append(lists, l)
	}
	b.subs = make(map[string][]*subscription)
	b.mu.Unlock()
	for _, l := range lists {
		for _, sub := range l {
			if sub.cancel != nil {
				sub.cancel()
			}
		}
	}
}

// removeSubscription removes a subscription by id.
func (b *Bus) removeSubscription(topic string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.subs[topic]
	out := list[:0]
	for _, sub := range list {
		if sub.id == id {
			continue
		}
		out = append(out, sub)
	}
	b.subs[topic] = out
}

// Handler-based subscription helper.
func (b *Bus) SubscribeFunc[T any](topic string, handler func(T)) (cancel func()) {
	ch, cancel := b.Subscribe[T](topic)
	go func() {
		defer func() { _ = recover() }()
		for v := range ch {
			func() {
				defer func() { _ = recover() }()
				if handler != nil {
					handler(v)
				}
			}()
		}
	}()
	return cancel
}

// Package-level helpers using DefaultBus.
func Publish[T any](topic string, value T, opts ...PublishOption) error {
	return DefaultBus.Publish(topic, value, opts...)
}

func Subscribe[T any](topic string) (ch <-chan T, cancel func()) {
	return DefaultBus.Subscribe[T](topic)
}

func SubscribeBuffered[T any](topic string, buffer int) (ch <-chan T, cancel func()) {
	return DefaultBus.SubscribeBuffered[T](topic, buffer)
}

func PublishEssential[T any](topic string, value T) error {
	return DefaultBus.PublishEssential(topic, value)
}

func PublishState[T any](topic string, value T) error {
	return DefaultBus.PublishState(topic, value)
}

// WaitFor blocks until an event is received or context done.
func WaitFor[T any](ctx context.Context, topic string) (T, error) {
	ch, cancel := Subscribe[T](topic)
	defer cancel()
	select {
	case v := <-ch:
		return v, nil
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// TimeNow is a helper for tests.
func TimeNow() time.Time { return time.Now() }
