package rfx

import (
	"slices"
	"sync"
)

// defaultSubscriptionBuffer is how many streaming messages a subscription
// holds by default before the oldest are discarded.
const defaultSubscriptionBuffer = 64

// streamingKinds are continuously produced data, or noise, that may be
// discarded when a subscriber falls behind. Other kinds carry device state
// and are always delivered.
const streamingKinds = KindSweep | KindScreen | KindRawData | KindUnhandled | KindProtocolError

// SubscribeOptions selects what a Subscription receives.
type SubscribeOptions struct {
	// Kinds selects the messages to receive. Zero selects all.
	Kinds Kind
	// Buffer is how many sweeps, screen images and other streaming messages
	// to hold while the subscriber is busy before discarding the oldest.
	// Zero selects a default.
	Buffer int
}

// Subscription delivers messages from a Device. Each subscription is
// buffered independently, so a slow subscriber affects neither the device
// nor other subscribers.
type Subscription struct {
	dev      *Device
	kinds    Kind
	ch       chan Message
	stop     chan struct{} // closed by Close
	stopOnce sync.Once
	done     chan struct{} // closed when the forwarder has exited
	// notify receives a value when a message is queued.
	notify chan struct{}
	limit  int

	mu        sync.Mutex
	items     []Message
	streaming int
	dropped   uint64
}

// Subscribe starts delivering messages received from now on. Messages are not
// retained for later subscribers; use Info, Config and Calibration for the
// current state. Call Close on the subscription when done.
func (d *Device) Subscribe(opts SubscribeOptions) *Subscription {
	s := &Subscription{
		dev:    d,
		kinds:  opts.Kinds,
		ch:     make(chan Message),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		notify: make(chan struct{}, 1),
		limit:  opts.Buffer,
	}
	if s.kinds == 0 {
		s.kinds = KindAll
	}
	if s.limit <= 0 {
		s.limit = defaultSubscriptionBuffer
	}
	d.subsMu.Lock()
	defer d.subsMu.Unlock()
	if d.closed {
		close(s.ch)
		close(s.done)
		return s
	}
	d.subs = append(d.subs, s)
	d.wg.Add(1)
	go s.forward()
	return s
}

// C returns the channel messages are delivered on. It is closed after Close,
// after the device is closed, or when reading from the device fails; see
// Device.Err.
func (s *Subscription) C() <-chan Message {
	return s.ch
}

// Close stops delivery and closes the channel.
func (s *Subscription) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.done
}

// Dropped returns the number of messages discarded because the subscriber
// fell behind.
func (s *Subscription) Dropped() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

// push queues msg without blocking. It is called by the device's reader.
func (s *Subscription) push(msg Message) {
	if msg.Kind()&s.kinds == 0 {
		return
	}
	s.mu.Lock()
	if msg.Kind()&streamingKinds != 0 {
		if s.streaming >= s.limit {
			i := slices.IndexFunc(s.items, func(m Message) bool { return m.Kind()&streamingKinds != 0 })
			s.items = slices.Delete(s.items, i, i+1)
			s.streaming--
			s.dropped++
		}
		s.streaming++
	}
	s.items = append(s.items, msg)
	s.mu.Unlock()
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *Subscription) pop() (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return nil, false
	}
	msg := s.items[0]
	s.items[0] = nil
	s.items = s.items[1:]
	if msg.Kind()&streamingKinds != 0 {
		s.streaming--
	}
	return msg, true
}

// forward moves messages from the queue to the channel. It ends on Close, or
// once reading has stopped and the queue is drained.
func (s *Subscription) forward() {
	defer s.dev.wg.Done()
	defer close(s.done)
	defer close(s.ch)
	defer s.dev.unsubscribe(s)
	for {
		msg, ok := s.pop()
		if !ok {
			select {
			case <-s.stop:
				return
			case <-s.dev.closeCh:
				return
			case <-s.notify:
				continue
			case <-s.dev.readerDone:
				// The reader may have queued messages just before exiting.
				if msg, ok = s.pop(); !ok {
					return
				}
			}
		}
		select {
		case s.ch <- msg:
		case <-s.stop:
			return
		case <-s.dev.closeCh:
			return
		}
	}
}

// waiter receives the first message accepted by match.
type waiter struct {
	match func(Message) bool
	ch    chan Message
}

// waiters delivers messages from the reader to commands waiting on a
// response, independent of any subscriber.
type waiters struct {
	mu   sync.Mutex
	list []*waiter
}

// add registers interest in the next message accepted by match. It must be
// called before sending the command that triggers the response. match runs
// on the reader goroutine.
func (ws *waiters) add(match func(Message) bool) *waiter {
	w := &waiter{match: match, ch: make(chan Message, 1)}
	ws.mu.Lock()
	ws.list = append(ws.list, w)
	ws.mu.Unlock()
	return w
}

func (ws *waiters) remove(w *waiter) {
	ws.mu.Lock()
	if i := slices.Index(ws.list, w); i >= 0 {
		ws.list = slices.Delete(ws.list, i, i+1)
	}
	ws.mu.Unlock()
}

func (ws *waiters) deliver(msg Message) {
	ws.mu.Lock()
	ws.list = slices.DeleteFunc(ws.list, func(w *waiter) bool {
		if !w.match(msg) {
			return false
		}
		w.ch <- msg
		return true
	})
	ws.mu.Unlock()
}
