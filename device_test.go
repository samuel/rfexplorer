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
	testSetup      = "#C2-M:006,005,01.34\r\n"
	testConfigExp  = "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
	testConfigMain = "#C2-F:5350000,5405400,0000,-120,0112,0,000,4850000,6100000,0600000,00600,0000,000\r\n"
)

// fakeAnalyzer answers the commands a 6G + WSUB3G unit would, and stores
// presets written to it.
type fakeAnalyzer struct {
	mu      sync.Mutex
	setup   string          // setup line, a 6G + WSUB3G if empty
	presets map[byte][]byte // index → preset record as written by the host
	clamp   bool            // adjust requested ranges instead of echoing them
	config  string          // current config line
	dumping bool
}

func (a *fakeAnalyzer) handle(cmd []byte) []byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.config == "" {
		a.config = testConfigExp
	}
	if a.setup == "" {
		a.setup = testSetup
	}
	// Like the device, report the config only if it changed.
	setConfig := func(config string) []byte {
		if config == a.config {
			return nil
		}
		a.config = config
		return []byte(config)
	}
	switch {
	case string(cmd) == "C0":
		return []byte("#CAL:11\r\n" + a.setup + "#QA:0\r\n" + a.config + "#Sn0123456789ABCDEF\r\n")
	case string(cmd) == "Cq":
		data := make([]byte, 154)
		data[134] = 0x08
		return cat("$q", 154, data, crlf)
	case string(cmd) == "Cn":
		return []byte("#Sn0123456789ABCDEF\r\n")
	case bytes.HasPrefix(cmd, []byte("C2-F:")):
		f := strings.Split(string(cmd[5:]), ",")
		start, _ := strconv.Atoi(f[0])
		end, _ := strconv.Atoi(f[1])
		if a.clamp {
			start += 100
		}
		step := (end - start) * 1000 / 111
		exp := a.config[len("#C2-F:2400000,0855854,0000,-120,0112,")]
		return setConfig(fmt.Sprintf("#C2-F:%07d,%07d,%s,%s,0112,%c,000,0015000,2700000,0600000,00510,0000,000\r\n", start, step, f[2], f[3], exp))
	case string(cmd) == "CM\x00":
		return setConfig(testConfigMain)
	case string(cmd) == "CM\x01":
		return setConfig(testConfigExp)
	case string(cmd) == "CW\x00":
		// The firmware ignores a zero byte.
		return nil
	case string(cmd) == "CW1":
		return setConfig("#C2-F:2412100,5000000,0000,-120,0013,1,002,0015000,2700000,0600000,00600,0000,000\r\n")
	case string(cmd) == "CW0":
		return setConfig("#C2-F:2412100,0540540,0000,-120,0112,1,000,0015000,2700000,0600000,00578,0000,004\r\n")
	case bytes.HasPrefix(cmd, []byte("C+")):
		return cat("#C+", cmd[2], crlf)
	case bytes.HasPrefix(cmd, []byte("CJ")):
		return setConfig(strings.Replace(a.config, ",0112,", fmt.Sprintf(",%04d,", (int(cmd[2])+1)*16), 1))
	case string(cmd) == "D1":
		a.dumping = true
		return cat("$D", payload(screenDataLen), crlf)
	case string(cmd) == "D0":
		a.dumping = false
	case string(cmd) == "CP\x00":
		var out []byte
		for i := range byte(maxPresetSlots) {
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

func newTestDevice(t *testing.T, a *fakeAnalyzer) (*Device, *rfxtest.Device) {
	t.Helper()
	if a == nil {
		a = &fakeAnalyzer{}
	}
	port := rfxtest.NewDevice(a.handle)
	dev, err := New(testContext(t), port)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dev.Close(); err != nil {
			t.Errorf("Close: %s", err)
		}
	})
	return dev, port
}

// collect reads messages until one satisfies stop.
func collect(t *testing.T, sub *Subscription, stop func(Message) bool) []Message {
	t.Helper()
	var msgs []Message
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg, ok := <-sub.C():
			if !ok {
				t.Fatalf("subscription closed after %d messages", len(msgs))
			}
			msgs = append(msgs, msg)
			if stop(msg) {
				return msgs
			}
		case <-timeout:
			t.Fatalf("timed out after %d messages", len(msgs))
		}
	}
}

