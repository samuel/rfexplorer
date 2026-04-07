package main

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"
)

type WifiSpectrumWidget struct {
	mu                     sync.Mutex
	min, max               float64
	values                 []float64
	maxValues              []float64
	maxHold                bool
	heatMapEnabled         bool
	heatMap                []int
	heatMapImg             *image.RGBA
	heatMapMax             int
	minFreqKHz, maxFreqKHz int
	theme                  *material.Theme
}

func NewWifiSpectrumWidget(minFreqKHz, maxFreqKHz int, theme *material.Theme) *WifiSpectrumWidget {
	sw := &WifiSpectrumWidget{
		min:            -120,
		max:            0,
		maxHold:        true,
		heatMapEnabled: true,
		minFreqKHz:     minFreqKHz,
		maxFreqKHz:     maxFreqKHz,
		theme:          theme,
	}
	return sw
}

func (sw *WifiSpectrumWidget) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	for i := range sw.maxValues {
		sw.maxValues[i] = sw.min
	}
	for i := range sw.heatMap {
		sw.heatMap[i] = 0
	}
	sw.heatMapMax = 0
}

func (sw *WifiSpectrumWidget) Update(values []float64) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if len(sw.values) != len(values) {
		sw.values = make([]float64, len(values))
		sw.maxValues = make([]float64, len(values))
		for i := range sw.maxValues {
			sw.maxValues[i] = sw.min
		}
		sw.heatMap = make([]int, len(values)*heatMapHeight)
		sw.heatMapMax = 0
	}
	copy(sw.values, values)
	for i, v := range values {
		if v > sw.maxValues[i] {
			sw.maxValues[i] = v
		}
		if sw.heatMapEnabled {
			a := (v - sw.min) / (sw.max - sw.min)
			if a < 0 {
				a = 0
			} else if a > 1 {
				a = 1
			}
			o := int(a*(heatMapHeight-1))*len(values) + i
			hm := sw.heatMap[o] + 1
			sw.heatMap[o] = hm
			if hm > sw.heatMapMax {
				sw.heatMapMax = hm
			}
		}
	}
}

func (sw *WifiSpectrumWidget) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Execute(op.InvalidateCmd{})

	// TODO: optimize by not rerendering images, creating new image ops, etc.. if values haven't changed

	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y

	if sw.heatMapEnabled {
		if sw.heatMapImg == nil || sw.heatMapImg.Bounds().Dx() != len(sw.values) || sw.heatMapImg.Bounds().Dy() != heatMapHeight {
			sw.heatMapImg = image.NewRGBA(image.Rect(0, 0, len(sw.values), heatMapHeight))
		}
		for y := 0; y < heatMapHeight; y++ {
			for x := 0; x < len(sw.values); x++ {
				var v float64
				if sw.heatMapMax > 0 {
					v = float64(sw.heatMap[((heatMapHeight-1)-y)*len(sw.values)+x]) / float64(sw.heatMapMax)
					if v > 1 {
						v = 1
					}
				}
				c := gradientColorForValue(v)
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

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.maxHold {
		var path clip.Path
		path.Begin(gtx.Ops)
		for i, v := range sw.maxValues {
			a := (v - sw.min) / (sw.max - sw.min)
			if a < 0 {
				a = 0
			} else if a > 1 {
				a = 1
			}
			y := (h - 1) - int((float64(h)-1)*a)
			x := i * (w - 1) / (len(sw.values) - 1)
			if i == 0 {
				path.MoveTo(f32.Pt(float32(x), float32(y)))
			} else {
				path.LineTo(f32.Pt(float32(x), float32(y)))
			}
		}
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xa0, G: 0, B: 0, A: 0xff},
			clip.Stroke{
				Path:  path.End(),
				Width: 2,
			}.Op())
	}

	var path clip.Path
	path.Begin(gtx.Ops)
	for i, v := range sw.values {
		y := (h - 1) - int((float64(h)-1)*(v-sw.min)/(sw.max-sw.min))
		x := i * (w - 1) / (len(sw.values) - 1)
		if i == 0 {
			path.MoveTo(f32.Pt(float32(x), float32(y)))
		} else {
			path.LineTo(f32.Pt(float32(x), float32(y)))
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x80, G: 0x80, B: 0xff, A: 0xff},
		clip.Stroke{
			Path:  path.End(),
			Width: 3,
		}.Op())

	for chanIx, freqRange := range wifi24ChannelRangesMHz {
		start := w * (freqRange[0]*1000 - sw.minFreqKHz) / (sw.maxFreqKHz - sw.minFreqKHz)
		end := w * (freqRange[1]*1000 - sw.minFreqKHz) / (sw.maxFreqKHz - sw.minFreqKHz)
		if start < 0 {
			start = 0
		}
		if end >= w {
			end = w - 1
		}
		y := h/3 + (chanIx%5)*40
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(f32.Pt(float32(start), float32(y)))
		path.LineTo(f32.Pt(float32(end), float32(y)))
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x50},
			clip.Stroke{
				Path:  path.End(),
				Width: 1,
			}.Op())

		trans := f32.Affine2D{}.Offset(f32.Pt(float32(start), float32(y-30)))
		aff := op.Affine(trans).Push(gtx.Ops)
		l := material.Overline(sw.theme, fmt.Sprintf("Ch %d", chanIx+1))
		l.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x50}
		l.Alignment = text.Middle
		gtx := gtx
		gtx.Constraints.Min.X = end - start
		gtx.Constraints.Max.X = end - start
		_ = l.Layout(gtx)
		aff.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
