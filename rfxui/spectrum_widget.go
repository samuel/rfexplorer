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
	"gioui.org/widget/material"
)

const heatMapHeight = 256

type SpectrumWidget struct {
	mu             sync.Mutex
	min, max       float64
	values         []float64
	maxValues      []float64
	maxHold        bool
	heatMapEnabled bool
	heatMap        []int
	heatMapImg     *image.RGBA
	heatMapMax     int
	startFreqKHz   int
	freqStepHz     int
}

func NewSpectrumWidget() *SpectrumWidget {
	sw := &SpectrumWidget{
		min:            -120,
		max:            0,
		maxHold:        true,
		heatMapEnabled: true,
	}
	return sw
}

func (sw *SpectrumWidget) Reset() {
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

func (sw *SpectrumWidget) Update(values []float64) {
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

func (sw *SpectrumWidget) SetFreqConfig(startFreqKHz, freqStepHz int) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.startFreqKHz = startFreqKHz
	sw.freqStepHz = freqStepHz
}

func (sw *SpectrumWidget) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
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
			Width: 4,
		}.Op())

	// Peak marker: find the highest value in maxValues.
	if len(sw.maxValues) > 1 && sw.freqStepHz > 0 {
		peakIdx := 0
		peakVal := sw.maxValues[0]
		for i, v := range sw.maxValues[1:] {
			if v > peakVal {
				peakVal = v
				peakIdx = i + 1
			}
		}

		peakA := (peakVal - sw.min) / (sw.max - sw.min)
		if peakA < 0 {
			peakA = 0
		} else if peakA > 1 {
			peakA = 1
		}
		peakX := peakIdx * (w - 1) / (len(sw.maxValues) - 1)
		peakY := (h - 1) - int(float64(h-1)*peakA)

		// Downward-pointing triangle sitting just above the peak.
		const (
			triHalf = 8
			triH    = 10
			gap     = 3
		)
		triTip := peakY - gap
		triBase := triTip - triH

		var tpath clip.Path
		tpath.Begin(gtx.Ops)
		tpath.MoveTo(f32.Pt(float32(peakX), float32(triTip)))
		tpath.LineTo(f32.Pt(float32(peakX-triHalf), float32(triBase)))
		tpath.LineTo(f32.Pt(float32(peakX+triHalf), float32(triBase)))
		tpath.Close()
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xe0, B: 0x00, A: 0xff},
			clip.Outline{Path: tpath.End()}.Op())

		// Label: frequency and power.
		freqMHz := (float64(sw.startFreqKHz) + float64(peakIdx)*float64(sw.freqStepHz)/1000.0) / 1000.0
		label := fmt.Sprintf("%.3f MHz\n%.1f dBm", freqMHz, peakVal)

		const labelW = 220
		labelX := peakX + triHalf + 4
		if labelX+labelW > w {
			labelX = peakX - triHalf - 4 - labelW
		}
		labelY := triBase - 2 - 50
		if labelY < 0 {
			labelY = 0
		}

		// Dark background behind the label for readability.
		bgRect := image.Rectangle{
			Min: image.Pt(labelX-2, labelY-2),
			Max: image.Pt(labelX+labelW+2, labelY+34),
		}
		paint.FillShape(gtx.Ops, color.NRGBA{A: 0xb0},
			clip.Rect(bgRect).Op())

		stack := op.Offset(image.Pt(labelX, labelY)).Push(gtx.Ops)
		gtxLabel := gtx
		gtxLabel.Constraints.Min.X = 0
		gtxLabel.Constraints.Max.X = labelW
		l := material.Body2(th, label)
		l.Color = color.NRGBA{R: 0xff, G: 0xe0, B: 0x00, A: 0xff}
		l.Layout(gtxLabel)
		stack.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
