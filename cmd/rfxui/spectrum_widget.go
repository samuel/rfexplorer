package main

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	"samuelks.com/rfx"
	"samuelks.com/rfx/trace"
)

const heatMapHeight = 256

// bandBaseColors is the color palette used to distinguish bands in an overlay.
// Each entry is an RGB triple; alpha is applied separately for fills vs labels.
var bandBaseColors = [][3]uint8{
	{0xff, 0x80, 0x00}, // orange
	{0x00, 0xcc, 0x44}, // green
	{0xcc, 0x44, 0xff}, // purple
	{0x00, 0xcc, 0xcc}, // cyan
	{0xff, 0xff, 0x00}, // yellow
	{0xff, 0x44, 0x44}, // red
	{0x44, 0x88, 0xff}, // blue
	{0x88, 0xff, 0x44}, // lime
}

func bandFillColor(i int) color.NRGBA {
	c := bandBaseColors[i%len(bandBaseColors)]
	return color.NRGBA{R: c[0], G: c[1], B: c[2], A: 0x28}
}

func bandLabelColor(i int) color.NRGBA {
	c := bandBaseColors[i%len(bandBaseColors)]
	return color.NRGBA{R: c[0], G: c[1], B: c[2], A: 0xff}
}

type SpectrumWidget struct {
	// UI-goroutine-only fields:
	hoverX   int
	gradient trace.Gradient

	mu             sync.Mutex
	min, max       float64
	sweep          rfx.Sweep
	maxHold        trace.MaxHold
	maxHoldEnabled bool
	heatMapEnabled bool
	heatMap        []int
	heatMapImg     *image.RGBA
	heatMapMax     int
	highlightLo    rfx.Frequency
	highlightHi    rfx.Frequency
}

func NewSpectrumWidget() *SpectrumWidget {
	sw := &SpectrumWidget{
		min:            -120,
		max:            0,
		maxHoldEnabled: true,
		heatMapEnabled: true,
		hoverX:         -1,
		gradient:       trace.DefaultGradient(),
	}
	return sw
}

func (sw *SpectrumWidget) resetLocked() {
	sw.maxHold.Reset()
	for i := range sw.heatMap {
		sw.heatMap[i] = 0
	}
	sw.heatMapMax = 0
}

func (sw *SpectrumWidget) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.resetLocked()
}

// SetAmplitude updates the displayed amplitude range and resets accumulated data.
func (sw *SpectrumWidget) SetAmplitude(bottom, top int) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.min = float64(bottom)
	sw.max = float64(top)
	sw.resetLocked()
}

func (sw *SpectrumWidget) Update(s *rfx.Sweep) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if len(sw.sweep.Samples) != len(s.Samples) {
		sw.heatMap = make([]int, len(s.Samples)*heatMapHeight)
		sw.heatMapMax = 0
	}
	sw.sweep.Time, sw.sweep.Start, sw.sweep.Step = s.Time, s.Start, s.Step
	sw.sweep.Samples = append(sw.sweep.Samples[:0], s.Samples...)
	sw.maxHold.Add(s)
	n := len(s.Samples)
	for i, v := range s.Samples {
		if sw.heatMapEnabled {
			a := (float64(v) - sw.min) / (sw.max - sw.min)
			if a < 0 {
				a = 0
			} else if a > 1 {
				a = 1
			}
			o := int(a*(heatMapHeight-1))*n + i
			hm := sw.heatMap[o] + 1
			sw.heatMap[o] = hm
			if hm > sw.heatMapMax {
				sw.heatMapMax = hm
			}
		}
	}
}

// SetHighlight marks a frequency range to highlight in the spectrum.
func (sw *SpectrumWidget) SetHighlight(lo, hi rfx.Frequency) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.highlightLo = lo
	sw.highlightHi = hi
}

// ClearHighlight removes any active frequency highlight.
func (sw *SpectrumWidget) ClearHighlight() {
	sw.SetHighlight(0, 0)
}

// sampleX returns the pixel column of sample i of n across width w.
func sampleX(i, n, w int) int {
	if n <= 1 {
		return 0
	}
	return i * (w - 1) / (n - 1)
}

// levelY returns the pixel row of level v across height h for the range
// [lo, hi], clamped to the widget.
func levelY(v, lo, hi float64, h int) int {
	a := 0.0
	if hi != lo {
		a = min(max((v-lo)/(hi-lo), 0), 1)
	}
	return (h - 1) - int(float64(h-1)*a)
}

