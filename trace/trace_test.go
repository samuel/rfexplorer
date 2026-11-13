package trace

import (
	"image/color"
	"math"
	"slices"
	"testing"

	"samuelks.com/rfx"
)

func sweep(start, step rfx.Frequency, samples ...float32) *rfx.Sweep {
	return &rfx.Sweep{Start: start, Step: step, Samples: samples}
}

func TestMaxHold(t *testing.T) {
	var m MaxHold
	m.Add(sweep(100, 10, -50, -60, -70))
	m.Add(sweep(100, 10, -55, -40, -80))
	if got, want := m.Trace().Samples, []float32{-50, -40, -70}; !slices.Equal(got, want) {
		t.Fatalf("held %v, want %v", got, want)
	}

	// A new frequency axis restarts the hold.
	m.Add(sweep(200, 10, -90, -90, -90))
	tr := m.Trace()
	if tr.Start != 200 || !slices.Equal(tr.Samples, []float32{-90, -90, -90}) {
		t.Fatalf("after axis change got start %v samples %v", tr.Start, tr.Samples)
	}
	m.Add(sweep(200, 10, -90, -90))
	if got := m.Trace().Samples; len(got) != 2 {
		t.Fatalf("after length change got %v", got)
	}

	m.Reset()
	if got := m.Trace().Samples; len(got) != 0 {
		t.Fatalf("after reset got %v", got)
	}
	m.Add(sweep(200, 10, -10, -20))
	if got, want := m.Trace().Samples, []float32{-10, -20}; !slices.Equal(got, want) {
		t.Fatalf("after reset and add got %v, want %v", got, want)
	}
}

func TestMaxHoldDoesNotAliasInput(t *testing.T) {
	var m MaxHold
	s := sweep(0, 1, -50)
	m.Add(s)
	s.Samples[0] = 0
	if got := m.Trace().Samples[0]; got != -50 {
		t.Fatalf("hold changed with input: %v", got)
	}
}

func approx(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestAverageLinearPower(t *testing.T) {
	a := NewAverage(2)
	a.Add(sweep(0, 1, -30, -50))
	a.Add(sweep(0, 1, -40, -50))
	// mean(1 µW, 0.1 µW) = 0.55 µW = -32.596 dBm, not the -35 dBm a dBm mean gives.
	got := a.Trace().Samples
	if !approx(float64(got[0]), -32.596, 0.001) || !approx(float64(got[1]), -50, 0.001) {
		t.Fatalf("average %v", got)
	}
	if a.Count() != 2 {
		t.Fatalf("count %d", a.Count())
	}

	// The oldest sweep drops out once N are held.
	a.Add(sweep(0, 1, -40, -50))
	if got := a.Trace().Samples; !approx(float64(got[0]), -40, 0.001) {
		t.Fatalf("sliding average %v", got)
	}

	a.Add(sweep(5, 1, -20, -20))
	if a.Count() != 1 || a.Trace().Start != 5 {
		t.Fatalf("axis change: count %d start %v", a.Count(), a.Trace().Start)
	}

	a.Reset()
	if a.Count() != 0 || len(a.Trace().Samples) != 0 {
		t.Fatal("reset left data")
	}
}

func TestPeak(t *testing.T) {
	s := sweep(1000, 100, -90, -50, -80, -40, -95)
	tests := []struct {
		name    string
		lo, hi  rfx.Frequency
		wantI   int
		wantDBm float32
		wantOK  bool
	}{
		{"whole sweep", 1000, 1400, 3, -40, true},
		{"inner range", 1050, 1250, 1, -50, true},
		{"edges inclusive", 1100, 1100, 1, -50, true},
		{"clamped", 0, 1150, 1, -50, true},
		{"narrow uses nearest to middle", 1130, 1190, 2, -80, true},
		{"below sweep", 0, 900, 0, 0, false},
		{"above sweep", 1500, 2000, 0, 0, false},
		{"inverted", 1300, 1100, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, dBm, ok := Peak(s, tt.lo, tt.hi)
			if i != tt.wantI || dBm != tt.wantDBm || ok != tt.wantOK {
				t.Fatalf("Peak = %d, %v, %v; want %d, %v, %v", i, dBm, ok, tt.wantI, tt.wantDBm, tt.wantOK)
			}
		})
	}
	if _, _, ok := Peak(&rfx.Sweep{Step: 1}, 0, 10); ok {
		t.Fatal("empty sweep reported a peak")
	}
}

