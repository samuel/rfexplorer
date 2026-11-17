package rfx

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var (
	test6G     = Board{Module: ModuleMain, Model: Model6G, Min: 4850 * MHz, Max: 6100 * MHz, MaxSpan: 600 * MHz}
	testWSUB3G = Board{Module: ModuleExpansion, Model: ModelWSUB3G, Min: 15 * MHz, Max: 2700 * MHz, MaxSpan: 600 * MHz, Reported: true}
)

func TestPickBoard(t *testing.T) {
	plusMain := Board{Module: ModuleMain, Model: ModelWSUB1GPlus, Min: 50 * KHz, Max: 960 * MHz, MaxSpan: 960 * MHz}
	plusExp := Board{Module: ModuleExpansion, Model: Model6GPlus, Min: 240 * MHz, Max: 6100 * MHz, MaxSpan: 960 * MHz}
	unknown := Board{Module: ModuleMain, Model: ModelW5G3G}
	for _, c := range []struct {
		name        string
		boards      []Board
		active      Module
		start, stop Frequency
		want        Module
		ok          bool
	}{
		{"active covers", []Board{test6G, testWSUB3G}, ModuleExpansion, 2400 * MHz, 2500 * MHz, ModuleExpansion, true},
		{"other covers", []Board{test6G, testWSUB3G}, ModuleExpansion, 5650 * MHz, 5950 * MHz, ModuleMain, true},
		{"between boards", []Board{test6G, testWSUB3G}, ModuleExpansion, 3 * GHz, 3100 * MHz, 0, false},
		{"span too wide", []Board{test6G, testWSUB3G}, ModuleMain, 4900 * MHz, 6 * GHz, 0, false},
		{"overlap prefers active", []Board{plusMain, plusExp}, ModuleExpansion, 400 * MHz, 500 * MHz, ModuleExpansion, true},
		{"overlap keeps main", []Board{plusMain, plusExp}, ModuleMain, 400 * MHz, 500 * MHz, ModuleMain, true},
		{"single board", []Board{testWSUB3G}, ModuleExpansion, 5 * GHz, 5100 * MHz, 0, false},
		{"unknown active board", []Board{unknown}, ModuleMain, 5 * GHz, 5100 * MHz, ModuleMain, true},
		{"no boards", nil, ModuleMain, 5 * GHz, 5100 * MHz, ModuleMain, true},
	} {
		b, ok := pickBoard(c.boards, c.active, func(b Board) bool { return b.Covers(c.start, c.stop) })
		if ok != c.ok || (ok && b.Module != c.want) {
			t.Errorf("%s: got %s, %v; want %s, %v", c.name, b.Module, ok, c.want, c.ok)
		}
	}
}

func TestRangeError(t *testing.T) {
	boards := []Board{test6G, testWSUB3G}
	for _, c := range []struct {
		err  *RangeError
		want string
	}{
		{&RangeError{Start: 3 * GHz, Stop: 4 * GHz, Boards: boards},
			"rfx: no board can sweep 3GHz to 4GHz; boards: 6G (Main) 4.85GHz to 6.1GHz, max span 600MHz, WSUB3G (Expansion) 15MHz to 2.7GHz, max span 600MHz"},
		{&RangeError{Start: 4900 * MHz, Stop: 6 * GHz, Boards: boards},
			"rfx: span 1.1GHz exceeds the 6G board's max span 600MHz"},
		{&RangeError{Start: 3 * GHz, Stop: 3 * GHz},
			"rfx: no board can tune to 3GHz"},
	} {
		if got := c.err.Error(); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
		if !errors.Is(c.err, ErrOutOfRange) {
			t.Errorf("%q doesn't match ErrOutOfRange", c.err)
		}
	}
}