func isSerial(sn string) func(Message) bool {
	return func(msg Message) bool {
		m, ok := msg.(*SerialNumber)
		return ok && m.Serial == sn
	}
}

func commandStrings(port *rfxtest.Device) []string {
	var s []string
	for _, cmd := range port.Commands() {
		s = append(s, string(cmd))
	}
	return s
}

var testPreset = Preset{Index: 3, Name: "Test", Start: 1 * MHz, Stop: 2 * MHz, AmpTop: 0, AmpBottom: -100, Iterations: 1}

func TestHandshake(t *testing.T) {
	dev, port := newTestDevice(t, nil)

	info := dev.Info()
	want := DeviceInfo{Main: Model6G, Expansion: ModelWSUB3G, Firmware: "1.34", Serial: "0123456789ABCDEF"}
	if info != want {
		t.Errorf("info %+v, want %+v", info, want)
	}
	if got := dev.ActiveModel(); got != ModelWSUB3G {
		t.Errorf("active model %s, want WSUB3G", got)
	}
	cfg := dev.Config()
	if cfg.Start != 2400*MHz || cfg.Points != 112 || !cfg.ExpansionActive {
		t.Errorf("config %+v", cfg)
	}
	if got := commandStrings(port); !reflect.DeepEqual(got, []string{"C0", "Cq"}) {
		t.Errorf("got commands %q, want C0 then Cq", got)
	}
	if sn, err := dev.SerialNumber(testContext(t)); err != nil || sn != "0123456789ABCDEF" {
		t.Errorf("SerialNumber = %q, %v", sn, err)
	}
	if got := len(port.Commands()); got != 2 {
		t.Errorf("known serial number was requested again")
	}
}

func TestHandshakeTimeout(t *testing.T) {
	port := rfxtest.NewDevice(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	dev, err := New(ctx, port)
	if !errors.Is(err, ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, %v; want a timeout", dev, err)
	}
	// The port must have been closed.
	if _, err := port.Write([]byte("#\x04C0")); err == nil {
		t.Error("port still open after failed handshake")
	}
}

func TestClose(t *testing.T) {
	a := &fakeAnalyzer{}
	dev, err := New(testContext(t), rfxtest.NewDevice(a.handle))
	if err != nil {
		t.Fatal(err)
	}
	// A subscriber that never reads must not hold up Close.
	stalled := dev.Subscribe(SubscribeOptions{})
	if _, err := dev.Resume(testContext(t)); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- dev.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}
	if err := dev.Close(); err != nil {
		t.Errorf("second Close: %s", err)
	}
	if err := dev.Err(); err != nil {
		t.Errorf("Err after Close: %s", err)
	}
	if _, ok := <-stalled.C(); ok {
		t.Error("subscription still open after Close")
	}
	stalled.Close()

	late := dev.Subscribe(SubscribeOptions{})
	if _, ok := <-late.C(); ok {
		t.Error("subscription to a closed device is open")
	}
	late.Close()

	ctx := testContext(t)
	if _, err := dev.Resume(ctx); !errors.Is(err, ErrClosed) {
		t.Errorf("Resume after Close = %v, want ErrClosed", err)
	}
	if err := dev.DeletePreset(ctx, 0); !errors.Is(err, ErrClosed) {
		t.Errorf("DeletePreset after Close = %v, want ErrClosed", err)
	}
}

func TestReadError(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	sub := dev.Subscribe(SubscribeOptions{})
	port.Send([]byte("#Sn0000000000000001\r\n"))
	errUnplugged := errors.New("unplugged")
	port.Fail(errUnplugged)

	// Everything received before the failure is delivered, then the channel
	// closes.
	var last Message
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case msg, ok := <-sub.C():
			if !ok {
				break loop
			}
			last = msg
		case <-timeout:
			t.Fatal("subscription was not closed")
		}
	}
	if !isSerial("0000000000000001")(last) {
		t.Errorf("last message %+v, want the serial number sent before the failure", last)
	}
	if err := dev.Err(); !errors.Is(err, errUnplugged) {
		t.Errorf("Err = %v, want %v", err, errUnplugged)
	}
	if _, err := dev.SetSweep(testContext(t), SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz, AmpBottom: -120}); !errors.Is(err, errUnplugged) {
		t.Errorf("SetSweep = %v, want %v", err, errUnplugged)
	}
}