func TestChannelPower(t *testing.T) {
	s := sweep(0, 1*rfx.MHz, -30, -30, -40, -40, -100)

	// Rect: mean of 1, 1, 0.1 µW = 0.7 µW.
	got, ok := ChannelPower(s, 1*rfx.MHz, 2*rfx.MHz, WindowRect)
	if !ok || !approx(got, rfx.MilliwattsToDBm(0.0007), 1e-9) {
		t.Fatalf("rect = %v, %v", got, ok)
	}

	// Blackman zeroes the edge samples, leaving only the center one.
	got, ok = ChannelPower(s, 1*rfx.MHz, 2*rfx.MHz, WindowBlackman)
	if !ok || !approx(got, -30, 1e-9) {
		t.Fatalf("blackman = %v, %v", got, ok)
	}

	// A channel with only edge samples falls back to an unweighted mean.
	got, ok = ChannelPower(s, 2500*rfx.KHz, 1*rfx.MHz, WindowBlackman)
	if !ok || !approx(got, -40, 1e-9) {
		t.Fatalf("edge-only blackman = %v, %v", got, ok)
	}

	// Narrower than the step: nearest sample to the center.
	got, ok = ChannelPower(s, 3100*rfx.KHz, 100*rfx.KHz, WindowRect)
	if !ok || !approx(got, -40, 1e-9) {
		t.Fatalf("narrow = %v, %v", got, ok)
	}

	if _, ok := ChannelPower(s, 100*rfx.MHz, 1*rfx.MHz, WindowRect); ok {
		t.Fatal("channel outside the sweep reported power")
	}
}

func TestResample(t *testing.T) {
	src := []float32{1, 5, 2, 3, 9, 4}
	dst := make([]float32, 3)
	Resample(dst, src)
	if want := []float32{5, 3, 9}; !slices.Equal(dst, want) {
		t.Fatalf("downsample %v, want %v", dst, want)
	}

	dst = make([]float32, 4)
	Resample(dst, []float32{1, 2})
	if want := []float32{1, 1, 2, 2}; !slices.Equal(dst, want) {
		t.Fatalf("upsample %v, want %v", dst, want)
	}

	dst = make([]float32, 5)
	Resample(dst, src[:5])
	if !slices.Equal(dst, src[:5]) {
		t.Fatalf("same size %v", dst)
	}

	dst = []float32{1, 2}
	Resample(dst, nil)
	if !slices.Equal(dst, []float32{0, 0}) {
		t.Fatalf("empty source %v", dst)
	}
}

func TestGradient(t *testing.T) {
	g := Gradient{{0, 0, 0, 255}, {100, 0, 0, 255}, {100, 200, 0, 255}}
	tests := []struct {
		v    float64
		want color.RGBA
	}{
		{-1, g[0]},
		{0, g[0]},
		{0.25, color.RGBA{50, 0, 0, 255}},
		{0.5, g[1]},
		{0.75, color.RGBA{100, 100, 0, 255}},
		{0.999, color.RGBA{100, 200, 0, 255}},
		{1, g[2]},
		{2, g[2]},
		{math.NaN(), g[0]},
	}
	for _, tt := range tests {
		if got := g.At(tt.v); got != tt.want {
			t.Errorf("At(%v) = %v, want %v", tt.v, got, tt.want)
		}
	}
	if got := (Gradient{}).At(0.5); got != (color.RGBA{}) {
		t.Errorf("empty gradient = %v", got)
	}

	// The top of the range reaches the last stop instead of saturating early.
	d := DefaultGradient()
	if got := d.At(0.95); got == d[len(d)-1] {
		t.Errorf("default gradient saturated at 0.95")
	}
}

func TestWaterfall(t *testing.T) {
	w := NewWaterfall(2, 3, -100, 0)
	g := DefaultGradient()
	pixel := func(x, y int) color.RGBA {
		first, second := w.Segments()
		if y < first.Rect.Dy() {
			return first.RGBAAt(x, first.Rect.Min.Y+y)
		}
		return second.RGBAAt(x, second.Rect.Min.Y+y-first.Rect.Dy())
	}

	w.Add([]float32{-100, 0})
	w.Add([]float32{0, -100, -100, -100}) // resampled by max to {0, -100}
	if pixel(0, 0) != g.At(1) || pixel(1, 0) != g.At(0) {
		t.Fatalf("newest row = %v %v", pixel(0, 0), pixel(1, 0))
	}
	if pixel(0, 1) != g.At(0) || pixel(1, 1) != g.At(1) {
		t.Fatalf("second row = %v %v", pixel(0, 1), pixel(1, 1))
	}
	if pixel(0, 2) != (color.RGBA{}) {
		t.Fatalf("unwritten row = %v", pixel(0, 2))
	}

	// Wrap around: the oldest row is overwritten.
	w.Add([]float32{-50, -50})
	w.Add([]float32{-100, -100})
	if pixel(0, 0) != g.At(0) || pixel(0, 1) != g.At(0.5) || pixel(0, 2) != g.At(1) {
		t.Fatalf("after wrap = %v %v %v", pixel(0, 0), pixel(0, 1), pixel(0, 2))
	}
	first, second := w.Segments()
	if first.Rect.Dy()+second.Rect.Dy() != 3 {
		t.Fatalf("segments cover %d rows", first.Rect.Dy()+second.Rect.Dy())
	}

	w.SetRange(-100, 0)
	if pixel(0, 0) == (color.RGBA{}) {
		t.Fatal("unchanged range cleared the image")
	}
	w.SetRange(-120, 0)
	if pixel(0, 0) != (color.RGBA{}) || pixel(0, 2) != (color.RGBA{}) {
		t.Fatal("range change did not clear the image")
	}

	w.Resize(4, 3)
	if width, height := w.Size(); width != 4 || height != 3 {
		t.Fatalf("size %dx%d", width, height)
	}
}
