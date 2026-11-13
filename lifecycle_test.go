package rfx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"samuelks.com/rfx/rfxtest"
)

const (
	testSetup  = "#C2-M:006,005,01.34\r\n"
	testConfig = "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
)

// fakeAnalyzer answers the commands a 6G + WSUB3G unit would, and stores
// presets written to it.
type fakeAnalyzer struct {
	mu      sync.Mutex
	presets map[byte][]byte // index → preset record as written by the host
	clamp   bool            // adjust requested ranges instead of echoing them
	config  string          // current config line
}

func (a *fakeAnalyzer) handle(cmd []byte) []byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case string(cmd) == "C0":
		if a.config == "" {
			a.config = testConfig
		}
		return []byte("#CAL:11\r\n" + testSetup + "#QA:0\r\n" + a.config + "#Sn0123456789ABCDEF\r\n")
	case string(cmd) == "Cq":
		return cat("$q", 154, make([]byte, 154), crlf)
	case bytes.HasPrefix(cmd, []byte("C2-F:")):
		// Like the device, report the new config only if it changed.
		f := strings.Split(string(cmd[5:]), ",")
		start, _ := strconv.Atoi(f[0])
		end, _ := strconv.Atoi(f[1])
		if a.clamp {
			start += 100
		}
		step := (end - start) * 1000 / 111
		config := fmt.Sprintf("#C2-F:%07d,%07d,%s,%s,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n", start, step, f[2], f[3])
		if config == a.config {
			return nil
		}
		a.config = config
		return []byte(config)
	case string(cmd) == "CP\x00":
		var out []byte
		for i := range byte(MaxPresets) {
			if rec, ok := a.presets[i]; ok {
				// "$P" len index \x01 name... as sent by the device.
				out = append(out, cat("$P", 0x20, rec[0], 0x01, rec[1:31], crlf)...)
			}
		}
		return append(out, "#PCK\r\n"...)
	case bytes.HasPrefix(cmd, []byte("CP\x01")):
		rec := bytes.Clone(cmd[3:])
		if a.presets == nil {
			a.presets = make(map[byte][]byte)
		}
		if rec[1] == 0 {
			delete(a.presets, rec[0])
		} else {
			a.presets[rec[0]] = rec
		}
		return []byte("#PCK\r\n")
	}
	return nil
}

func TestMain(m *testing.M) {
	// The fake device sends no sweeps, so pending commands only time out.
	commandTimeout = 10 * time.Millisecond
	os.Exit(m.Run())
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func newTestRFE(t *testing.T) (*RFExplorer, *rfxtest.Device, *fakeAnalyzer) {
	t.Helper()
	a := &fakeAnalyzer{}
	dev := rfxtest.NewDevice(a.handle)
	rfe, err := New(testContext(t), dev)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rfe.Close(); err != nil {
			t.Errorf("Close: %s", err)
		}
	})
	return rfe, dev, a
}

// collect reads packets until one satisfies stop.
func collect(t *testing.T, rfe *RFExplorer, stop func(Packet) bool) []Packet {
	t.Helper()
	var pkts []Packet
	timeout := time.After(5 * time.Second)
	for {
		select {
		case pkt, ok := <-rfe.Chan():
			if !ok {
				t.Fatalf("channel closed after %d packets: %v", len(pkts), rfe.Err())
			}
			pkts = append(pkts, pkt)
			if stop(pkt) {
				return pkts
			}
		case <-timeout:
			t.Fatalf("timed out after %d packets", len(pkts))
		}
	}
}

func isType[T Packet](pkt Packet) bool {
	_, ok := pkt.(T)
	return ok
}