func TestSubscribe(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	// The reply to the calibration request sent on connect may still arrive.
	all := dev.Subscribe(SubscribeOptions{Kinds: KindAll &^ KindCalibration})
	defer all.Close()
	sweeps := dev.Subscribe(SubscribeOptions{Kinds: KindSweep | KindSerialNumber})
	defer sweeps.Close()
	closed := dev.Subscribe(SubscribeOptions{})
	closed.Close()
	closed.Close()
	if _, ok := <-closed.C(); ok {
		t.Error("closed subscription delivered a message")
	}

	port.Send(cat("$S", 3, 0, 20, 200, crlf, "#C+\x04\r\n", "#QA:0\r\n", "$S", 1, 100, crlf, "#Sn0000000000000009\r\n"))
	end := isSerial("0000000000000009")

	var kinds []Kind
	for _, msg := range collect(t, all, end) {
		kinds = append(kinds, msg.Kind())
	}
	if want := []Kind{KindSweep, KindCalculator, KindInputStage, KindSweep, KindSerialNumber}; !reflect.DeepEqual(kinds, want) {
		t.Errorf("unfiltered: got %v, want %v", kinds, want)
	}

	msgs := collect(t, sweeps, end)
	if len(msgs) != 3 {
		t.Fatalf("filtered: got %d messages, want 3", len(msgs))
	}
	// Sweeps carry the frequency axis of the config they were measured with.
	s, ok := msgs[0].(*Sweep)
	if !ok {
		t.Fatalf("filtered: first message is %T", msgs[0])
	}
	if s.Start != 2400*MHz || s.Step != 855854 || s.Time.IsZero() {
		t.Errorf("sweep start %s step %s time %v", s.Start, s.Step, s.Time)
	}
	if !reflect.DeepEqual(s.Samples, []float32{0, -10, -100}) {
		t.Errorf("sweep samples %v", s.Samples)
	}
	if dev.Config().Calculator != CalculatorMaxHold {
		t.Errorf("calculator change not tracked in Config")
	}
}

// A subscriber that isn't reading must not stall the reader, lose device
// state, or hold up commands or other subscribers.
func TestSlowSubscriber(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	slow := dev.Subscribe(SubscribeOptions{Buffer: 8})
	defer slow.Close()
	fast := dev.Subscribe(SubscribeOptions{Kinds: KindSerialNumber})
	defer fast.Close()

	const sweeps = 1000
	sweep := cat("$S", 4, 1, 2, 3, 4, crlf)
	var stream []byte
	for i := range sweeps {
		stream = append(stream, sweep...)
		if i%100 == 0 {
			stream = append(stream, "#C+\x04\r\n"...)
		}
	}
	port.Send(stream)

	ctx := testContext(t)
	if err := dev.WritePreset(ctx, testPreset); err != nil {
		t.Fatalf("WritePreset with a stalled subscriber: %s", err)
	}
	cfg, err := dev.SetSweep(ctx, SweepRequest{Start: 433 * MHz, Stop: 434 * MHz, AmpTop: -10, AmpBottom: -110})
	if err != nil {
		t.Fatalf("SetSweep with a stalled subscriber: %s", err)
	}
	if cfg.Start != 433*MHz || cfg.AmpTop != -10 {
		t.Errorf("got config %+v", cfg)
	}

	// Mark the end of the stream, then drain.
	port.Send([]byte("#Sn0000000000000002\r\n"))
	end := isSerial("0000000000000002")
	collect(t, fast, end)
	var nSweeps, nModes, nAcks int
	for _, msg := range collect(t, slow, end) {
		switch msg.Kind() {
		case KindSweep:
			nSweeps++
		case KindCalculator:
			nModes++
		case KindPresetsEnd:
			nAcks++
		}
	}
	if nModes != sweeps/100 || nAcks != 1 {
		t.Errorf("got %d calculator modes and %d preset acks, want %d and 1", nModes, nAcks, sweeps/100)
	}
	// One more than the buffer can be in flight to the channel.
	if nSweeps == 0 || nSweeps > 9 {
		t.Errorf("got %d sweeps, want 1 to 9", nSweeps)
	}
	if got := slow.Dropped(); got != uint64(sweeps-nSweeps) {
		t.Errorf("Dropped = %d, want %d", got, sweeps-nSweeps)
	}
	if got := fast.Dropped(); got != 0 {
		t.Errorf("other subscriber dropped %d", got)
	}
}

