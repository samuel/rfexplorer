package rfx

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// presetFrame is a preset as sent by the device: index 0, "RC Remote",
// 27-28 MHz, -20 to -120 dBm, 10 iterations, expansion board, no marker.
var presetFrame = []byte{
	0x24, 0x50, 0x20, 0x00, 0x01, 0x52, 0x43, 0x20, 0x52, 0x65, 0x6D, 0x6F, 0x74, 0x65, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x78, 0x69, 0x00, 0x00, 0x60, 0x6D, 0x00, 0x00, 0x00, 0xEC, 0x88, 0x0A, 0x00,
	0x01, 0x42, 0x00,
}

// payload returns n bytes that include a CRLF, to check that binary frames
// are delimited by length.
func payload(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(0x60 + i%64)
	}
	if n >= 4 {
		b[1], b[2] = '\r', '\n'
		b[n-2], b[n-1] = '\r', '\n'
	}
	return b
}

func cat(parts ...any) []byte {
	var b []byte
	for _, p := range parts {
		switch p := p.(type) {
		case string:
			b = append(b, p...)
		case []byte:
			b = append(b, p...)
		case byte:
			b = append(b, p)
		case int:
			b = append(b, byte(p))
		}
	}
	return b
}

// testFrames has one frame of every kind, without terminators.
func testFrames() [][]byte {
	presetCRLF := bytes.Clone(presetFrame)
	copy(presetCRLF[23:27], []byte{'\r', '\n', 0, 0}) // max frequency 2573 kHz
	return [][]byte{
		[]byte("#C2-M:006,005,01.34"),
		cat("$S", 112, payload(112)),
		[]byte("#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000"),
		cat("$s", 2, payload(32)),
		cat("$s", 0, payload(4096)),
		cat("$z", 0x01, 0x2c, payload(300)),
		cat("$D", payload(screenDataLen)),
		cat("$R", 5, 0, payload(5)),
		presetFrame,
		presetCRLF,
		cat("$q", 154, payload(154)),
		cat("$Q", 0x02, 0x01, payload(258)),
		[]byte("#C+\x04"),
		[]byte("RF Explorer 01.34 27-Apr-22 01.04.05"),
		[]byte("#Sn0123456789ABCDEF"),
		[]byte("$X unknown binary type"),
		[]byte("#PCK"),
	}
}

func joinFrames(frames [][]byte) []byte {
	var b []byte
	for _, f := range frames {
		b = append(b, f...)
		b = append(b, crlf...)
	}
	return b
}

// drain returns copies of all frames currently available.
func drain(f *framer) [][]byte {
	var frames [][]byte
	for {
		frame, ok := f.Next()
		if !ok {
			return frames
		}
		frames = append(frames, bytes.Clone(frame))
	}
}

func equalFrames(a, b [][]byte) bool {
	return slices.EqualFunc(a, b, bytes.Equal)
}

func TestFramer(t *testing.T) {
	want := testFrames()
	var f framer
	f.Write(joinFrames(want))
	got := drain(&f)
	if len(got) != len(want) {
		t.Fatalf("got %d frames, want %d", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("frame %d: got %d bytes %.40q, want %d bytes %.40q", i, len(got[i]), got[i], len(want[i]), want[i])
		}
	}
}

func TestFramerSplit(t *testing.T) {
	want := testFrames()
	stream := joinFrames(want)

	t.Run("every split point", func(t *testing.T) {
		for k := range len(stream) + 1 {
			var f framer
			f.Write(stream[:k])
			got := drain(&f)
			f.Write(stream[k:])
			got = append(got, drain(&f)...)
			if !equalFrames(got, want) {
				t.Fatalf("split at %d: got %d frames, want %d", k, len(got), len(want))
			}
		}
	})

	t.Run("byte at a time", func(t *testing.T) {
		var f framer
		var got [][]byte
		for i := range stream {
			f.Write(stream[i : i+1])
			got = append(got, drain(&f)...)
		}
		if !equalFrames(got, want) {
			t.Fatalf("got %d frames, want %d", len(got), len(want))
		}
	})
}

