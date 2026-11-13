package main

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"samuelks.com/rfx"
	"samuelks.com/rfx/bands"
	"samuelks.com/rfx/trace"
)

const chanRowDp unit.Dp = 36

// ChannelOverviewWidget displays per-channel signal activity for a BandSet.
// Each band occupies a horizontal row; channels are rendered as VU-meter bars
// positioned by frequency across the band set's span. Clicking a channel
// selects it (highlighting its frequency range in the spectrum) and shows its
// max received power. Clicking again deselects. It is used only from the UI
// goroutine.
type ChannelOverviewWidget struct {
	selectedBand int
	selectedChan int
	clicks       [][]gesture.Click
	min, max     float64
	bandSet      *bands.BandSet
	sweep        rfx.Sweep
	maxHold      trace.MaxHold
	spanLo       rfx.Frequency
	spanHi       rfx.Frequency
}

func NewChannelOverviewWidget() *ChannelOverviewWidget {
	return &ChannelOverviewWidget{
		min:          -120,
		max:          0,
		selectedBand: -1,
		selectedChan: -1,
	}
}

func (cw *ChannelOverviewWidget) SetBandSet(bs *bands.BandSet) {
	cw.bandSet = bs
	cw.maxHold.Reset()
	if bs == nil {
		cw.clicks = nil
		cw.spanLo, cw.spanHi = 0, 0
	} else {
		cw.spanLo, cw.spanHi = bs.Span()
		cw.clicks = make([][]gesture.Click, len(bs.Bands))
		for bi, band := range bs.Bands {
			cw.clicks[bi] = make([]gesture.Click, len(band.Channels))
		}
	}
	// Reset selection since band set changed.
	cw.selectedBand = -1
	cw.selectedChan = -1
}

// SetAmplitude updates the displayed amplitude range, resetting the
// per-channel max hold if it changed.
func (cw *ChannelOverviewWidget) SetAmplitude(bottom, top int) {
	if float64(bottom) == cw.min && float64(top) == cw.max {
		return
	}
	cw.min = float64(bottom)
	cw.max = float64(top)
	cw.maxHold.Reset()
}

func (cw *ChannelOverviewWidget) Update(s *rfx.Sweep) {
	cw.sweep.Time, cw.sweep.Start, cw.sweep.Step = s.Time, s.Start, s.Step
	cw.sweep.Samples = append(cw.sweep.Samples[:0], s.Samples...)
	cw.maxHold.Add(s)
}

// SelectedRange returns the frequency range of the selected channel, or
// ok=false if no channel is selected.
func (cw *ChannelOverviewWidget) SelectedRange() (lo, hi rfx.Frequency, ok bool) {
	if cw.selectedBand < 0 {
		return 0, 0, false
	}
	if cw.bandSet == nil || cw.selectedBand >= len(cw.bandSet.Bands) {
		return 0, 0, false
	}
	band := cw.bandSet.Bands[cw.selectedBand]
	if cw.selectedChan >= len(band.Channels) {
		return 0, 0, false
	}
	ch := band.Channels[cw.selectedChan]
	return ch.Low(), ch.High(), true
}

// channelLevels returns the peak level of every channel in the latest sweep
// and in the max-hold trace, using ampMin where a channel is outside the
// sweep.
func (cw *ChannelOverviewWidget) channelLevels(bs *bands.BandSet, ampMin float64) (current, held [][]float64) {
	maxTrace := cw.maxHold.Trace()
	level := func(s *rfx.Sweep, ch bands.Channel) float64 {
		if _, v, ok := trace.Peak(s, ch.Low(), ch.High()); ok {
			return float64(v)
		}
		return ampMin
	}
	current = make([][]float64, len(bs.Bands))
	held = make([][]float64, len(bs.Bands))
	for bi, band := range bs.Bands {
		current[bi] = make([]float64, len(band.Channels))
		held[bi] = make([]float64, len(band.Channels))
		for ci, ch := range band.Channels {
			current[bi][ci] = level(&cw.sweep, ch)
			held[bi][ci] = level(&maxTrace, ch)
		}
	}
	return current, held
}