func TestConcurrentCommands(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)

	const n = 30
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			if err := dev.SetLCD(ctx, true); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() {
			p := testPreset
			p.Index = i
			if err := dev.WritePreset(ctx, p); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	// Interleaved writes would corrupt the command framing.
	var lcd, presets int
	for _, cmd := range port.Commands()[2:] {
		switch {
		case string(cmd) == "L1":
			lcd++
		case len(cmd) == 34 && bytes.HasPrefix(cmd, []byte("CP\x01")):
			presets++
		default:
			t.Errorf("unexpected command %q", cmd)
		}
	}
	if lcd != n || presets != n {
		t.Errorf("got %d LCD and %d preset commands, want %d each", lcd, presets, n)
	}
}

func TestPresets(t *testing.T) {
	dev, _ := newTestDevice(t, nil)
	ctx := testContext(t)

	want := []Preset{
		{Index: 0, Name: "VTX 5.8GHz", Start: 5632 * MHz, Stop: 5958 * MHz, AmpTop: -20, AmpBottom: -90,
			Calculator: CalculatorNormal, Iterations: 4, Mainboard: true, Marker: MarkerPeak},
		{Index: 29, Name: "123456789012", Start: 0, Stop: 9999999 * KHz, AmpTop: 35, AmpBottom: 25,
			Calculator: CalculatorMaxHold, Iterations: 16, Mainboard: false, Marker: MarkerManual},
	}
	for _, p := range want {
		if err := dev.WritePreset(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	temp := testPreset
	temp.Index = 5
	if err := dev.WritePreset(ctx, temp); err != nil {
		t.Fatal(err)
	}
	if err := dev.DeletePreset(ctx, 5); err != nil {
		t.Fatal(err)
	}

	got, err := dev.Presets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	// A standard unit has 30 slots.
	over := testPreset
	over.Index = 30
	if err := dev.WritePreset(ctx, over); err == nil {
		t.Error("preset beyond the device's slots accepted")
	}
	if err := dev.DeletePreset(ctx, 30); err == nil {
		t.Error("delete beyond the device's slots accepted")
	}
	invalid := testPreset
	invalid.Name = ""
	if err := dev.WritePreset(ctx, invalid); err == nil {
		t.Error("invalid preset accepted")
	}
}

func TestPresetSlotsPlus(t *testing.T) {
	dev, _ := newTestDevice(t, &fakeAnalyzer{setup: "#C2-M:010,255,03.28\r\n"})
	p := testPreset
	p.Index = 99
	if err := dev.WritePreset(testContext(t), p); err != nil {
		t.Errorf("slot 99 on a Plus unit: %s", err)
	}
}

func TestRequestTimeout(t *testing.T) {
	port := rfxtest.NewDevice(func(cmd []byte) []byte {
		if string(cmd) == "C0" {
			return []byte(testSetup + testConfigExp)
		}
		return nil
	})
	dev, err := New(testContext(t), port)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // not under test
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := dev.WritePreset(ctx, testPreset); !errors.Is(err, ErrTimeout) {
		t.Errorf("WritePreset = %v, want a timeout", err)
	}
	if _, err := dev.Presets(ctx); !errors.Is(err, ErrTimeout) {
		t.Errorf("Presets = %v, want a timeout", err)
	}
	canceled, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := dev.SetCalculator(canceled, CalculatorMax); !errors.Is(err, context.Canceled) || errors.Is(err, ErrTimeout) {
		t.Errorf("SetCalculator with a canceled context = %v", err)
	}
}

func TestSetSweep(t *testing.T) {
	a := &fakeAnalyzer{}
	dev, port := newTestDevice(t, a)
	ctx := testContext(t)
	req := SweepRequest{Start: 5350 * MHz, Stop: 5950 * MHz, AmpTop: 10, AmpBottom: -100}

	cfg, err := dev.SetSweep(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Start != 5350*MHz || cfg.AmpTop != 10 || cfg.AmpBottom != -100 {
		t.Errorf("got %+v", cfg)
	}
	if dev.Config() != cfg {
		t.Errorf("Config() = %+v, want %+v", dev.Config(), cfg)
	}
	if cmds := commandStrings(port); len(cmds) != 3 || cmds[2] != "C2-F:5350000,5950000,0010,-100" {
		t.Errorf("sent %q", cmds[2:])
	}

	// The device doesn't echo an unchanged configuration, so it is asked.
	cfg, err = dev.SetSweep(ctx, req)
	if err != nil {
		t.Fatalf("unchanged config: %s", err)
	}
	if cfg.Start != 5350*MHz {
		t.Errorf("unchanged config: got %+v", cfg)
	}
	if cmds := commandStrings(port); len(cmds) != 4 || cmds[3] != "C0" {
		t.Errorf("unchanged config: sent %q, want only a config request", cmds[3:])
	}

	// A device that adjusts the range reports a different start frequency.
	a.mu.Lock()
	a.clamp = true
	a.mu.Unlock()
	defer func(d time.Duration) { configEchoTimeout = d }(configEchoTimeout)
	configEchoTimeout = 50 * time.Millisecond
	cfg, err = dev.SetSweep(ctx, SweepRequest{Start: 100 * MHz, Stop: 200 * MHz, AmpBottom: -120})
	if err != nil {
		t.Fatalf("adjusted config: %s", err)
	}
	if cfg.Start != 100100*KHz {
		t.Errorf("adjusted config: got %+v", cfg)
	}

	n := len(port.Commands())
	for _, c := range []struct {
		name string
		req  SweepRequest
	}{
		{"reversed range", SweepRequest{Start: 2500 * MHz, Stop: 2400 * MHz, AmpBottom: -120}},
		{"empty range", SweepRequest{Start: 2400 * MHz, Stop: 2400 * MHz, AmpBottom: -120}},
		{"range under 1 kHz", SweepRequest{Start: 2400 * MHz, Stop: 2400*MHz + 999, AmpBottom: -120}},
		{"negative start", SweepRequest{Start: -1, Stop: 2400 * MHz, AmpBottom: -120}},
		{"stop too high", SweepRequest{Start: 2400 * MHz, Stop: 10 * GHz, AmpBottom: -120}},
		{"top too high", SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz, AmpTop: 36, AmpBottom: -120}},
		{"bottom too low", SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz, AmpBottom: -121}},
		{"range under 10 dB", SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz, AmpTop: -50, AmpBottom: -55}},
		{"no amplitude range", SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz}},
	} {
		if _, err := dev.SetSweep(ctx, c.req); err == nil {
			t.Errorf("%s: no error", c.name)
		}
	}
	if got := len(port.Commands()); got != n {
		t.Errorf("%d commands sent for invalid requests", got-n)
	}
}