func TestHandshake(t *testing.T) {
	rfe, dev, _ := newTestRFE(t)

	if got := rfe.MainBoardModel(); got != Model6G {
		t.Errorf("main board %s, want 6G", got)
	}
	if got := rfe.ActiveModel(); got != ModelWSUB3G {
		t.Errorf("active model %s, want WSUB3G", got)
	}

	// Packets that arrived during the handshake are still delivered.
	pkts := collect(t, rfe, isType[*CalibrationDataPacket])
	var types []string
	for _, pkt := range pkts {
		types = append(types, pkt.Type())
	}
	want := []string{"CalibrationAvailability", "CurrentSetup", "UnhandledPacket", "CurrentConfig", "SerialNumber", "CalibrationData"}
	if !reflect.DeepEqual(types, want) {
		t.Errorf("got packets %v, want %v", types, want)
	}

	cmds := dev.Commands()
	if len(cmds) != 2 || string(cmds[0]) != "C0" || string(cmds[1]) != "Cq" {
		t.Errorf("got commands %q, want C0 then Cq", cmds)
	}
}

func TestHandshakeTimeout(t *testing.T) {
	dev := rfxtest.NewDevice(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rfe, err := New(ctx, dev)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, %v; want deadline exceeded", rfe, err)
	}
	// The port must have been closed.
	if _, err := dev.Write([]byte("#\x04C0")); err == nil {
		t.Error("port still open after failed handshake")
	}
}

func TestClose(t *testing.T) {
	a := &fakeAnalyzer{}
	rfe, err := New(testContext(t), rfxtest.NewDevice(a.handle))
	if err != nil {
		t.Fatal(err)
	}
	// The reader is blocked in Read and nothing is consuming packets.
	done := make(chan error, 1)
	go func() { done <- rfe.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}
	if err := rfe.Close(); err != nil {
		t.Errorf("second Close: %s", err)
	}
	if err := rfe.Err(); err != nil {
		t.Errorf("Err after Close: %s", err)
	}
	for range rfe.Chan() {
		// Buffered packets may remain; the channel must end.
	}
	if err := rfe.RequestConfig(); err == nil {
		t.Error("command succeeded after Close")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rfe.DeletePreset(ctx, 0); err == nil {
		t.Error("DeletePreset succeeded after Close")
	}
}

func TestReadError(t *testing.T) {
	rfe, dev, _ := newTestRFE(t)
	dev.Send([]byte("#Sn0000000000000001\r\n"))
	errUnplugged := errors.New("unplugged")
	dev.Fail(errUnplugged)

	// Everything received before the failure is delivered, then the channel
	// closes.
	var last Packet
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case pkt, ok := <-rfe.Chan():
			if !ok {
				break loop
			}
			last = pkt
		case <-timeout:
			t.Fatal("channel was not closed")
		}
	}
	if sn, ok := last.(*SerialNumberPacket); !ok || sn.SN != "0000000000000001" {
		t.Errorf("last packet %+v, want the serial number sent before the failure", last)
	}
	if err := rfe.Err(); !errors.Is(err, errUnplugged) {
		t.Errorf("Err = %v, want %v", err, errUnplugged)
	}
	if _, err := rfe.SetAnalyzerConfig(testContext(t), 2400000, 2500000, 0, -120, 0); !errors.Is(err, errUnplugged) {
		t.Errorf("SetAnalyzerConfig = %v, want %v", err, errUnplugged)
	}
}

