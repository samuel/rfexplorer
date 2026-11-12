package main

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
)

type WaterfallWidget struct {
	w   *Waterfall
	m   *image.RGBA
	src paint.ImageOp
	row int
}

func NewWaterfallWidget(w *Waterfall) *WaterfallWidget {
	return &WaterfallWidget{w: w, row: -1}
}

func (ww *WaterfallWidget) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Execute(op.InvalidateCmd{})

	newRow := ww.w.Row()
	if newRow != ww.row {
		ww.m = ww.w.Image(ww.m)
		ww.row = newRow
		ww.src = paint.NewImageOp(ww.m)
	}
	imgBounds := ww.m.Bounds()

	trans := f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(gtx.Constraints.Max.X)/float32(imgBounds.Dx()), float32(gtx.Constraints.Max.Y)/float32(imgBounds.Dy())))
	defer op.Affine(trans).Push(gtx.Ops).Pop()

	ww.src.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
