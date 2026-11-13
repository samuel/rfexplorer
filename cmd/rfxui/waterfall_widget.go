package main

import (
	"image"
	"image/draw"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"samuelks.com/rfx"
	"samuelks.com/rfx/trace"
)

// WaterfallWidget draws a trace.Waterfall, one column per sweep point.
type WaterfallWidget struct {
	// UI-goroutine-only fields:
	img   *image.RGBA
	src   paint.ImageOp
	drawn int

	mu      sync.Mutex
	w       *trace.Waterfall
	version int
}

func NewWaterfallWidget(height int) *WaterfallWidget {
	return &WaterfallWidget{w: trace.NewWaterfall(1, height, -120, 0), drawn: -1}
}

// Update adds a sweep as the newest row, resizing the waterfall to the
// sweep's point count.
func (ww *WaterfallWidget) Update(s *rfx.Sweep) {
	ww.mu.Lock()
	defer ww.mu.Unlock()
	_, h := ww.w.Size()
	ww.w.Resize(len(s.Samples), h)
	ww.w.Add(s.Samples)
	ww.version++
}

// SetAmplitude sets the displayed level range, clearing the waterfall if it
// changed.
func (ww *WaterfallWidget) SetAmplitude(bottom, top int) {
	ww.mu.Lock()
	defer ww.mu.Unlock()
	ww.w.SetRange(float64(bottom), float64(top))
	ww.version++
}

// snapshot copies the waterfall in display order into ww.img.
func (ww *WaterfallWidget) snapshot() {
	ww.mu.Lock()
	defer ww.mu.Unlock()
	if ww.version == ww.drawn && ww.img != nil {
		return
	}
	ww.drawn = ww.version
	w, h := ww.w.Size()
	if ww.img == nil || ww.img.Rect.Dx() != w || ww.img.Rect.Dy() != h {
		ww.img = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	first, second := ww.w.Segments()
	draw.Draw(ww.img, ww.img.Rect, first, first.Rect.Min, draw.Src)
	draw.Draw(ww.img, ww.img.Rect.Add(image.Pt(0, first.Rect.Dy())), second, second.Rect.Min, draw.Src)
	ww.src = paint.NewImageOp(ww.img)
}

func (ww *WaterfallWidget) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Execute(op.InvalidateCmd{})

	ww.snapshot()
	b := ww.img.Bounds()
	trans := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(gtx.Constraints.Max.X)/float32(b.Dx()), float32(gtx.Constraints.Max.Y)/float32(b.Dy())))
	defer op.Affine(trans).Push(gtx.Ops).Pop()

	ww.src.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