// A consumer that isn't reading must not stall the reader, lose device
// state, or hold up commands waiting on a response.
func TestSlowConsumer(t *testing.T) {
	rfe, dev, _ := newTestRFE(t)

	const sweeps = 1000
	sweep := cat("$S", 4, 1, 2, 3, 4, crlf)
	var stream []byte
	for i := range sweeps {
		stream = append(stream, sweep...)
		if i%100 == 0 {
			stream = append(stream, "#C+\x04\r\n"...)
		}
	}
	dev.Send(stream)

	ctx := testContext(t)
	if err := rfe.UpdatePreset(ctx, &Preset{Index: 3, Name: "Test", MinFreqKHz: 1000, MaxFreqKHz: 2000, AmpTopDBm: 0, AmpBottomDBm: -100, CalcIterations: 1}); err != nil {
		t.Fatalf("UpdatePreset with a stalled consumer: %s", err)
	}
	cfg, err := rfe.SetAnalyzerConfig(ctx, 433000, 434000, -10, -110, 0)
	if err != nil {
		t.Fatalf("SetAnalyzerConfig with a stalled consumer: %s", err)
	}
	if cfg.StartFreqKHZ != 433000 || cfg.AmpTopDBM != -10 {
		t.Errorf("got config %+v", cfg)
	}

	// Mark the end of the stream, then drain.
	dev.Send([]byte("#Sn0000000000000002\r\n"))
	pkts := collect(t, rfe, func(pkt Packet) bool {
		sn, ok := pkt.(*SerialNumberPacket)
		return ok && sn.SN == "0000000000000002"
	})
	var nSweeps, nModes, nAcks int
	for _, pkt := range pkts {
		switch pkt.(type) {
		case *SweepDataPacket:
			nSweeps++
		case *CurrentCalculatorMode:
			nModes++
		case *EndOfPresetsPacket:
			nAcks++
		}
	}
	if nModes != sweeps/100 || nAcks != 1 {
		t.Errorf("got %d calculator modes and %d preset acks, want %d and 1", nModes, nAcks, sweeps/100)
	}
	// The channel buffer and the packet in flight add to the queue limit.
	if limit := maxQueuedStreamPackets + cap(rfe.readCh) + 1; nSweeps == 0 || nSweeps > limit {
		t.Errorf("got %d sweeps, want 1 to %d", nSweeps, limit)
	}
	if rfe.Dropped() == 0 {
		t.Error("no packets reported dropped")
	}
}

