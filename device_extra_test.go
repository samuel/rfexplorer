package rfx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"samuelks.com/rfx/rfxtest"
)

// sniffingAnalyzer extends fakeAnalyzer with sniffer mode, input stage and
// amplitude offset, as a WSUB3G-equipped unit with an input stage would
// answer them.
type sniffingAnalyzer struct {
	fakeAnalyzer
	sniffer    string // current sniffer config line; empty in analyzer mode
	inputStage byte
	echoStage  bool // answer the input stage command with #QA
	offset     int
}

func (a *sniffingAnalyzer) handle(cmd []byte) []byte {
	switch {
	case bytes.HasPrefix(cmd, []byte("C4-F:")):
		// C4-F:<freq>,<rbw>,<threshold>,<delay> answered with
		// #C4-F:<freq>,<exp>,006,<delay>,0,<rbw>,<threshold>,
		f := strings.Split(string(cmd[5:]), ",")
		a.sniffer = fmt.Sprintf("#C4-F:%s,1,006,%s,0,%s,%s,\r\n", f[0], f[3], f[1], f[2])
		return []byte(a.sniffer)
	case string(cmd) == "C0" && a.sniffer != "":
		return []byte("#C2-M:006,005,01.34\r\n#QA:" + strconv.Itoa(int(a.inputStage)) + "\r\n" + a.sniffer)
	case string(cmd) == "C0":
		out := a.fakeAnalyzer.handle(cmd)
		return bytes.Replace(out, []byte("#QA:0"), []byte("#QA:"+strconv.Itoa(int(a.inputStage))), 1)
	case len(cmd) == 2 && cmd[0] == 'a':
		a.inputStage = cmd[1] - '0'
		if a.echoStage {
			return []byte("#QA:" + string(cmd[1]) + "\r\n")
		}
		return nil
	case len(cmd) == 3 && string(cmd[:2]) == "CO":
		a.offset = int(int8(cmd[2]))
		a.mu.Lock()
		defer a.mu.Unlock()
		f := strings.Split(strings.TrimSuffix(a.config, "\r\n"), ",")
		f[11] = fmt.Sprintf("%04d", a.offset)
		a.config = strings.Join(f, ",") + "\r\n"
		return []byte(a.config)
	}
	return a.fakeAnalyzer.handle(cmd)
}

func newSniffingDevice(t *testing.T, a *sniffingAnalyzer) (*Device, *rfxtest.Device) {
	t.Helper()
	port := rfxtest.NewDevice(a.handle)
	dev, err := New(testContext(t), port)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() }) //nolint:errcheck // test cleanup
	return dev, port
}

func TestSniffer(t *testing.T) {
	a := &sniffingAnalyzer{}
	dev, port := newSniffingDevice(t, a)
	ctx := testContext(t)
	if _, ok := dev.Sniffer(); ok {
		t.Fatal("sniffer mode reported before switching")
	}

	sub := dev.Subscribe(SubscribeOptions{Kinds: KindRawData})
	defer sub.Close()

	cfg, err := dev.SetSniffer(ctx, SnifferRequest{Frequency: 315060 * KHz, RBW: 225 * KHz, ThresholdDBm: -50, Delay: 266})
	if err != nil {
		t.Fatal(err)
	}
	want := SnifferConfig{Start: 315060 * KHz, ExpansionActive: true, Mode: ModeRFSniffer, Delay: 266, RBW: 225 * KHz, ThresholdDBm: -50}
	if cfg != want {
		t.Errorf("config %+v, want %+v", cfg, want)
	}
	if got, ok := dev.Sniffer(); !ok || got != want {
		t.Errorf("Sniffer() = %+v, %v", got, ok)
	}
	if cmds := commandStrings(port); cmds[len(cmds)-1] != "C4-F:0315060,00225,100,00266" {
		t.Errorf("sent %q", cmds[len(cmds)-1])
	}

	port.Send(cat("$R", 2, 0, 0xf0, 0x81, crlf))
	msg := collect(t, sub, func(Message) bool { return true })[0].(*RawData)
	if msg.Time.IsZero() {
		t.Error("capture not timestamped")
	}
	wantPulses := []Pulse{{true, 4}, {false, 4}, {true, 1}, {false, 6}, {true, 1}}
	if got := msg.Pulses(); !slices.Equal(got, wantPulses) {
		t.Errorf("pulses %v, want %v", got, wantPulses)
	}
	if msg.Len() != 16 || !msg.Bit(0) || msg.Bit(4) || !msg.Bit(15) {
		t.Error("bits read in the wrong order")
	}

	// A device in sniffer mode connects too.
	dev2, err := New(testContext(t), rfxtest.NewDevice(a.handle))
	if err != nil {
		t.Fatalf("connecting in sniffer mode: %s", err)
	}
	defer dev2.Close() //nolint:errcheck // test cleanup
	if _, ok := dev2.Sniffer(); !ok {
		t.Error("sniffer mode not detected on connect")
	}

	for _, req := range []SnifferRequest{
		{Frequency: 0, RBW: 225 * KHz, Delay: 1},
		{Frequency: 315 * MHz, RBW: 0, Delay: 1},
		{Frequency: 315 * MHz, RBW: 225 * KHz, ThresholdDBm: -128, Delay: 1},
		{Frequency: 315 * MHz, RBW: 225 * KHz, ThresholdDBm: 1, Delay: 1},
		{Frequency: 315 * MHz, RBW: 225 * KHz, Delay: 0},
		{Frequency: 315 * MHz, RBW: 225 * KHz, Delay: 100000},
	} {
		if _, err := dev.SetSniffer(ctx, req); err == nil {
			t.Errorf("invalid request %+v accepted", req)
		}
	}
}