const (
	markerTriHalf = 8
	markerTriH    = 10
	markerGap     = 3
	markerLabelW  = 220
)

// drawMarker draws a downward-pointing triangle at (pixelX, tipY) and a label
// to the right (or left if near the edge). widgetW is the full widget width in
// pixels.
func drawMarker(gtx layout.Context, th *material.Theme, pixelX, tipY, widgetW int, label string, col color.NRGBA) {
	triTip := tipY - markerGap
	triBase := triTip - markerTriH

	var tpath clip.Path
	tpath.Begin(gtx.Ops)
	tpath.MoveTo(f32.Pt(float32(pixelX), float32(triTip)))
	tpath.LineTo(f32.Pt(float32(pixelX-markerTriHalf), float32(triBase)))
	tpath.LineTo(f32.Pt(float32(pixelX+markerTriHalf), float32(triBase)))
	tpath.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: tpath.End()}.Op())

	labelX := pixelX + markerTriHalf + 4
	if labelX+markerLabelW > widgetW {
		labelX = pixelX - markerTriHalf - 4 - markerLabelW
	}
	labelY := max(triBase-2-50, 0)

	bgRect := image.Rectangle{
		Min: image.Pt(labelX-2, labelY-2),
		Max: image.Pt(labelX+markerLabelW+2, labelY+34),
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0xb0}, clip.Rect(bgRect).Op())

	stack := op.Offset(image.Pt(labelX, labelY)).Push(gtx.Ops)
	gtxLabel := gtx
	gtxLabel.Constraints.Min.X = 0
	gtxLabel.Constraints.Max.X = markerLabelW
	l := material.Body2(th, label)
	l.Color = col
	l.Layout(gtxLabel)
	stack.Pop()
}