func TestSelectModule(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)

	cfg, err := dev.SelectModule(ctx, ModuleMain)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExpansionActive || cfg.Start != 5350*MHz {
		t.Errorf("got %+v", cfg)
	}
	if got := dev.ActiveModel(); got != Model6G {
		t.Errorf("active model %s, want 6G", got)
	}
	// The 6G table starts at offset 134 of the calibration data.
	if cal := dev.Calibration(); len(cal) != 20 || cal[0] != 4 {
		t.Errorf("calibration for 6G: %v", cal)
	}

	// Selecting the active module only asks for the config.
	if _, err := dev.SelectModule(ctx, ModuleMain); err != nil {
		t.Fatal(err)
	}
	if cmds := commandStrings(port); !reflect.DeepEqual(cmds[2:], []string{"CM\x00", "C0"}) {
		t.Errorf("sent %q", cmds[2:])
	}

	cfg, err = dev.SelectModule(ctx, ModuleExpansion)
	if err != nil || !cfg.ExpansionActive {
		t.Errorf("got %+v, %v", cfg, err)
	}
	if cal := dev.Calibration(); len(cal) != 154 {
		t.Errorf("calibration for WSUB3G has %d entries", len(cal))
	}
	if _, err := dev.SelectModule(ctx, 2); err == nil {
		t.Error("invalid module accepted")
	}
}