func TestFramerNoise(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want []string
	}{
		{"bare marker line", "#\r\n#QA:0\r\n", []string{"#", "#QA:0"}},
		{"empty lines", "\r\n\r\n#QA:0\r\n\r\n", []string{"#QA:0"}},
		{"junk before frame", "\x00\xff\x10#QA:0\r\n", []string{"\x00\xff\x10", "#QA:0"}},
		{"junk line", "\x00\xff\x10\r\n#QA:0\r\n", []string{"\x00\xff\x10", "#QA:0"}},
		{"incomplete header", "$", nil},
		{"incomplete length", "$z\x01", nil},
		{"incomplete payload", "$S\x04AB", nil},
		{"missing terminator", "$S\x04ABCD", nil},
		{"wrong terminator", "$S\x02ABXY#QA:0\r\n", []string{"$S\x02ABXY", "#QA:0"}},
		{"truncated sweep", "$S\x70ABC\r\n#QA:0\r\n" + strings.Repeat("\x80", 120) + "\r\n#Sn1\r\n",
			[]string{"$S\x70ABC", "#QA:0", strings.Repeat("\x80", 120), "#Sn1"}},
		{"partial frame at start", "\xa9\xb0\xad\r\n$S\x02AB\r\n", []string{"\xa9\xb0\xad", "$S\x02AB"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var f framer
			f.Write([]byte(c.in))
			var got []string
			for _, frame := range drain(&f) {
				got = append(got, string(frame))
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// A line without a terminator must not buffer without bound or hide the
// frames after it.
func TestFramerOverlongLine(t *testing.T) {
	var f framer
	f.Write([]byte("#" + strings.Repeat("a", 5000)))
	junk := drain(&f)
	var n int
	for _, frame := range junk {
		if len(frame) > maxLineLen {
			t.Errorf("frame of %d bytes exceeds the %d byte limit", len(frame), maxLineLen)
		}
		n += len(frame)
	}
	if buffered := 5001 - n; buffered > maxLineLen+len(crlf) {
		t.Errorf("%d bytes left buffered", buffered)
	}
	f.Write([]byte("\r\n#QA:0\r\n"))
	got := drain(&f)
	if len(got) == 0 || string(got[len(got)-1]) != "#QA:0" {
		t.Errorf("frame after an overlong line was not recovered: %q", got)
	}
}

// TestCaptures decodes traffic recorded from a 6G + WSUB3G unit running
// firmware 1.34. Each capture starts partway through a sweep.
func TestCaptures(t *testing.T) {
	for _, c := range []struct {
		file string
		want map[string]int
	}{
		{"6g-wsub3g-fw1.34-startup.bin", map[string]int{
			"CalibrationAvailability": 1, "CalibrationData": 1, "CurrentConfig": 1, "CurrentSetup": 1,
			"EndOfPresets": 1, "Preset": 12, "SerialNumber": 2, "SweepData": 13, "UnhandledPacket": 4,
		}},
		{"6g-wsub3g-fw1.34-screen.bin", map[string]int{
			"ScreenImage": 2, "SweepData": 3, "UnhandledPacket": 1,
		}},
		{"6g-wsub3g-fw1.34-config.bin", map[string]int{
			"CalibrationAvailability": 3, "CurrentCalculatorMode": 2, "CurrentConfig": 3, "CurrentSetup": 3,
			"SerialNumber": 3, "SweepData": 27, "UnhandledPacket": 10,
		}},
	} {
		t.Run(c.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", c.file))
			if err != nil {
				t.Fatal(err)
			}
			var f framer
			f.Write(data)
			got := make(map[string]int)
			for _, frame := range drain(&f) {
				pkt, err := decode(frame)
				if err != nil {
					t.Errorf("decode %.40q: %s", frame, err)
					continue
				}
				got[pkt.Type()]++
				switch pkt := pkt.(type) {
				case *SweepDataPacket:
					if len(pkt.Samples) != 112 {
						t.Errorf("sweep has %d samples, want 112", len(pkt.Samples))
					}
				case *UnhandledPacket:
					t.Logf("unhandled: %.60q", pkt.Data)
				}
			}
			for typ, n := range c.want {
				if got[typ] != n {
					t.Errorf("%s: got %d, want %d", typ, got[typ], n)
				}
			}
			for typ, n := range got {
				if _, ok := c.want[typ]; !ok {
					t.Errorf("%s: got %d, want 0", typ, n)
				}
			}
		})
	}
}

func FuzzFramer(f *testing.F) {
	f.Add(joinFrames(testFrames()[:4]), uint(10))
	f.Add([]byte("$S\x02ABXY#QA:0\r\n"), uint(3))
	f.Add([]byte("\r\n$\r\n#\r\n$z\xff\xff"), uint(1))
	f.Add([]byte("#"+strings.Repeat("a", 2*maxLineLen)+"\r\n$D"), uint(maxLineLen))
	f.Fuzz(func(t *testing.T, data []byte, split uint) {
		var whole framer
		whole.Write(data)
		want := drain(&whole)

		// Framing must not depend on how the stream is chunked.
		k := int(split % uint(len(data)+1))
		var parts framer
		parts.Write(data[:k])
		got := drain(&parts)
		parts.Write(data[k:])
		got = append(got, drain(&parts)...)
		if !equalFrames(got, want) {
			t.Fatalf("split at %d: got %q, want %q", k, got, want)
		}

		var n int
		for _, frame := range want {
			if len(frame) == 0 {
				t.Fatal("empty frame")
			}
			n += len(frame)
			if _, err := decode(frame); err != nil {
				continue
			}
		}
		if n > len(data) {
			t.Fatalf("frames hold %d bytes from %d bytes of input", n, len(data))
		}
	})
}