func TestGeneratorUnsupported(t *testing.T) {
	port := rfxtest.NewDevice(func(cmd []byte) []byte {
		if string(cmd) == "C0" {
			return []byte("#C3-M:060,255,01.15\r\n#C3-G:0186525,0186525,-030,0001000,0001,0,0,0000,0,0,0,0,00000\r\n")
		}
		return nil
	})
	_, err := New(testContext(t), port)
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "generator") {
		t.Fatalf("got %v, want ErrUnsupported for a generator", err)
	}
}

func TestInputStage(t *testing.T) {
	defer func(d time.Duration) { configEchoTimeout = d }(configEchoTimeout)
	configEchoTimeout = 50 * time.Millisecond
	ctx := testContext(t)

	// A 6G with a WSUB3G expansion has no input stage.
	dev, _ := newSniffingDevice(t, &sniffingAnalyzer{})
	if err := dev.SetInputStage(ctx, InputStageLNA25dB); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("got %v, want ErrUnsupported", err)
	}

	for _, echo := range []bool{true, false} {
		a := &sniffingAnalyzer{echoStage: echo}
		a.setup = "#C2-M:014,255,01.34\r\n" // 6G+
		a.config = testConfigMain
		dev, port := newSniffingDevice(t, a)
		if err := dev.SetInputStage(ctx, InputStageLNA25dB); err != nil {
			t.Fatalf("echo %v: %s", echo, err)
		}
		if got := dev.InputStage(); got != InputStageLNA25dB {
			t.Errorf("echo %v: input stage %s", echo, got)
		}
		cmds := commandStrings(port)
		if !slices.Contains(cmds, "a2") {
			t.Errorf("echo %v: sent %q", echo, cmds)
		}
		if got := slices.Contains(cmds[2:], "C0"); got == echo {
			t.Errorf("echo %v: asked for the state: %v", echo, got)
		}
	}

	// A device that ignores the command.
	a := &sniffingAnalyzer{}
	a.setup = "#C2-M:014,255,01.34\r\n"
	a.config = testConfigMain
	ignore := rfxtest.NewDevice(func(cmd []byte) []byte {
		if cmd[0] == 'a' {
			return nil
		}
		return a.handle(cmd)
	})
	dev, err := New(ctx, ignore)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // test cleanup
	if err := dev.SetInputStage(ctx, InputStageAttenuator30dB); !errors.Is(err, ErrNotApplied) {
		t.Errorf("got %v, want ErrNotApplied", err)
	}
	if err := dev.SetInputStage(ctx, 9); err == nil {
		t.Error("invalid input stage accepted")
	}
}

