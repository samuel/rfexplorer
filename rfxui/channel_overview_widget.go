package main

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/samuel/rfexplorer/rfxbands"
)

const chanRowDp unit.Dp = 36

// ChannelOverviewWidget displays per-channel signal activity for a BandSet.
// Each band occupies a horizontal row; channels are rendered as VU-meter bars
// positioned by frequency across the band set's span. Clicking a channel
// selects it (highlighting its frequency range in the spectrum) and shows its
// max received power. Clicking again deselects.
type ChannelOverviewWidget struct {
	// UI-goroutine-only fields (no mutex needed):
	selectedBand int
	selectedChan int
	clicks       [][]gesture.Click

	mu           sync.Mutex
	min, max     float64
	bandSet      *rfxbands.BandSet
	startFreqKHz int
	freqStepHz   int
	values       []float64
	chanMaxPower [][]float64
	spanMinKHz   int
	spanMaxKHz   int
}

func NewChannelOverviewWidget() *ChannelOverviewWidget {
	return &ChannelOverviewWidget{
		min:          -120,
		max:          0,
		selectedBand: -1,
		selectedChan: -1,
	}
}

func (cw *ChannelOverviewWidget) SetBandSet(bs *rfxbands.BandSet) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.bandSet = bs
	if bs == nil {
		cw.clicks = nil
		cw.chanMaxPower = nil
		cw.spanMinKHz = 0
		cw.spanMaxKHz = 0
	} else {
		cw.spanMinKHz, cw.spanMaxKHz = bs.SpanKHz()
		cw.clicks = make([][]gesture.Click, len(bs.Bands))
		cw.chanMaxPower = make([][]float64, len(bs.Bands))
		for bi, band := range bs.Bands {
			cw.clicks[bi] = make([]gesture.Click, len(band.Channels))
			cw.chanMaxPower[bi] = make([]float64, len(band.Channels))
			for ci := range cw.chanMaxPower[bi] {
				cw.chanMaxPower[bi][ci] = cw.min
			}
		}
	}
	// Reset selection since band set changed.
	cw.selectedBand = -1
	cw.selectedChan = -1
}

func (cw *ChannelOverviewWidget) SetFreqConfig(startFreqKHz, freqStepHz int) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.startFreqKHz = startFreqKHz
	cw.freqStepHz = freqStepHz
}

// SetAmplitude updates the displayed amplitude range and resets per-channel max-hold data.
func (cw *ChannelOverviewWidget) SetAmplitude(bottom, top int) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.min = float64(bottom)
	cw.max = float64(top)
	for bi := range cw.chanMaxPower {
		for ci := range cw.chanMaxPower[bi] {
			cw.chanMaxPower[bi][ci] = float64(bottom)
		}
	}
}

func (cw *ChannelOverviewWidget) Update(values []float64) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if len(cw.values) != len(values) {
		cw.values = make([]float64, len(values))
	}
	copy(cw.values, values)
	if cw.bandSet == nil || cw.freqStepHz == 0 || len(cw.values) == 0 {
		return
	}
	startHz := cw.startFreqKHz * 1000
	for bi, band := range cw.bandSet.Bands {
		if bi >= len(cw.chanMaxPower) {
			break
		}
		for ci, ch := range band.Channels {
			if ci >= len(cw.chanMaxPower[bi]) {
				break
			}
			loHz := ch.CenterKHz*1000 - ch.WidthKHz*1000/2
			hiHz := ch.CenterKHz*1000 + ch.WidthKHz*1000/2
			loIdx := (loHz - startHz) / cw.freqStepHz
			hiIdx := (hiHz - startHz) / cw.freqStepHz
			if loIdx < 0 {
				loIdx = 0
			}
			if hiIdx >= len(cw.values) {
				hiIdx = len(cw.values) - 1
			}
			if loIdx > hiIdx {
				continue
			}
			maxPow := cw.values[loIdx]
			for _, v := range cw.values[loIdx+1 : hiIdx+1] {
				if v > maxPow {
					maxPow = v
				}
			}
			if maxPow > cw.chanMaxPower[bi][ci] {
				cw.chanMaxPower[bi][ci] = maxPow
			}
		}
	}
}

// SelectedRange returns the frequency range (kHz) of the selected channel,
// or ok=false if no channel is selected.
func (cw *ChannelOverviewWidget) SelectedRange() (loKHz, hiKHz int, ok bool) {
	if cw.selectedBand < 0 {
		return 0, 0, false
	}
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if cw.bandSet == nil || cw.selectedBand >= len(cw.bandSet.Bands) {
		return 0, 0, false
	}
	band := cw.bandSet.Bands[cw.selectedBand]
	if cw.selectedChan >= len(band.Channels) {
		return 0, 0, false
	}
	ch := band.Channels[cw.selectedChan]
	return ch.CenterKHz - ch.WidthKHz/2, ch.CenterKHz + ch.WidthKHz/2, true
}

func (cw *ChannelOverviewWidget) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	cw.mu.Lock()
	bs := cw.bandSet
	clicks := cw.clicks
	chanMaxPower := cw.chanMaxPower
	spanMinKHz := cw.spanMinKHz
	spanMaxKHz := cw.spanMaxKHz
	startFreqKHz := cw.startFreqKHz
	freqStepHz := cw.freqStepHz
	values := cw.values
	ampMin := cw.min
	ampMax := cw.max
	cw.mu.Unlock()

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

	spanKHz := spanMaxKHz - spanMinKHz
	if spanKHz <= 0 {
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

	// Compute current per-channel max from the latest sweep values.
	// channelCurrent[bi][ci] = max dBm in channel's freq range this sweep.
	channelCurrent := make([][]float64, numBands)
	for bi, band := range bs.Bands {
		channelCurrent[bi] = make([]float64, len(band.Channels))
		for ci, ch := range band.Channels {
			channelCurrent[bi][ci] = ampMin
			if freqStepHz > 0 && len(values) > 0 {
				startHz := startFreqKHz * 1000
				loHz := ch.CenterKHz*1000 - ch.WidthKHz*1000/2
				hiHz := ch.CenterKHz*1000 + ch.WidthKHz*1000/2
				loIdx := (loHz - startHz) / freqStepHz
				hiIdx := (hiHz - startHz) / freqStepHz
				if loIdx < 0 {
					loIdx = 0
				}
				if hiIdx >= len(values) {
					hiIdx = len(values) - 1
				}
				if loIdx <= hiIdx {
					cur := values[loIdx]
					for _, v := range values[loIdx+1 : hiIdx+1] {
						if v > cur {
							cur = v
						}
					}
					channelCurrent[bi][ci] = cur
				}
			}
		}
	}

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

			chLeftKHz := ch.CenterKHz - ch.WidthKHz/2
			chRightKHz := ch.CenterKHz + ch.WidthKHz/2
			leftX := (chLeftKHz - spanMinKHz) * w / spanKHz
			rightX := (chRightKHz - spanMinKHz) * w / spanKHz
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
