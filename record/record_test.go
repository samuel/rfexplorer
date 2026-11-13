package record

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/rfxtest"
)

const (
	testSetup  = "#C2-M:006,005,01.34\r\n"
	testConfig = "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
)

func handler(cmd []byte) []byte {
	if string(cmd) == "C0" {
		return []byte(testSetup + testConfig)
	}
	return nil
}

func sweep(level byte) []byte {
	b := append([]byte("$S\x70"), bytes.Repeat([]byte{level}, 112)...)
	return append(b, "\r\n"...)
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// recordSession connects to a fake device through a Recorder and receives
// three sweeps.
func recordSession(t *testing.T, path string) {
	t.Helper()
	port := rfxtest.NewDevice(handler)
	rec, err := Create(path, port)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := rfx.New(testContext(t), rec)
	if err != nil {
		t.Fatal(err)
	}
	sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep})
	for i := range 3 {
		time.Sleep(20 * time.Millisecond)
		port.Send(sweep(byte(100 + i)))
		<-sub.C()
	}
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rec.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.rfxrec")
	recordSession(t, path)

	recs, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var sent, received int
	for i, r := range recs {
		if i > 0 && r.At < recs[i-1].At {
			t.Errorf("record %d goes back in time", i)
		}
		switch r.Dir {
		case ToDevice:
			sent++
		case FromDevice:
			received++
		}
	}
	if sent < 2 || received < 2 {
		t.Fatalf("recorded %d writes and %d reads", sent, received)
	}

	// Replay into a new Device: the handshake succeeds from the recording
	// and the sweeps arrive with their recorded levels.
	player, err := Open(path, PlayerOptions{Speed: 1})
	if err != nil {
		t.Fatal(err)
	}
	dev, err := rfx.New(testContext(t), player)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // test cleanup
	if dev.Info().Main != rfx.Model6G || dev.Config().Start != 2400*rfx.MHz {
		t.Errorf("replayed info %+v config %+v", dev.Info(), dev.Config())
	}
	sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep})
	var levels []float32
	for msg := range sub.C() {
		levels = append(levels, msg.(*rfx.Sweep).Samples[0])
	}
	// Subscribing may miss sweeps sent during the handshake; the last
	// arrive in order and the stream ends with the recording.
	if len(levels) == 0 || levels[len(levels)-1] != -51 {
		t.Errorf("replayed sweep levels %v", levels)
	}
	if err := dev.Err(); err == nil || !errors.Is(err, io.EOF) {
		t.Errorf("end of recording reported as %v", err)
	}
}

func TestPlayerLoopAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.rfxrec")
	recordSession(t, path)
	p, err := Open(path, PlayerOptions{Loop: true})
	if err != nil {
		t.Fatal(err)
	}
	// Without delays, looping keeps producing data.
	buf := make([]byte, 64)
	total := 0
	for total < 10_000 {
		n, err := p.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		total += n
	}

	slow, err := Open(path, PlayerOptions{Speed: 0.0001})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slow.Read(buf); err != nil { // the first chunk is due at once
		t.Fatal(err)
	}
	done := make(chan error)
	go func() {
		for {
			if _, err := slow.Read(buf); err != nil {
				done <- err
				return
			}
		}
	}()
	time.Sleep(20 * time.Millisecond)
	if err := slow.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("read after close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not unblock Read")
	}
	if _, err := slow.Write([]byte("#\x04C0")); err == nil {
		t.Error("write after close succeeded")
	}
}

func TestReadAllRejectsBadInput(t *testing.T) {
	for name, data := range map[string]string{
		"not a recording": "hello",
		"bad direction":   header + "x\x00\x01a",
		"truncated":       header + "<\x00\x05ab",
	} {
		if _, err := ReadAll(bytes.NewReader([]byte(data))); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewPlayer(bytes.NewReader([]byte(header+">\x00\x02C0")), PlayerOptions{}); err == nil {
		t.Error("recording without device data accepted")
	}
}

func readFile(path string) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ReadAll(bytes.NewReader(data))
}
