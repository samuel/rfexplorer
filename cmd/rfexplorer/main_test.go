package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/record"
	"samuelks.com/rfx/rfxtest"
)

const (
	testSetup  = "#C2-M:006,005,01.34\r\n"
	testConfig = "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// recordDevice records a session with a fake 6G + WSUB3G for -replay.
func recordDevice(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.rfxrec")
	port := rfxtest.NewDevice(func(cmd []byte) []byte {
		if string(cmd) == "C0" {
			return []byte(testSetup + testConfig + "#Sn0123456789ABCDEF\r\n")
		}
		return nil
	})
	rec, err := record.Create(path, port)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := rfx.New(testContext(t), rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInfoFromReplay(t *testing.T) {
	path := recordDevice(t)
	var out bytes.Buffer
	if err := run(testContext(t), []string{"info", "-replay", path}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Main board:      6G", "Expansion board: WSUB3G", "0123-4567-89AB-CDEF", "Sweep:           2.4GHz to 2.49"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestRunErrors(t *testing.T) {
	ctx := testContext(t)
	if err := run(ctx, []string{"bogus"}, &bytes.Buffer{}); err == nil {
		t.Error("unknown command accepted")
	}
	if err := run(ctx, []string{"presets"}, &bytes.Buffer{}); err == nil {
		t.Error("presets without an action accepted")
	}
	if err := run(ctx, []string{"sniff", "-replay", "x"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "-freq") {
		t.Errorf("sniff without -freq: %v", err)
	}
	if err := run(ctx, []string{"info", "-replay", "x", "-record", "y"}, &bytes.Buffer{}); err == nil {
		t.Error("-record with -replay accepted")
	}
}

func TestPresetFiles(t *testing.T) {
	presets := []rfx.Preset{
		{Index: 0, Name: "WiFi", Start: 2400 * rfx.MHz, Stop: 2500 * rfx.MHz, AmpTop: 0, AmpBottom: -120,
			Calculator: rfx.CalculatorMaxHold, Iterations: 4, Mainboard: false, Marker: rfx.MarkerPeak},
		{Index: 5, Name: "433", Start: 433 * rfx.MHz, Stop: 435 * rfx.MHz, AmpTop: -20, AmpBottom: -100,
			Iterations: 1, Mainboard: false, Marker: rfx.MarkerNone},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "presets.json")
	data, err := json.MarshalIndent(presets, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readPresets(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != presets[0] || got[1] != presets[1] {
		t.Fatalf("read %+v", got)
	}

	var out bytes.Buffer
	if err := listPresets(&out, got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "6     433") || !strings.Contains(out.String(), "MaxHold") {
		t.Errorf("list:\n%s", out.String())
	}

	for name, content := range map[string]string{
		"duplicate slot": `[{"index":1,"name":"a","start":"1MHz","stop":"2MHz","amp_bottom":-100,"iterations":1},
			{"index":1,"name":"b","start":"1MHz","stop":"2MHz","amp_bottom":-100,"iterations":1}]`,
		"invalid preset": `[{"index":1,"name":"","start":"1MHz","stop":"2MHz","amp_bottom":-100,"iterations":1}]`,
		"bad mode":       `[{"index":1,"name":"a","calculator":"Fast"}]`,
	} {
		p := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readPresets(p); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestPresetFromFlags(t *testing.T) {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	p, err := presetFromFlags(fs, []string{"-slot", "3", "-name", "Remote", "-start", "27MHz", "-stop", "28MHz",
		"-calculator", "maxhold", "-board", "expansion", "-marker", "none"})
	if err != nil {
		t.Fatal(err)
	}
	want := rfx.Preset{Index: 2, Name: "Remote", Start: 27 * rfx.MHz, Stop: 28 * rfx.MHz, AmpTop: 0, AmpBottom: -100,
		Calculator: rfx.CalculatorMaxHold, Iterations: 4, Marker: rfx.MarkerNone}
	if p != want {
		t.Fatalf("got %+v, want %+v", p, want)
	}

	fs = flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(&bytes.Buffer{})
	if _, err := presetFromFlags(fs, []string{"-slot", "3", "-name", "x"}); err == nil {
		t.Error("missing frequencies accepted")
	}
}

func TestWritePresets(t *testing.T) {
	var written []string
	port := rfxtest.NewDevice(func(cmd []byte) []byte {
		switch {
		case string(cmd) == "C0":
			return []byte(testSetup + testConfig)
		case bytes.HasPrefix(cmd, []byte("CP\x01")):
			written = append(written, strings.TrimRight(string(cmd[4:16]), "\x00"))
			return []byte("#PCK\r\n")
		}
		return nil
	})
	dev, err := rfx.New(testContext(t), port)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // test cleanup

	ok := rfx.Preset{Index: 0, Name: "ok", Start: 100 * rfx.MHz, Stop: 200 * rfx.MHz, AmpBottom: -100, Iterations: 1}
	beyond := ok
	beyond.Index, beyond.Name = 30, "beyond"
	if err := writePresets(testContext(t), dev, []rfx.Preset{ok, beyond}, &bytes.Buffer{}); err == nil {
		t.Fatal("slot beyond a standard unit's 30 accepted")
	}
	if len(written) != 0 {
		t.Fatalf("wrote %q before checking every preset", written)
	}
	var out bytes.Buffer
	if err := writePresets(testContext(t), dev, []rfx.Preset{ok}, &out); err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0] != "ok" || !strings.Contains(out.String(), "Wrote slot 1: ok") {
		t.Errorf("wrote %q, output %q", written, out.String())
	}
}

func TestCaptureFormats(t *testing.T) {
	raw := &rfx.RawData{Data: []byte{0xf0, 0x81}}
	for format, want := range map[string]string{
		"pulses": "+4 -4 +1 -6 +1",
		"bits":   "1111000010000001",
		"hex":    "f081",
	} {
		if got := captureFormats[format](raw); got != want {
			t.Errorf("%s: got %q, want %q", format, got, want)
		}
	}
}