func (cw *ChannelOverviewWidget) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bs := cw.bandSet
	clicks := cw.clicks
	spanLo, spanHi := cw.spanLo, cw.spanHi
	ampMin := cw.min
	ampMax := cw.max
	var channelCurrent, chanMaxPower [][]float64
	if bs != nil {
		channelCurrent, chanMaxPower = cw.channelLevels(bs, ampMin)
	}

	if bs == nil || len(bs.Bands) == 0 {
		return layout.Dimensions{}
	}

	numBands := len(bs.Bands)
	rowH := max(gtx.Dp(chanRowDp), 20)
	totalH := numBands * rowH
	w := gtx.Constraints.Max.X
	if w == 0 {
		w = gtx.Constraints.Min.X
	}

	// Clip all drawing to the widget bounds.
	defer clip.Rect(image.Rectangle{Max: image.Pt(w, totalH)}).Push(gtx.Ops).Pop()

	// Widget background.
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff},
		clip.Rect(image.Rectangle{Max: image.Pt(w, totalH)}).Op())

	span := spanHi - spanLo
	if span <= 0 {
		return layout.Dimensions{Size: image.Pt(w, totalH)}
	}

	// Process click events for all channels.
	for bi, band := range bs.Bands {
		if bi >= len(clicks) {
			break
		}
		for ci := range band.Channels {
			if ci >= len(clicks[bi]) {
				break
			}
			for {
				e, ok := clicks[bi][ci].Update(gtx.Source)
				if !ok {
					break
				}
				if e.Kind == gesture.KindClick {
					if cw.selectedBand == bi && cw.selectedChan == ci {
						cw.selectedBand = -1
						cw.selectedChan = -1
					} else {
						cw.selectedBand = bi
						cw.selectedChan = ci
					}
				}
			}
		}
	}

	selectedBand := cw.selectedBand
	selectedChan := cw.selectedChan

	// Render.
	for bi, band := range bs.Bands {
		if bi >= len(clicks) || bi >= len(chanMaxPower) {
			break
		}
		bandY := bi * rowH
		labelCol := bandLabelColor(bi)

		// Alternating band row background (slightly lighter than widget bg).
		bg := color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xff}
		if bi%2 == 1 {
			bg = color.NRGBA{R: 0x2c, G: 0x2c, B: 0x2c, A: 0xff}
		}
		paint.FillShape(gtx.Ops, bg,
			clip.Rect(image.Rectangle{
				Min: image.Pt(0, bandY),
				Max: image.Pt(w, bandY+rowH),
			}).Op())

		// Row separator line.
		if bi > 0 {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff},
				clip.Rect(image.Rectangle{
					Min: image.Pt(0, bandY),
					Max: image.Pt(w, bandY+1),
				}).Op())
		}

		for ci, ch := range band.Channels {
			if ci >= len(clicks[bi]) {
				break
			}

			leftX := int((ch.Low() - spanLo) * rfx.Frequency(w) / span)
			rightX := int((ch.High() - spanLo) * rfx.Frequency(w) / span)
			if rightX <= leftX {
				rightX = leftX + 1
			}
			leftX = max(leftX, 0)
			rightX = min(rightX, w)
			if leftX >= w || rightX <= 0 {
				continue
			}

			isSelected := selectedBand == bi && selectedChan == ci

			// Cell border.
			borderCol := bandLabelColor(bi)
			borderCol.A = 0x60
			paint.FillShape(gtx.Ops, borderCol,
				clip.Rect(image.Rectangle{
					Min: image.Pt(leftX, bandY+1),
					Max: image.Pt(leftX+1, bandY+rowH-1),
				}).Op())

			// Channel background.
			chBg := bg
			if isSelected {
				chBg = color.NRGBA{R: 0x33, G: 0x33, B: 0x66, A: 0xff}
			}
			innerRect := image.Rectangle{
				Min: image.Pt(leftX+1, bandY+1),
				Max: image.Pt(rightX, bandY+rowH-1),
			}
			paint.FillShape(gtx.Ops, chBg, clip.Rect(innerRect).Op())

			// VU-meter fill (current sweep max), from bottom up.
			cur := channelCurrent[bi][ci]
			fillA := (cur - ampMin) / (ampMax - ampMin)
			if fillA < 0 {
				fillA = 0
			} else if fillA > 1 {
				fillA = 1
			}
			fillH := int(fillA * float64(rowH-2))
			if fillH > 0 {
				vuCol := bandFillColor(bi)
				vuCol.A = 0xc0
				paint.FillShape(gtx.Ops, vuCol,
					clip.Rect(image.Rectangle{
						Min: image.Pt(leftX+1, bandY+rowH-1-fillH),
						Max: image.Pt(rightX-1, bandY+rowH-1),
					}).Op())
			}

			// Max-hold tick line.
			if bi < len(chanMaxPower) && ci < len(chanMaxPower[bi]) {
				maxPow := chanMaxPower[bi][ci]
				maxA := (maxPow - ampMin) / (ampMax - ampMin)
				if maxA < 0 {
					maxA = 0
				} else if maxA > 1 {
					maxA = 1
				}
				tickY := bandY + rowH - 1 - int(maxA*float64(rowH-2))
				paint.FillShape(gtx.Ops, labelCol,
					clip.Rect(image.Rectangle{
						Min: image.Pt(leftX+1, tickY),
						Max: image.Pt(rightX-1, tickY+2),
					}).Op())
			}

			// Channel name label.
			if rightX-leftX >= 10 {
				chLabelStack := op.Offset(image.Pt(leftX+2, bandY+2)).Push(gtx.Ops)
				gtxCh := gtx
				gtxCh.Constraints.Min.X = 0
				gtxCh.Constraints.Max.X = rightX - leftX - 4
				lch := material.Caption(th, ch.Name)
				lch.Color = labelCol
				lch.Layout(gtxCh)
				chLabelStack.Pop()
			}

			// Max power label for selected channel.
			if isSelected && bi < len(chanMaxPower) && ci < len(chanMaxPower[bi]) {
				maxPow := chanMaxPower[bi][ci]
				labelY := bandY + rowH - gtx.Sp(12) - 2
				lStack := op.Offset(image.Pt(leftX+2, labelY)).Push(gtx.Ops)
				gtxL := gtx
				gtxL.Constraints.Min.X = 0
				gtxL.Constraints.Max.X = rightX - leftX - 4
				lm := material.Caption(th, fmt.Sprintf("%.1f", maxPow))
				lm.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				lm.Layout(gtxL)
				lStack.Pop()
			}

			// Register click target.
			chArea := clip.Rect(image.Rectangle{
				Min: image.Pt(leftX, bandY),
				Max: image.Pt(rightX, bandY+rowH),
			}).Push(gtx.Ops)
			clicks[bi][ci].Add(gtx.Ops)
			chArea.Pop()
		}

		// Band name label, right-aligned in a small overlay at the left edge.
		nameStack := op.Offset(image.Pt(2, bandY+2)).Push(gtx.Ops)
		gtxName := gtx
		gtxName.Constraints.Min.X = 0
		gtxName.Constraints.Max.X = 56
		ln := material.Caption(th, band.Name)
		ln.Color = color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
		ln.Layout(gtxName)
		nameStack.Pop()
	}

	return layout.Dimensions{Size: image.Pt(w, totalH)}
}
