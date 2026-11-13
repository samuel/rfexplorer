package rfx

import (
	"slices"
	"sync"
)

// maxQueuedStreamPackets is how many streaming packets are held for a slow
// consumer before the oldest are discarded.
const maxQueuedStreamPackets = 64

// packetQueue buffers packets between the reader and the consumer so that a
// slow consumer never stalls the reader. Streaming packets are bounded by
// discarding the oldest. Packets carrying device state are always kept.
type packetQueue struct {
	// notify receives a value when a packet is pushed onto an empty queue.
	notify chan struct{}

	mu        sync.Mutex
	items     []Packet
	streaming int
	dropped   uint64
}

func newPacketQueue() *packetQueue {
	return &packetQueue{notify: make(chan struct{}, 1)}
}

// isStreaming reports whether pkt is continuously produced data, or noise,
// that may be discarded when the consumer falls behind.
func isStreaming(pkt Packet) bool {
	switch pkt.(type) {
	case *SweepDataPacket, *ScreenImage, *RawData, *UnhandledPacket, *DecodeErrorPacket:
		return true
	}
	return false
}

func (q *packetQueue) push(pkt Packet) {
	q.mu.Lock()
	if isStreaming(pkt) {
		if q.streaming >= maxQueuedStreamPackets {
			i := slices.IndexFunc(q.items, isStreaming)
			q.items = slices.Delete(q.items, i, i+1)
			q.streaming--
			q.dropped++
		}
		q.streaming++
	}
	q.items = append(q.items, pkt)
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *packetQueue) pop() (Packet, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	pkt := q.items[0]
	q.items[0] = nil
	q.items = q.items[1:]
	if isStreaming(pkt) {
		q.streaming--
	}
	return pkt, true
}

func (q *packetQueue) droppedCount() uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.dropped
}

// waiter receives the first packet accepted by match.
type waiter struct {
	match func(Packet) bool
	ch    chan Packet
}

// waiters delivers packets from the reader to callers waiting on a response,
// independent of the consumer draining the packet channel.
type waiters struct {
	mu   sync.Mutex
	list []*waiter
}

// add registers interest in the next packet accepted by match. It must be
// called before sending the command that triggers the response.
func (ws *waiters) add(match func(Packet) bool) *waiter {
	w := &waiter{match: match, ch: make(chan Packet, 1)}
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

func (ws *waiters) deliver(pkt Packet) {
	ws.mu.Lock()
	ws.list = slices.DeleteFunc(ws.list, func(w *waiter) bool {
		if !w.match(pkt) {
			return false
		}
		w.ch <- pkt
		return true
	})
	ws.mu.Unlock()
}
