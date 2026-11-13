package rfx

import "time"

// deviceRxBuffer is the size of the device's receive buffer. Measured on a
// 6G + WSUB3G unit running firmware 1.34: commands beyond 64 pending bytes are
// lost and leave the device's command parser out of step.
const deviceRxBuffer = 64

// commandTimeout is how long a command is assumed to stay in the device's
// buffer when no sweeps are being received, as in hold mode.
var commandTimeout = 3 * time.Second

// txWindow tracks commands the device may not have consumed yet, so that
// writes never overflow its receive buffer. The device handles about one
// command per sweep.
type txWindow struct {
	pending   []sentCommand
	bytes     int
	lastSweep time.Time
}

type sentCommand struct {
	size int
	at   time.Time
}

// reserve records a command of size bytes sent at now. If it would overflow
// the device's buffer, nothing is recorded and wait is the longest it can
// take for room to become available.
func (w *txWindow) reserve(size int, now time.Time) (wait time.Duration, ok bool) {
	// Without sweeps to count, fall back to assuming commands are consumed
	// after a while.
	if now.Sub(w.lastSweep) >= commandTimeout {
		for len(w.pending) > 0 && now.Sub(w.pending[0].at) >= commandTimeout {
			w.retire()
		}
	}
	if len(w.pending) > 0 && w.bytes+size > deviceRxBuffer {
		return commandTimeout, false
	}
	w.pending = append(w.pending, sentCommand{size: size, at: now})
	w.bytes += size
	return 0, true
}

// sweep records that a sweep was received at now. The oldest command is
// consumed once a whole sweep has been received since it was sent.
func (w *txWindow) sweep(now time.Time) {
	if len(w.pending) > 0 && !w.lastSweep.IsZero() && w.pending[0].at.Before(w.lastSweep) {
		w.retire()
	}
	w.lastSweep = now
}

func (w *txWindow) retire() {
	w.bytes -= w.pending[0].size
	w.pending = w.pending[1:]
}