func TestAmplitudeOffsetAndDSP(t *testing.T) {
	dev, port := newSniffingDevice(t, &sniffingAnalyzer{})
	ctx := testContext(t)
	cfg, err := dev.SetAmplitudeOffset(ctx, -20)
	if err != nil || cfg.AmpOffset != -20 {
		t.Fatalf("got %+v, %v", cfg, err)
	}
	if err := dev.SetDSP(ctx, DSPFast); err != nil {
		t.Fatal(err)
	}
	cmds := commandStrings(port)
	if want := []string{"CO\xec", "Cp2"}; !reflect.DeepEqual(cmds[len(cmds)-2:], want) {
		t.Errorf("sent %q, want %q", cmds[len(cmds)-2:], want)
	}
	if _, err := dev.SetAmplitudeOffset(ctx, 101); err == nil {
		t.Error("offset out of range accepted")
	}
	if err := dev.SetDSP(ctx, 4); err == nil {
		t.Error("invalid DSP mode accepted")
	}
}

func TestTracking(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	sub := dev.Subscribe(SubscribeOptions{Kinds: KindTracking})
	defer sub.Close()
	port.Send([]byte("#K1\r\n"))
	collect(t, sub, func(Message) bool { return true })
	if !dev.Tracking() {
		t.Error("tracking mode not tracked")
	}
	port.Send([]byte("#K0\r\n"))
	collect(t, sub, func(Message) bool { return true })
	if dev.Tracking() {
		t.Error("tracking mode still on")
	}
}

func TestPresetJSON(t *testing.T) {
	p := testPreset
	p.Calculator = CalculatorMaxHold
	p.Marker = MarkerManual
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"index":3,"name":"Test","start":"1MHz","stop":"2MHz","amp_top":0,"amp_bottom":-100,"calculator":"MaxHold","iterations":1,"mainboard":false,"marker":"Manual"}`
	if string(b) != want {
		t.Fatalf("got %s\nwant %s", b, want)
	}
	var got Preset
	if err := json.Unmarshal([]byte(strings.ReplaceAll(want, `"MaxHold"`, `"maxhold"`)), &got); err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("round trip %+v, want %+v", got, p)
	}
	if err := json.Unmarshal([]byte(`{"calculator":"Sideways"}`), &got); err == nil {
		t.Error("unknown calculator mode accepted")
	}
	if _, err := json.Marshal(Preset{Calculator: CalculatorInvalid}); err == nil {
		t.Error("invalid calculator mode marshaled")
	}
}

func TestOpenDetectsBaudRate(t *testing.T) {
	defer func(f func(string, BaudRate) (io.ReadWriteCloser, error)) { openPort = f }(openPort)
	var tried []BaudRate
	openPort = func(path string, baud BaudRate) (io.ReadWriteCloser, error) {
		tried = append(tried, baud)
		if path != "/dev/test" {
			return nil, fmt.Errorf("unexpected path %q", path)
		}
		if baud != BaudRate115200 {
			// At the wrong rate the device's answers are garbage.
			return rfxtest.NewDevice(nil), nil
		}
		a := &fakeAnalyzer{}
		return rfxtest.NewDevice(a.handle), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	dev, err := Open(ctx, "/dev/test", WithBaudRate(0))
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // test cleanup
	if want := []BaudRate{DefaultBaudRate, BaudRate115200}; !slices.Equal(tried, want) {
		t.Errorf("tried %v, want %v", tried, want)
	}

	// An explicit rate is the only one tried.
	tried = nil
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := Open(ctx, "/dev/test", WithBaudRate(BaudRate9600)); !errors.Is(err, ErrTimeout) {
		t.Errorf("got %v, want a timeout", err)
	}
	if !slices.Equal(tried, []BaudRate{BaudRate9600}) {
		t.Errorf("tried %v", tried)
	}
}

func TestOpenDefaultBaudRateOnly(t *testing.T) {
	defer func(f func(string, BaudRate) (io.ReadWriteCloser, error)) { openPort = f }(openPort)
	var tried []BaudRate
	openPort = func(path string, baud BaudRate) (io.ReadWriteCloser, error) {
		tried = append(tried, baud)
		return rfxtest.NewDevice(nil), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := Open(ctx, "/dev/test"); !errors.Is(err, ErrTimeout) {
		t.Errorf("got %v, want a timeout", err)
	}
	// Commands at a wrong rate could confuse the device, so other rates
	// are only tried when asked.
	if !slices.Equal(tried, []BaudRate{DefaultBaudRate}) {
		t.Errorf("tried %v", tried)
	}
}
