package main

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"samuelks.com/rfx"
	"samuelks.com/rfx/trace"
)

// WaterfallWidget draws a trace.Waterfall, one column per sweep point. It
// is used only from the UI goroutine.
type WaterfallWidget struct {
	w           *trace.Waterfall
	start, step rfx.Frequency
	first       paint.ImageOp
	second      paint.ImageOp
	firstRows   int
	dirty       bool
}

func NewWaterfallWidget(height int) *WaterfallWidget {
	return &WaterfallWidget{w: trace.NewWaterfall(1, height, -120, 0), dirty: true}
}

// Update adds a sweep as the newest row. The waterfall is cleared when the
// sweep's frequency axis changes, and resized to its point count.
func (ww *WaterfallWidget) Update(s *rfx.Sweep) {
	if s.Start != ww.start || s.Step != ww.step {
		ww.start, ww.step = s.Start, s.Step
		ww.w.Clear()
	}
	_, h := ww.w.Size()
	ww.w.Resize(len(s.Samples), h)
	ww.w.Add(s.Samples)
	ww.dirty = true
}

// SetAmplitude sets the displayed level range, clearing the waterfall if it
// changed.
func (ww *WaterfallWidget) SetAmplitude(bottom, top int) {
	ww.w.SetRange(float64(bottom), float64(top))
	ww.dirty = true
}

func (ww *WaterfallWidget) Layout(gtx layout.Context) layout.Dimensions {
	width, height := ww.w.Size()
	if ww.dirty {
		// The image ops reference the ring buffer directly. Gio uploads
		// them during the frame, before the next sweep is added.
		first, second := ww.w.Segments()
		ww.first = paint.NewImageOp(first)
		ww.second = paint.NewImageOp(second)
		ww.firstRows = first.Rect.Dy()
		ww.dirty = false
	}

	size := gtx.Constraints.Max
	scale := f32.Pt(float32(size.X)/float32(width), float32(size.Y)/float32(height))
	defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, scale)).Push(gtx.Ops).Pop()

	ww.first.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	if ww.firstRows < height {
		off := op.Offset(image.Pt(0, ww.firstRows)).Push(gtx.Ops)
		ww.second.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		off.Pop()
	}

	return layout.Dimensions{Size: size}
}
