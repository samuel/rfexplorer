package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"samuelks.com/rfx"
	"samuelks.com/rfx/bands"
	"samuelks.com/rfx/trace"
)

func newTestTUI(t *testing.T, width, height int) *tui {
	t.Helper()
	scr, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: vt.Col(width), Y: vt.Row(height)}))
	if err != nil {
		t.Fatal(err)
	}
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(scr.Fini)
	return &tui{
		screen:       scr,
		config:       rfx.SweepConfig{AmpTop: 0, AmpBottom: -100},
		wifiChannels: channelsOf(bands.Builtin(bands.NameWiFi24)),
		vtxChannels:  channelsOf(bands.Builtin(bands.NameFPV58)),
	}
}

func (t *tui) row(y int) string {
	w, _ := t.screen.Size()
	var b strings.Builder
	for x := range w {
		s, _, _ := t.screen.Get(x, y)
		if s == "" {
			s = " "
		}
		b.WriteString(s)
	}
	return b.String()
}

func (t *tui) find(r string) (x, y int, ok bool) {
	w, h := t.screen.Size()
	for y := range h {
		for x := range w {
			if s, _, _ := t.screen.Get(x, y); s == r {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func TestColumnOfMatchesResample(t *testing.T) {
	for _, tc := range []struct{ n, w int }{{3, 2}, {112, 47}, {10, 10}, {3, 8}, {1, 5}, {4096, 100}} {
		for i := range tc.n {
			src := make([]float32, tc.n)
			src[i] = 1
			dst := make([]float32, tc.w)
			trace.Resample(dst, src)
			if x := columnOf(i, tc.n, tc.w); dst[x] != 1 {
				t.Fatalf("n=%d w=%d: sample %d mapped to column %d, which does not contain it", tc.n, tc.w, i, x)
			}
		}
	}
}

func TestDrawTraceScalesToWidth(t *testing.T) {
	tt := newTestTUI(t, 80, 24)
	samples := make([]float32, 112)
	for i := range samples {
		samples[i] = -90
	}
	samples[100] = -20
	tt.sweep = &rfx.Sweep{Start: 2400 * rfx.MHz, Step: rfx.MHz, Samples: samples}
	tt.maxHold.Add(tt.sweep)
	tt.draw()

	plotW := 80 - leftWidth - 1
	x, _, ok := tt.find("V")
	if !ok {
		t.Fatal("no peak marker")
	}
	if want := leftWidth + columnOf(100, len(samples), plotW); x != want {
		t.Fatalf("peak marker at column %d, want %d", x, want)
	}
	if !strings.Contains(tt.row(24-1), "2511.000") {
		t.Fatalf("last frequency label missing: %q", tt.row(23))
	}
	// The axis spans the plot width and no further.
	axis := tt.row(24 - 2)
	if got := strings.Count(axis, "-"); got != plotW {
		t.Fatalf("axis has %d columns, want %d", got, plotW)
	}
}

func TestDrawChannelsMarksStrongest(t *testing.T) {
	tt := newTestTUI(t, 32+14*5+1, 24)
	tt.wifi = true
	samples := make([]float32, 100)
	for i := range samples {
		samples[i] = -90
	}
	// Channel 6 is centered on 2437 MHz.
	for i := 34; i <= 40; i++ {
		samples[i] = -30
	}
	tt.sweep = &rfx.Sweep{Start: 2400 * rfx.MHz, Step: rfx.MHz, Samples: samples}
	tt.draw()

	_, y, ok := tt.find("V")
	if !ok {
		t.Fatal("no peak marker in channel mode")
	}
	if !strings.Contains(tt.row(y-2), "ch 6") {
		t.Fatalf("marker label %q, want channel 6", tt.row(y-2))
	}
}

func TestDrawTooSmallDoesNotPanic(t *testing.T) {
	tt := newTestTUI(t, 20, 4)
	tt.sweep = &rfx.Sweep{Start: rfx.MHz, Step: rfx.KHz, Samples: []float32{-50}}
	tt.draw()
	tt.wifi = true
	tt.draw()
}
