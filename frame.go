package rfx

import "bytes"

// maxLineLen is the longest CRLF-terminated frame accepted. A text frame
// without a terminator within this many bytes is treated as corrupt.
const maxLineLen = 1024

// screenDataLen is the size of the LCD bitmap in a screen dump frame.
const screenDataLen = 128 * 64 / 8

var crlf = []byte{'\r', '\n'}

// eeot is the Early End Of Transmission marker that firmware after 1.27 sends
// in place of the rest of a sweep it abandons.
var eeot = []byte{0xff, 0xfe, 0xff, 0xfe, 0x00}

// framer splits the byte stream from the device into frames. Every frame on
// the wire is terminated by CRLF. Binary frames carry their own length and are
// delimited by it, since their payload may contain CRLF. Text frames and
// anything unrecognized are delimited by the first CRLF.
type framer struct {
	buf []byte
	pos int // start of unconsumed data in buf
}

// Write appends received bytes. It invalidates frames returned by Next.
func (f *framer) Write(p []byte) {
	if f.pos > 0 {
		n := copy(f.buf, f.buf[f.pos:])
		f.buf = f.buf[:n]
		f.pos = 0
	}
	f.buf = append(f.buf, p...)
}

// Next returns the next complete frame without its terminator, or false if
// more data is needed. The frame aliases the internal buffer and is valid
// until the next call to Write.
func (f *framer) Next() ([]byte, bool) {
	for {
		b := f.buf[f.pos:]
		if len(b) == 0 {
			return nil, false
		}
		frame, consumed := nextFrame(b)
		if consumed == 0 {
			return nil, false
		}
		f.pos += consumed
		if len(frame) != 0 {
			return frame, true
		}
	}
}

// nextFrame extracts one frame from the start of b. It returns the frame and
// the number of bytes to consume, or consumed == 0 if b is incomplete.
func nextFrame(b []byte) (frame []byte, consumed int) {
	switch b[0] {
	case '$':
		n, binary, ok := binaryFrameLen(b)
		if !ok {
			return nil, 0
		}
		if !binary {
			return nextLine(b)
		}
		if isSweepFrame(b[1]) {
			// An abandoned sweep ends at the marker, which is kept in the
			// frame so decode can report it.
			limit := min(len(b), n+len(crlf))
			if i := bytes.Index(b[2:limit], eeot); i >= 0 {
				end := 2 + i + len(eeot)
				return b[:end], end
			}
		}
		if len(b) < n+len(crlf) {
			return nil, 0
		}
		if b[n] == '\r' && b[n+1] == '\n' {
			return b[:n], n + len(crlf)
		}
		// The declared length doesn't end at a terminator.
		return nextJunk(b)
	case '#':
		return nextLine(b)
	}
	return nextJunk(b)
}

func isSweepFrame(c byte) bool {
	return c == 'S' || c == 's' || c == 'z'
}

// binaryFrameLen returns the length of the length-delimited frame at the
// start of b, excluding the terminator. binary is false for frame types that
// are delimited by CRLF instead. ok is false if the header is incomplete.
func binaryFrameLen(b []byte) (n int, binary, ok bool) {
	if len(b) < 2 {
		return 0, false, false
	}
	var hdr int
	switch b[1] {
	case 'S', 's', 'P', 'q':
		hdr = 3
	case 'z', 'R', 'Q':
		hdr = 4
	case 'D':
		return 2 + screenDataLen, true, true
	default:
		return 0, false, true
	}
	if len(b) < hdr {
		return 0, true, false
	}
	switch b[1] {
	case 's':
		// Count of 16-sample blocks, with 0 meaning 256.
		n = int(b[2]) * 16
		if n == 0 {
			n = 4096
		}
	case 'z':
		n = int(b[2])<<8 | int(b[3])
	case 'R', 'Q':
		n = int(b[2]) | int(b[3])<<8
	default:
		n = int(b[2])
	}
	return hdr + n, true, true
}

// nextLine extracts a CRLF-terminated frame that starts with a frame marker.
func nextLine(b []byte) (frame []byte, consumed int) {
	limit := min(len(b), maxLineLen+len(crlf))
	if i := bytes.Index(b[:limit], crlf); i >= 0 {
		return b[:i], i + len(crlf)
	}
	if len(b) >= limit && limit == maxLineLen+len(crlf) {
		return nextJunk(b)
	}
	return nil, 0
}

// nextJunk extracts bytes that are not a valid frame: everything up to the
// first CRLF or the next frame marker after the first byte, so that parsing
// resynchronizes on the following frame.
func nextJunk(b []byte) (frame []byte, consumed int) {
	limit := min(len(b), maxLineLen)
	eol := bytes.Index(b[:limit], crlf)
	marker := bytes.IndexAny(b[1:limit], "$#")
	if marker >= 0 {
		marker++
	}
	switch {
	case eol >= 0 && (marker < 0 || eol < marker):
		if eol == 0 {
			// A bare terminator carries nothing; Next skips it.
			return nil, len(crlf)
		}
		return b[:eol], eol + len(crlf)
	case marker >= 0:
		return b[:marker], marker
	case len(b) >= maxLineLen:
		return b[:limit], limit
	}
	return nil, 0
}