func TestBoards(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)

	// The active WSUB3G is reported; the 6G has its published limits.
	if got, want := dev.Boards(), []Board{test6G, testWSUB3G}; !reflect.DeepEqual(got, want) {
		t.Errorf("Boards() = %+v, want %+v", got, want)
	}
	if b, err := dev.BoardFor(5650*MHz, 5950*MHz); err != nil || b.Module != ModuleMain {
		t.Errorf("BoardFor(5.8GHz) = %+v, %v", b, err)
	}

	// Ranges no board can sweep are rejected without a command.
	n := len(port.Commands())
	var rangeErr *RangeError
	for _, req := range []SweepRequest{
		{Start: 3 * GHz, Stop: 4 * GHz, AmpBottom: -120},
		{Start: 4900 * MHz, Stop: 6 * GHz, AmpBottom: -120},
		{Start: 1 * MHz, Stop: 10 * MHz, AmpBottom: -120},
	} {
		if _, err := dev.SetSweep(ctx, req); !errors.As(err, &rangeErr) || !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetSweep(%s to %s) = %v, want a RangeError", req.Start, req.Stop, err)
		}
	}
	if got := len(port.Commands()); got != n {
		t.Errorf("%d commands sent for out of range requests", got-n)
	}

	// Selecting the 6G replaces its published limits with reported ones.
	if _, err := dev.SetSweep(ctx, SweepRequest{Start: 5650 * MHz, Stop: 5950 * MHz, AmpBottom: -120}); err != nil {
		t.Fatal(err)
	}
	reported := test6G
	reported.Reported = true
	if got := dev.Boards()[0]; got != reported {
		t.Errorf("main board %+v, want %+v", got, reported)
	}
	// A range only the WSUB3G covers selects it again.
	if _, err := dev.SetSweep(ctx, SweepRequest{Start: 2400 * MHz, Stop: 2500 * MHz, AmpBottom: -120}); err != nil {
		t.Fatal(err)
	}
	if cmds := commandStrings(port)[n:]; !reflect.DeepEqual(cmds, []string{"CM\x00", "C2-F:5650000,5950000,0000,-120", "CM\x01", "C2-F:2400000,2500000,0000,-120"}) {
		t.Errorf("sent %q", cmds)
	}
}

func TestBoardsSingle(t *testing.T) {
	dev, _ := newTestDevice(t, &fakeAnalyzer{setup: "#C2-M:014,255,01.34\r\n", config: testConfigMain})
	boards := dev.Boards()
	if len(boards) != 1 || boards[0].Model != Model6GPlus || !boards[0].Reported {
		t.Errorf("Boards() = %+v, want only the reported 6G+", boards)
	}
}

func TestWifiModeBoard(t *testing.T) {
	dev, port := newTestDevice(t, nil)
	ctx := testContext(t)
	n := len(port.Commands())
	// 5 GHz WiFi needs the 6G board. The fake doesn't answer CW2.
	_, _ = dev.SetWifiMode(ctx, Wifi5GHz) //nolint:errcheck // only the commands matter
	if cmds := commandStrings(port)[n:]; len(cmds) < 2 || cmds[0] != "CM\x00" || cmds[1] != "CW2" {
		t.Errorf("sent %q", cmds)
	}

	// Boards can't be switched in WiFi mode.
	dev, port = newTestDevice(t, nil)
	if _, err := dev.SetWifiMode(ctx, Wifi24GHz); err != nil {
		t.Fatal(err)
	}
	n = len(port.Commands())
	if _, err := dev.SetWifiMode(ctx, Wifi5GHz); err == nil || !strings.Contains(err.Error(), "WiFi mode") {
		t.Errorf("SetWifiMode in WiFi mode = %v", err)
	}
	if got := len(port.Commands()); got != n {
		t.Errorf("sent %q", commandStrings(port)[n:])
	}
}

func TestSnifferBoard(t *testing.T) {
	a := &sniffingAnalyzer{}
	a.config = testConfigMain
	dev, port := newSniffingDevice(t, a)
	ctx := testContext(t)
	if _, err := dev.SetSniffer(ctx, SnifferRequest{Frequency: 315 * MHz, RBW: 225 * KHz, ThresholdDBm: -50, Delay: 266}); err != nil {
		t.Fatal(err)
	}
	cmds := commandStrings(port)
	if i := slices.Index(cmds, "CM\x01"); i < 0 || cmds[len(cmds)-1] != "C4-F:0315000,00225,100,00266" {
		t.Errorf("sent %q", cmds)
	}
	if got := dev.ActiveModel(); got != ModelWSUB3G {
		t.Errorf("active model %s while sniffing", got)
	}
	// The 6G can't be selected in sniffer mode.
	if _, err := dev.SetSniffer(ctx, SnifferRequest{Frequency: 5800 * MHz, RBW: 225 * KHz, ThresholdDBm: -50, Delay: 266}); err == nil {
		t.Error("board switched in sniffer mode")
	}
}

func TestPresetRange(t *testing.T) {
	dev, _ := newTestDevice(t, nil)
	ctx := testContext(t)
	p := testPreset
	p.Start, p.Stop = 2800*MHz, 2900*MHz
	if err := dev.WritePreset(ctx, p); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("preset outside the WSUB3G = %v, want ErrOutOfRange", err)
	}
	// The 6G's limits haven't been reported, so they aren't checked.
	p.Mainboard = true
	if err := dev.WritePreset(ctx, p); err != nil {
		t.Errorf("preset for the unreported 6G = %v", err)
	}
}