func (sw *SpectrumWidget) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	gtx.Execute(op.InvalidateCmd{})

	// TODO: optimize by not rerendering images, creating new image ops, etc.. if values haven't changed

	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y

	// Register pointer input over the full widget area.
	hitArea := clip.Rect(image.Rectangle{Max: image.Pt(w, h)}).Push(gtx.Ops)
	event.Op(gtx.Ops, sw)
	hitArea.Pop()

	// Process pointer events to track hover position.
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: sw,
			Kinds:  pointer.Move | pointer.Enter | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Move, pointer.Enter:
				sw.hoverX = int(e.Position.X)
			case pointer.Leave, pointer.Cancel:
				sw.hoverX = -1
			}
		}
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()
	n := len(sw.sweep.Samples)

	if sw.heatMapEnabled && n > 0 {
		if sw.heatMapImg == nil || sw.heatMapImg.Bounds().Dx() != n || sw.heatMapImg.Bounds().Dy() != heatMapHeight {
			sw.heatMapImg = image.NewRGBA(image.Rect(0, 0, n, heatMapHeight))
		}
		for y := range heatMapHeight {
			for x := range n {
				var v float64
				if sw.heatMapMax > 0 {
					v = float64(sw.heatMap[((heatMapHeight-1)-y)*n+x]) / float64(sw.heatMapMax)
				}
				c := sw.gradient.At(v)
				sw.heatMapImg.Pix[y*sw.heatMapImg.Stride+x*4] = c.R
				sw.heatMapImg.Pix[y*sw.heatMapImg.Stride+x*4+1] = c.G
				sw.heatMapImg.Pix[y*sw.heatMapImg.Stride+x*4+2] = c.B
				sw.heatMapImg.Pix[y*sw.heatMapImg.Stride+x*4+3] = c.A
			}
		}
		imgBounds := sw.heatMapImg.Bounds()
		trans := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(w)/float32(imgBounds.Dx()), float32(h)/float32(imgBounds.Dy())))
		aff := op.Affine(trans).Push(gtx.Ops)
		paint.NewImageOp(sw.heatMapImg).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		aff.Pop()
	}

	for i := 1; i < 10; i++ {
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(f32.Pt(float32(i*(w-1)/10), 0))
		path.LineTo(f32.Pt(float32(i*(w-1)/10), float32(h-1)))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0x50, G: 0x50, B: 0x50, A: 0x50},
			clip.Stroke{
				Path:  path.End(),
				Width: 1,
			}.Op())
	}
	for i := 1; i <= 5; i++ {
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(f32.Pt(0, float32(i*(h-1)/5)))
		path.LineTo(f32.Pt(float32(w-1), float32(i*(h-1)/5)))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0x50, G: 0x50, B: 0x50, A: 0x50},
			clip.Stroke{
				Path:  path.End(),
				Width: 1,
			}.Op())
	}

	// Amplitude Y-axis labels at the top and each horizontal grid line.
	labelColor := color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}
	labelH := gtx.Sp(11)
	for i := range 6 {
		dbm := sw.max - float64(i)*(sw.max-sw.min)/5.0
		gridY := i * (h - 1) / 5
		labelY := gridY + 2
		if labelY+labelH > h {
			labelY = h - labelH - 2
		}
		stack := op.Offset(image.Pt(4, labelY)).Push(gtx.Ops)
		gtxL := gtx
		gtxL.Constraints.Min.X = 0
		gtxL.Constraints.Max.X = 80
		l := material.Caption(th, fmt.Sprintf("%.0f", dbm))
		l.Color = labelColor
		l.Layout(gtxL)
		stack.Pop()
	}

	// Highlight rect: draw a translucent overlay for the selected channel range.
	if sw.highlightLo != sw.highlightHi && sw.sweep.Step > 0 && n > 1 {
		span := sw.sweep.Step * rfx.Frequency(n-1)
		freqToPixel := func(f rfx.Frequency) int {
			return int((f - sw.sweep.Start) * rfx.Frequency(w-1) / span)
		}
		leftX := max(freqToPixel(sw.highlightLo), 0)
		rightX := min(freqToPixel(sw.highlightHi), w)
		if leftX < rightX {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x28},
				clip.Rect(image.Rectangle{
					Min: image.Pt(leftX, 0),
					Max: image.Pt(rightX, h),
				}).Op())
		}
	}

	maxTrace := sw.maxHold.Trace()
	strokeTrace := func(samples []float32, width float32, col color.NRGBA) {
		if len(samples) == 0 {
			return
		}
		var path clip.Path
		path.Begin(gtx.Ops)
		for i, v := range samples {
			pt := f32.Pt(float32(sampleX(i, len(samples), w)), float32(levelY(float64(v), sw.min, sw.max, h)))
			if i == 0 {
				path.MoveTo(pt)
			} else {
				path.LineTo(pt)
			}
		}
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: path.End(), Width: width}.Op())
	}
	if sw.maxHoldEnabled {
		strokeTrace(maxTrace.Samples, 2, color.NRGBA{R: 0xa0, G: 0, B: 0, A: 0xff})
	}
	strokeTrace(sw.sweep.Samples, 4, color.NRGBA{R: 0x80, G: 0x80, B: 0xff, A: 0xff})

	// Peak marker at the highest max-hold level.
	if peakIdx, peakVal := maxTrace.Peak(); peakIdx >= 0 && maxTrace.Step > 0 {
		peakX := sampleX(peakIdx, len(maxTrace.Samples), w)
		peakY := levelY(float64(peakVal), sw.min, sw.max, h)
		label := fmt.Sprintf("%.3f MHz\n%.1f dBm", maxTrace.FrequencyAt(peakIdx).MHz(), peakVal)
		drawMarker(gtx, th, peakX, peakY, w, label, color.NRGBA{R: 0xff, G: 0xe0, B: 0x00, A: 0xff})
	}

	// Hover marker: show frequency and power at the cursor position.
	if sw.hoverX >= 0 && n > 0 && sw.sweep.Step > 0 {
		hoverIdx := 0
		if w > 1 {
			hoverIdx = min(max(sw.hoverX*(n-1)/(w-1), 0), n-1)
		}
		hoverVal := sw.sweep.Samples[hoverIdx]
		hoverPixX := sampleX(hoverIdx, n, w)
		hoverPixY := levelY(float64(hoverVal), sw.min, sw.max, h)
		label := fmt.Sprintf("%.3f MHz\n%.1f dBm", sw.sweep.FrequencyAt(hoverIdx).MHz(), hoverVal)
		drawMarker(gtx, th, hoverPixX, hoverPixY, w, label, color.NRGBA{R: 0x00, G: 0xff, B: 0xff, A: 0xff})
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