// A device that answers but ignores a mode change must not be reported as
// having applied it.
func TestNotApplied(t *testing.T) {
	port := rfxtest.NewDevice(func(cmd []byte) []byte {
		if string(cmd) == "C0" {
			return []byte(testSetup + testConfigExp)
		}
		return nil
	})
	dev, err := New(testContext(t), port)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // not under test
	defer func(d time.Duration) { configEchoTimeout = d }(configEchoTimeout)
	configEchoTimeout = 50 * time.Millisecond

	cfg, err := dev.SelectModule(testContext(t), ModuleMain)
	if !errors.Is(err, ErrNotApplied) {
		t.Errorf("SelectModule = %v, want ErrNotApplied", err)
	}
	if !cfg.ExpansionActive {
		t.Errorf("returned config %+v is not the device's", cfg)
	}
	// A range the device adjusts is returned as reported, without an error.
	if _, err := dev.SetSweep(testContext(t), SweepRequest{Start: 1 * GHz, Stop: 2 * GHz, AmpBottom: -120}); err != nil {
		t.Errorf("SetSweep = %v", err)
	}
}

func TestCommands(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)
	n := len(port.Commands())
	for _, c := range []struct {
		name string
		send func() error
		want string
	}{
		{"calculator", func() error { return dev.SetCalculator(ctx, CalculatorMaxHistorical) }, "C+\x05"},
		{"hold", func() error { return dev.Hold(ctx) }, "CH"},
		{"lcd off", func() error { return dev.SetLCD(ctx, false) }, "L0"},
		{"lcd on", func() error { return dev.SetLCD(ctx, true) }, "L1"},
		{"screen dump on", func() error { return dev.SetScreenDump(ctx, true) }, "D1"},
		{"screen dump off", func() error { return dev.SetScreenDump(ctx, false) }, "D0"},
		{"tracking step", func() error { return dev.SetTrackingStep(ctx, 0x1234) }, "k\x12\x34"},
		{"baud rate 500000", func() error { return dev.SetBaudRate(ctx, BaudRate500000) }, "c0"},
		{"baud rate 115200", func() error { return dev.SetBaudRate(ctx, BaudRate115200) }, "c8"},
		{"shutdown", func() error { return dev.Shutdown(ctx) }, "S"},
		{"raw", func() error { return dev.SendRaw(ctx, []byte("Cr")) }, "Cr"},
	} {
		if err := c.send(); err != nil {
			t.Errorf("%s: %s", c.name, err)
			continue
		}
		cmds := commandStrings(port)
		if len(cmds) != n+1 || cmds[n] != c.want {
			t.Errorf("%s: sent %q, want %q", c.name, cmds[n:], c.want)
		}
		n = len(cmds)
	}
	if dev.Config().Calculator != CalculatorMaxHistorical {
		t.Error("calculator mode not tracked")
	}

	if err := dev.SetCalculator(ctx, CalculatorInvalid); err == nil {
		t.Error("invalid calculator mode accepted")
	}
	if err := dev.SetBaudRate(ctx, 12345); err == nil {
		t.Error("invalid baud rate accepted")
	}
	if err := dev.SendRaw(ctx, make([]byte, 254)); err == nil {
		t.Error("oversized command accepted")
	}
	if got := len(port.Commands()); got != n {
		t.Errorf("%d commands sent for invalid arguments", got-n)
	}
}

