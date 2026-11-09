package eventbus

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
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
	once     sync.Once
	closedCh chan struct{}
}

func newSubscriberQueue[T any]() *subscriberQueue[T] {
	q := &subscriberQueue[T]{closedCh: make(chan struct{})}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *subscriberQueue[T]) close() {
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		q.cond.Broadcast()
		close(q.closedCh)
	})
}

// Subscribe subscribes to topic for events of type T.
func Subscribe[T any](b *Bus, topic string) (ch <-chan T, cancel func()) {
	buf := b.bufferSize
	return SubscribeBuffered[T](b, topic, buf)
}

// SubscribeBuffered subscribes with an explicit buffer size.
func SubscribeBuffered[T any](b *Bus, topic string, buffer int) (ch <-chan T, cancel func()) {
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

	// state-slot pump goroutine (isolates panics). It owns closing out.
	go func() {
		defer func() { _ = recover() }()
		defer close(out)
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
		if q.closed {
			q.mu.Unlock()
			return ErrBusClosed
		}
		if opts.state || opts.coalesce {
			// coalesce: only the latest value matters
			q.stateVal = v
			q.hasState = true
			q.mu.Unlock()
			q.cond.Broadcast()
			return nil
		}
		q.mu.Unlock()
		// Non-coalesced: enqueue checks closed state before send; pump owns
		// closing out after observing closed, so no send races with close.
		select {
		case <-q.closedCh:
			return ErrBusClosed
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

	var cancelOnce sync.Once
	cancel = func() {
		cancelOnce.Do(func() {
			q.close()
			b.removeSubscription(topic, id)
		})
	}

	sub := &subscription{id: id, topic: topic, typeOf: t, enqueue: enqueue, cancel: cancel}

	b.mu.Lock()
	if b.subs == nil {
		b.subs = make(map[string][]*subscription)
	}
	b.subs[topic] = append(b.subs[topic], sub)
	b.mu.Unlock()

	return out, cancel
}

// Publish publishes an event to all subscribers of topic with matching type.
func Publish[T any](b *Bus, topic string, value T, opts ...PublishOption) error {
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
func PublishEssential[T any](b *Bus, topic string, value T) error {
	return Publish[T](b, topic, value, Essential())
}

// PublishState publishes a state (coalescable) event.
func PublishState[T any](b *Bus, topic string, value T) error {
	return Publish[T](b, topic, value, State())
}

// Close closes the bus.
func (b *Bus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
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

// SubscribeFunc subscribes and invokes handler for each event.
func SubscribeFunc[T any](b *Bus, topic string, handler func(T)) (cancel func()) {
	ch, cancel := Subscribe[T](b, topic)
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
