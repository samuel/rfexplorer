package rfx

import (
	"testing"
	"time"
)

func TestTxWindow(t *testing.T) {
	defer func(d time.Duration) { commandTimeout = d }(commandTimeout)
	commandTimeout = 2 * time.Second

	var w txWindow
	now := time.Unix(1000, 0)
	tick := func(d time.Duration) { now = now.Add(d) }

	// Sixteen 4-byte commands fill the device buffer exactly.
	for i := range 16 {
		if _, ok := w.reserve(4, now); !ok {
			t.Fatalf("command %d refused with %d bytes pending", i, w.bytes)
		}
		tick(time.Millisecond)
	}
	wait, ok := w.reserve(4, now)
	if ok {
		t.Fatal("command accepted beyond the device buffer")
	}
	if wait != commandTimeout {
		t.Errorf("wait %v out of range", wait)
	}

	// The sweep in progress when a command was sent doesn't consume it.
	w.sweep(now)
	if _, ok := w.reserve(4, now); ok {
		t.Fatal("command accepted after a partial sweep")
	}
	// Each further sweep consumes one command.
	tick(200 * time.Millisecond)
	w.sweep(now)
	if _, ok := w.reserve(4, now); !ok {
		t.Fatal("command refused after a full sweep")
	}
	if _, ok := w.reserve(4, now); ok {
		t.Fatal("more than one command consumed per sweep")
	}

	// While sweeps keep arriving, commands are only consumed by them, however
	// long they have been pending.
	for range 10 {
		tick(commandTimeout / 4)
		w.sweep(now)
	}
	if got, want := len(w.pending), 16-10; got != want {
		t.Errorf("%d commands pending after 10 sweeps, want %d", got, want)
	}

	// Without sweeps, commands time out.
	tick(commandTimeout)
	if _, ok := w.reserve(deviceRxBuffer, now); !ok {
		t.Fatal("command refused after pending commands timed out")
	}
	if w.bytes != deviceRxBuffer || len(w.pending) != 1 {
		t.Errorf("got %d bytes in %d commands pending", w.bytes, len(w.pending))
	}

	// A command larger than the buffer is still sent once nothing is pending.
	tick(commandTimeout)
	if _, ok := w.reserve(deviceRxBuffer+10, now); !ok {
		t.Fatal("oversized command refused with nothing pending")
	}
}