func TestWifiMode(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)
	cfg, err := dev.SetWifiMode(ctx, Wifi24GHz)
	if err != nil || cfg.Mode != ModeWifiAnalyzer || cfg.Points != 13 {
		t.Fatalf("got %+v, %v", cfg, err)
	}
	if dev.SupportsHighRes() {
		t.Error("high resolution reported in WiFi mode")
	}
	cfg, err = dev.SetWifiMode(ctx, WifiDisabled)
	if err != nil || cfg.Mode != ModeSpectrumAnalyzer {
		t.Fatalf("got %+v, %v", cfg, err)
	}
	if cmds := commandStrings(port); !reflect.DeepEqual(cmds[2:], []string{"CW1", "CW0"}) {
		t.Errorf("sent %q", cmds[2:])
	}
	// Disabling again only confirms the state.
	if cfg, err := dev.SetWifiMode(ctx, WifiDisabled); err != nil || cfg.Mode != ModeSpectrumAnalyzer {
		t.Errorf("got %+v, %v", cfg, err)
	}
	if cmds := commandStrings(port); cmds[len(cmds)-1] != "C0" {
		t.Errorf("sent %q", cmds[4:])
	}
	if _, err := dev.SetWifiMode(ctx, 3); err == nil {
		t.Error("invalid WiFi mode accepted")
	}
}

func TestSweepPoints(t *testing.T) {
	ctx := testContext(t)

	// A 6G with a WSUB3G expansion has a fixed resolution.
	dev, port := newTestDevice(t, nil)
	if dev.SupportsHighRes() || dev.SupportsInputStage() {
		t.Error("6G + WSUB3G reports high resolution or input stage support")
	}
	if _, err := dev.SetSweepPoints(ctx, 240); !errors.Is(err, ErrUnsupported) {
		t.Errorf("got %v, want ErrUnsupported", err)
	}
	if got := len(port.Commands()); got != 2 {
		t.Errorf("unsupported command was sent")
	}

	plus, port := newTestDevice(t, &fakeAnalyzer{setup: "#C2-M:010,255,03.28\r\n", config: testConfigMain})
	if !plus.SupportsHighRes() || !plus.SupportsInputStage() {
		t.Error("WSUB1G+ doesn't report high resolution and input stage support")
	}
	cfg, err := plus.SetSweepPoints(ctx, 240)
	if err != nil || cfg.Points != 240 {
		t.Fatalf("got %+v, %v", cfg, err)
	}
	if cmds := commandStrings(port); cmds[2] != "CJ\x0e" {
		t.Errorf("sent %q", cmds[2:])
	}
	for _, points := range []int{111, MaxSweepPoints + 1} {
		if _, err := plus.SetSweepPoints(ctx, points); err == nil {
			t.Errorf("%d points accepted", points)
		}
	}
}

func TestScreenshot(t *testing.T) {
	a := &fakeAnalyzer{}
	dev, port := newTestDevice(t, a)
	img, err := dev.Screenshot(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(img.Data, payload(screenDataLen)) {
		t.Error("screenshot data differs from what the device sent")
	}
	if cmds := commandStrings(port); !reflect.DeepEqual(cmds[2:], []string{"D1", "D0"}) {
		t.Errorf("sent %q, want the dump turned on then off", cmds[2:])
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.dumping {
		t.Error("screen dump left on")
	}
}

func TestPresetValidate(t *testing.T) {
	valid := Preset{Index: 1, Name: "2.4G WiFi", Start: 2401 * MHz, Stop: 2495 * MHz, AmpTop: 0, AmpBottom: -100, Iterations: 4}
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
		{"reversed range", func(p *Preset) { p.Start, p.Stop = 2*MHz, 1*MHz }},
		{"frequency too high", func(p *Preset) { p.Stop = 10 * GHz }},
		{"top too high", func(p *Preset) { p.AmpTop = 36 }},
		{"bottom too low", func(p *Preset) { p.AmpBottom = -121 }},
		{"range under 10 dB", func(p *Preset) { p.AmpTop, p.AmpBottom = -50, -59 }},
		{"no iterations", func(p *Preset) { p.Iterations = 0 }},
		{"too many iterations", func(p *Preset) { p.Iterations = 17 }},
		{"invalid calculator mode", func(p *Preset) { p.Calculator = 6 }},
		{"invalid marker mode", func(p *Preset) { p.Marker = 3 }},
	} {
		p := valid
		c.edit(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: no error", c.name)
		}
	}
}
