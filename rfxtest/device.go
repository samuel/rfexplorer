// Package rfxtest provides an in-memory stand-in for an RF Explorer serial
// port, for testing code that uses package rfx without hardware.
package rfxtest

import (
	"io"
	"sync"
)

// Handler responds to a command written by the host. cmd is the command
// without its "#" and length prefix. The returned bytes are sent to the host.
type Handler func(cmd []byte) []byte

// Device is a scripted RF Explorer. It implements io.ReadWriteCloser.
type Device struct {
	mu       sync.Mutex
	cond     *sync.Cond
	handler  Handler
	out      []byte // bytes waiting to be read by the host
	in       []byte // partial command written by the host
	commands [][]byte
	err      error
	closed   bool
}

// NewDevice returns a device that answers commands using handler. A nil
// handler never answers.
func NewDevice(handler Handler) *Device {
	d := &Device{handler: handler}
	d.cond = sync.NewCond(&d.mu)
	return d
}

// Read blocks until the device has sent data, fails, or is closed.
func (d *Device) Read(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for len(d.out) == 0 {
		if d.closed {
			return 0, io.ErrClosedPipe
		}
		if d.err != nil {
			return 0, d.err
		}
		d.cond.Wait()
	}
	n := copy(p, d.out)
	d.out = d.out[n:]
	return n, nil
}

// Write accepts host commands. Each complete command is recorded and passed
// to the handler.
func (d *Device) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return 0, io.ErrClosedPipe
	}
	d.in = append(d.in, p...)
	for len(d.in) >= 2 {
		// "#" length:byte payload, where length counts the whole command.
		n := int(d.in[1])
		if d.in[0] != '#' || n < 2 {
			// Not a command. Record the bytes so tests can detect it.
			d.commands = append(d.commands, d.in)
			d.in = nil
			break
		}
		if len(d.in) < n {
			break
		}
		cmd := append([]byte(nil), d.in[2:n]...)
		d.in = d.in[n:]
		d.commands = append(d.commands, cmd)
		if d.handler != nil {
			d.out = append(d.out, d.handler(cmd)...)
		}
	}
	d.cond.Broadcast()
	return len(p), nil
}

// Close unblocks Read and fails further I/O.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	d.cond.Broadcast()
	return nil
}

// Send queues bytes for the host to read, as if sent by the device.
func (d *Device) Send(b []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.out = append(d.out, b...)
	d.cond.Broadcast()
}

// Fail makes Read return err once all queued data has been read.
func (d *Device) Fail(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.err = err
	d.cond.Broadcast()
}

// Commands returns the commands received so far, without their "#" and
// length prefix.
func (d *Device) Commands() [][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([][]byte(nil), d.commands...)
}
