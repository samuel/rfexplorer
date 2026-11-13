// Package record captures the serial traffic of an RF Explorer to a file and
// plays it back as a stand-in serial port, so frontends can be developed and
// regressions reproduced without hardware.
//
// A recording starts with the line "RFXREC1\n" followed by one record per
// read or write: a direction byte ('<' from the device, '>' to the device),
// the time since the recording started in nanoseconds and the data length as
// unsigned varints, then the data.
package record

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const header = "RFXREC1\n"

// Directions of a record.
const (
	FromDevice byte = '<'
	ToDevice   byte = '>'
)

// Recorder is a serial port that records the traffic of the port it wraps.
type Recorder struct {
	port  io.ReadWriteCloser
	start time.Time

	mu  sync.Mutex
	w   *bufio.Writer
	c   io.Closer
	err error
}

// NewRecorder records the traffic of port to w. Closing the recorder closes
// port, and w if it is an io.Closer.
func NewRecorder(port io.ReadWriteCloser, w io.Writer) (*Recorder, error) {
	r := &Recorder{port: port, start: time.Now(), w: bufio.NewWriter(w)}
	if c, ok := w.(io.Closer); ok {
		r.c = c
	}
	if _, err := r.w.WriteString(header); err != nil {
		return nil, fmt.Errorf("record: writing header: %w", err)
	}
	return r, nil
}

// Create records the traffic of port to a new file at path.
func Create(path string, port io.ReadWriteCloser) (*Recorder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("record: %w", err)
	}
	r, err := NewRecorder(port, f)
	if err != nil {
		_ = f.Close() // reporting the header error
		return nil, err
	}
	return r, nil
}

func (r *Recorder) record(dir byte, p []byte) {
	if len(p) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return
	}
	var hdr [1 + 2*binary.MaxVarintLen64]byte
	hdr[0] = dir
	n := 1 + binary.PutUvarint(hdr[1:], uint64(time.Since(r.start)))
	n += binary.PutUvarint(hdr[n:], uint64(len(p)))
	if _, err := r.w.Write(hdr[:n]); err != nil {
		r.err = err
		return
	}
	if _, err := r.w.Write(p); err != nil {
		r.err = err
	}
}

// Read reads from the port and records what was read.
func (r *Recorder) Read(p []byte) (int, error) {
	n, err := r.port.Read(p)
	r.record(FromDevice, p[:n])
	return n, err
}

// Write records p and writes it to the port.
func (r *Recorder) Write(p []byte) (int, error) {
	r.record(ToDevice, p)
	return r.port.Write(p)
}

// Err returns the first error writing the recording.
func (r *Recorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// Close closes the port and finishes the recording.
func (r *Recorder) Close() error {
	err := r.port.Close()
	r.mu.Lock()
	defer r.mu.Unlock()
	if ferr := r.w.Flush(); ferr != nil && r.err == nil {
		r.err = ferr
	}
	if r.c != nil {
		if cerr := r.c.Close(); cerr != nil && r.err == nil {
			r.err = cerr
		}
		r.c = nil
	}
	if err == nil && r.err != nil {
		err = fmt.Errorf("record: %w", r.err)
	}
	return err
}

// Record is one read or write in a recording.
type Record struct {
	Dir  byte
	At   time.Duration
	Data []byte
}

// ReadAll parses a recording.
func ReadAll(r io.Reader) ([]Record, error) {
	br := bufio.NewReader(r)
	hdr := make([]byte, len(header))
	if _, err := io.ReadFull(br, hdr); err != nil || string(hdr) != header {
		return nil, errors.New("record: not a recording")
	}
	var recs []Record
	for {
		dir, err := br.ReadByte()
		if errors.Is(err, io.EOF) {
			return recs, nil
		}
		if err != nil {
			return nil, fmt.Errorf("record: %w", err)
		}
		if dir != FromDevice && dir != ToDevice {
			return nil, fmt.Errorf("record: invalid direction %q at record %d", dir, len(recs))
		}
		at, err := binary.ReadUvarint(br)
		if err != nil {
			return nil, fmt.Errorf("record: truncated record %d: %w", len(recs), err)
		}
		n, err := binary.ReadUvarint(br)
		if err != nil || n > 1<<24 {
			return nil, fmt.Errorf("record: invalid length in record %d", len(recs))
		}
		data := make([]byte, n)
		if _, err := io.ReadFull(br, data); err != nil {
			return nil, fmt.Errorf("record: truncated record %d: %w", len(recs), err)
		}
		recs = append(recs, Record{Dir: dir, At: time.Duration(at), Data: data})
	}
}

// PlayerOptions configures a Player.
type PlayerOptions struct {
	// Speed scales the recorded timing: 1 plays in real time, 2 twice as
	// fast. Zero plays without delays.
	Speed float64
	// Loop restarts the recording at its end instead of reporting io.EOF.
	Loop bool
}

// Player is a serial port that plays back what a device sent in a
// recording. Commands written to it are accepted and ignored, so commands
// that wait for an answer time out.
type Player struct {
	opts    PlayerOptions
	chunks  []Record
	closeCh chan struct{}
	once    sync.Once

	// Read state, used by one reader at a time.
	readMu  sync.Mutex
	next    int
	pending []byte
	start   time.Time
	offset  time.Duration // recording time at start
}

// NewPlayer plays the recording read from r.
func NewPlayer(r io.Reader, opts PlayerOptions) (*Player, error) {
	recs, err := ReadAll(r)
	if err != nil {
		return nil, err
	}
	var chunks []Record
	for _, rec := range recs {
		if rec.Dir == FromDevice {
			chunks = append(chunks, rec)
		}
	}
	if len(chunks) == 0 {
		return nil, errors.New("record: recording has no device data")
	}
	return &Player{opts: opts, chunks: chunks, closeCh: make(chan struct{})}, nil
}

// Open plays the recording in the file at path.
func Open(path string, opts PlayerOptions) (*Player, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("record: %w", err)
	}
	return NewPlayer(bytes.NewReader(data), opts)
}

// Read returns recorded device data once it is due.
func (p *Player) Read(b []byte) (int, error) {
	p.readMu.Lock()
	defer p.readMu.Unlock()
	for len(p.pending) == 0 {
		if p.next == len(p.chunks) {
			if !p.opts.Loop {
				return 0, io.EOF
			}
			p.next = 0
			p.start = time.Time{}
		}
		c := p.chunks[p.next]
		if p.start.IsZero() {
			p.start = time.Now()
			p.offset = c.At
		}
		if p.opts.Speed > 0 {
			due := p.start.Add(time.Duration(float64(c.At-p.offset) / p.opts.Speed))
			if d := time.Until(due); d > 0 {
				t := time.NewTimer(d)
				select {
				case <-t.C:
				case <-p.closeCh:
					t.Stop()
					return 0, io.ErrClosedPipe
				}
			}
		}
		select {
		case <-p.closeCh:
			return 0, io.ErrClosedPipe
		default:
		}
		p.pending = c.Data
		p.next++
	}
	n := copy(b, p.pending)
	p.pending = p.pending[n:]
	return n, nil
}

// Write discards commands.
func (p *Player) Write(b []byte) (int, error) {
	select {
	case <-p.closeCh:
		return 0, io.ErrClosedPipe
	default:
	}
	return len(b), nil
}

// Close stops playback and unblocks Read.
func (p *Player) Close() error {
	p.once.Do(func() { close(p.closeCh) })
	return nil
}