func TestConcurrentCommands(t *testing.T) {
	rfe, dev, _ := newTestRFE(t)
	ctx := testContext(t)

	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			if err := rfe.SetCalculatorMode(CalculatorModeMaxHold); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() {
			p := &Preset{Index: i, Name: "Concurrent", MinFreqKHz: 1000, MaxFreqKHz: 2000, AmpTopDBm: 0, AmpBottomDBm: -100, CalcIterations: 1}
			if err := rfe.UpdatePreset(ctx, p); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	// Interleaved writes would corrupt the command framing.
	var modes, presets int
	for _, cmd := range dev.Commands()[2:] {
		switch {
		case string(cmd) == "C+\x04":
			modes++
		case len(cmd) == 34 && bytes.HasPrefix(cmd, []byte("CP\x01")):
			presets++
		default:
			t.Errorf("unexpected command %q", cmd)
		}
	}
	if modes != n || presets != n {
		t.Errorf("got %d mode and %d preset commands, want %d each", modes, presets, n)
	}
}

func TestPresetRoundTrip(t *testing.T) {
	rfe, _, _ := newTestRFE(t)
	ctx := testContext(t)

	want := []*Preset{
		{Index: 0, Name: "VTX 5.8GHz", MinFreqKHz: 5632000, MaxFreqKHz: 5958000, AmpTopDBm: -20, AmpBottomDBm: -90,
			CalcMode: CalculatorModeNormal, CalcIterations: 4, Mainboard: true, MarkerMode: MarkerModePeak},
		{Index: 99, Name: "123456789012", MinFreqKHz: 0, MaxFreqKHz: 9999999, AmpTopDBm: 35, AmpBottomDBm: 25,
			CalcMode: CalculatorModeMaxHold, CalcIterations: 16, Mainboard: false, MarkerMode: MarkerModeManual},
	}
	for _, p := range want {
		if err := rfe.UpdatePreset(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := rfe.UpdatePreset(ctx, &Preset{Index: 5, Name: "Temporary", MinFreqKHz: 1, MaxFreqKHz: 2, AmpTopDBm: 0, AmpBottomDBm: -100, CalcIterations: 1}); err != nil {
		t.Fatal(err)
	}
	if err := rfe.DeletePreset(ctx, 5); err != nil {
		t.Fatal(err)
	}

	// Skip the acks of the writes above, then read the list back.
	collect(t, rfe, isType[*CalibrationDataPacket])
	for range len(want) + 2 {
		collect(t, rfe, isType[*EndOfPresetsPacket])
	}
	if err := rfe.RequestPresets(); err != nil {
		t.Fatal(err)
	}
	var got []*Preset
	for _, pkt := range collect(t, rfe, isType[*EndOfPresetsPacket]) {
		if p, ok := pkt.(*Preset); ok {
			got = append(got, p)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %d presets, want %d", len(got), len(want))
		for _, p := range got {
			t.Logf("got %+v", p)
		}
	}
}

func TestUpdatePresetTimeout(t *testing.T) {
	dev := rfxtest.NewDevice(func(cmd []byte) []byte {
		if string(cmd) == "C0" {
			return []byte(testConfig)
		}
		return nil
	})
	rfe, err := New(testContext(t), dev)
	if err != nil {
		t.Fatal(err)
	}
	defer rfe.Close() //nolint:errcheck // not under test
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	p := &Preset{Name: "Test", MinFreqKHz: 1000, MaxFreqKHz: 2000, AmpTopDBm: 0, AmpBottomDBm: -100, CalcIterations: 1}
	if err := rfe.UpdatePreset(ctx, p); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want deadline exceeded", err)
	}
}

func TestSetAnalyzerConfig(t *testing.T) {
	rfe, dev, a := newTestRFE(t)
	ctx := testContext(t)

	cfg, err := rfe.SetAnalyzerConfig(ctx, 5350000, 5950000, 10, -100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StartFreqKHZ != 5350000 || cfg.AmpTopDBM != 10 || cfg.AmpBottomDBM != -100 {
		t.Errorf("got %+v", cfg)
	}
	if cmds := dev.Commands(); len(cmds) != 3 || string(cmds[2]) != "C2-F:5350000,5950000,0010,-100" {
		t.Errorf("sent %q", cmds[2:])
	}

	// The device doesn't echo an unchanged configuration, so it is asked.
	cfg, err = rfe.SetAnalyzerConfig(ctx, 5350000, 5950000, 10, -100, 0)
	if err != nil {
		t.Fatalf("unchanged config: %s", err)
	}
	if cfg.StartFreqKHZ != 5350000 {
		t.Errorf("unchanged config: got %+v", cfg)
	}
	if cmds := dev.Commands(); len(cmds) != 4 || string(cmds[3]) != "C0" {
		t.Errorf("unchanged config: sent %q, want only a config request", cmds[3:])
	}

	// A device that adjusts the range reports a different start frequency.
	a.mu.Lock()
	a.clamp = true
	a.mu.Unlock()
	configEchoTimeout = 50 * time.Millisecond
	defer func() { configEchoTimeout = 1500 * time.Millisecond }()
	cfg, err = rfe.SetAnalyzerConfig(ctx, 100000, 200000, 0, -120, 0)
	if err != nil {
		t.Fatalf("adjusted config: %s", err)
	}
	if cfg.StartFreqKHZ != 100100 {
		t.Errorf("adjusted config: got %+v", cfg)
	}

	n := len(dev.Commands())
	for _, c := range []struct {
		name                    string
		start, end, top, bottom int
	}{
		{"reversed range", 2500000, 2400000, 0, -120},
		{"empty range", 2400000, 2400000, 0, -120},
		{"negative start", -1, 2400000, 0, -120},
		{"end too high", 2400000, 10000000, 0, -120},
		{"top too high", 2400000, 2500000, 36, -120},
		{"bottom too low", 2400000, 2500000, 0, -121},
		{"range under 10 dB", 2400000, 2500000, -50, -55},
		{"bottom above top", 2400000, 2500000, -100, -50},
	} {
		if _, err := rfe.SetAnalyzerConfig(ctx, c.start, c.end, c.top, c.bottom, 0); err == nil {
			t.Errorf("%s: no error", c.name)
		}
	}
	if got := len(dev.Commands()); got != n {
		t.Errorf("%d commands sent for invalid configurations", got-n)
	}
}

func TestCommandEncoding(t *testing.T) {
	rfe, dev, _ := newTestRFE(t)
	n := len(dev.Commands())
	for _, c := range []struct {
		name string
		send func() error
		want string
	}{
		{"calculator mode", func() error { return rfe.SetCalculatorMode(CalculatorModeMaxHistorical) }, "C+\x05"},
		{"realtime", rfe.Realtime, "C+\x00"},
		{"max hold", rfe.SetMaxHold, "C+\x04"},
		{"wifi 2.4", func() error { return rfe.SetWifiMode(WifiMode24GHz) }, "CW\x01"},
		{"wifi off", func() error { return rfe.SetWifiMode(WifiModeDisabled) }, "CW\x00"},
		{"sweep points", func() error { return rfe.SetSweepPoints(240) }, "CJ\x0e"},
		{"sweep points min", func() error { return rfe.SetSweepPoints(1) }, "CJ\x00"},
		{"sweep points max", func() error { return rfe.SetSweepPoints(100000) }, "CJ\xff"},
		{"sweep points ex", func() error { return rfe.SetSweepPointsEx(512) }, "Cj\x02\x00"},
		{"sweep points ex max", func() error { return rfe.SetSweepPointsEx(1 << 20) }, "Cj\xff\xff"},
		{"tracking step", func() error { return rfe.SetTrackingStep(0x1234) }, "k\x12\x34"},
		{"module main", rfe.SwitchModuleMain, "CM\x00"},
		{"lcd off", func() error { return rfe.SetLCDEnabled(false) }, "L0"},
	} {
		if err := c.send(); err != nil {
			t.Errorf("%s: %s", c.name, err)
			continue
		}
		cmds := dev.Commands()
		if len(cmds) != n+1 || string(cmds[n]) != c.want {
			t.Errorf("%s: sent %q, want %q", c.name, cmds[n:], c.want)
		}
		n = len(cmds)
	}

	if err := rfe.SetCalculatorMode(CalculatorModeInvalid); err == nil {
		t.Error("invalid calculator mode accepted")
	}
	if err := rfe.SetWifiMode(3); err == nil {
		t.Error("invalid WiFi mode accepted")
	}
}

func TestPresetValidate(t *testing.T) {
	valid := Preset{Index: 1, Name: "2.4G WiFi", MinFreqKHz: 2401000, MaxFreqKHz: 2495000, AmpTopDBm: 0, AmpBottomDBm: -100, CalcIterations: 4}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid preset rejected: %s", err)
	}
	for _, c := range []struct {
		name string
		edit func(p *Preset)
	}{
		{"negative index", func(p *Preset) { p.Index = -1 }},
		{"index too high", func(p *Preset) { p.Index = 100 }},
		{"empty name", func(p *Preset) { p.Name = "" }},
		{"long name", func(p *Preset) { p.Name = "1234567890123" }},
		{"non-ASCII name", func(p *Preset) { p.Name = "café" }},
		{"control character in name", func(p *Preset) { p.Name = "a\x00b" }},
		{"reversed range", func(p *Preset) { p.MinFreqKHz, p.MaxFreqKHz = 2, 1 }},
		{"frequency too high", func(p *Preset) { p.MaxFreqKHz = 10000000 }},
		{"top too high", func(p *Preset) { p.AmpTopDBm = 36 }},
		{"bottom too low", func(p *Preset) { p.AmpBottomDBm = -121 }},
		{"range under 10 dB", func(p *Preset) { p.AmpTopDBm, p.AmpBottomDBm = -50, -59 }},
		{"no iterations", func(p *Preset) { p.CalcIterations = 0 }},
		{"too many iterations", func(p *Preset) { p.CalcIterations = 17 }},
		{"invalid calculator mode", func(p *Preset) { p.CalcMode = 6 }},
		{"invalid marker mode", func(p *Preset) { p.MarkerMode = 3 }},
	} {
		p := valid
		c.edit(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: no error", c.name)
		}
	}
}
