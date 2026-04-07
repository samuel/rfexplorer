package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/samuel/rfexplorer/rfx"
)

// defaultMargin is a margin applied in multiple places to give
// widgets room to breathe.
var defaultMargin = unit.Dp(10)

// firstUnusedPresetIndex returns the lowest index not already used by any preset.
func firstUnusedPresetIndex(presets []*rfx.Preset) int {
	used := make(map[int]bool, len(presets))
	for _, p := range presets {
		used[p.Index] = true
	}
	for i := range 100 {
		if !used[i] {
			return i
		}
	}
	return 0
}

// layoutEditorField renders an editor inside a bordered, padded box.
func layoutEditorField(gtx layout.Context, th *material.Theme, ed *widget.Editor) layout.Dimensions {
	borderColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
	if gtx.Focused(ed) {
		borderColor = th.Palette.ContrastBg
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			rr := gtx.Dp(4)
			rect := image.Rectangle{Max: gtx.Constraints.Min}
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
				clip.UniformRRect(rect, rr).Op(gtx.Ops))
			border := clip.Stroke{
				Path:  clip.UniformRRect(rect, rr).Path(gtx.Ops),
				Width: float32(gtx.Dp(1)),
			}.Op()
			paint.FillShape(gtx.Ops, borderColor, border)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.UniformInset(unit.Dp(6)).Layout(gtx,
				material.Editor(th, ed, "").Layout)
		}),
	)
}

type UI struct {
	Theme     *material.Theme
	Waterfall *WaterfallWidget
	Spectrum  *SpectrumWidget
	// WifiSpectrum *WifiSpectrumWidget

	mu                   sync.RWMutex
	rfe                  *rfx.RFExplorer
	centerFrequencyInput *widget.Editor
	frequencySpanInput   *widget.Editor
	receivingPreset      bool
	receivedPresets      []*rfx.Preset
	presets              []*rfx.Preset
	presetList           *widget.List
	presetClicks         []gesture.Click
	rfeConfig            rfx.CurrentConfigPacket
	lcdEnabled      bool
	lcdSwitch       *widget.Bool
	expModuleActive bool
	expModuleSwitch *widget.Bool
	presetEditor    *presetEditor
	addPresetBtn    widget.Clickable
	leftBg, leftFg color.NRGBA
}

func NewUI(rfe *rfx.RFExplorer) *UI {
	theme := material.NewTheme() // gofont.Collection())
	theme.Palette = material.Palette{
		Fg:         color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		Bg:         color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
		ContrastBg: color.NRGBA{R: 0x3f, G: 0x51, B: 0xb5, A: 0xff},
		ContrastFg: color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
	}
	ui := &UI{
		Theme: theme,

		rfe:                  rfe,
		lcdEnabled:      true,
		lcdSwitch:       &widget.Bool{Value: true},
		expModuleSwitch: &widget.Bool{},
		presetEditor:    newPresetEditor(),
		centerFrequencyInput: &widget.Editor{SingleLine: true, Submit: true},
		frequencySpanInput:   &widget.Editor{SingleLine: true, Submit: true},
		presetList: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		leftBg: color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		leftFg: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
	w := NewWaterfall(112, 256, -120, 0, true, true)
	ui.Waterfall = NewWaterfallWidget(w)
	ui.Spectrum = NewSpectrumWidget()
	// ui.WifiSpectrum = NewWifiSpectrumWidget(2400000, 2495000, ui.Theme)
	return ui
}

func (ui *UI) setRFEConfig(cfg rfx.CurrentConfigPacket) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.rfeConfig = cfg
	ui.expModuleActive = cfg.ExpModuleActive
	ui.expModuleSwitch.Value = cfg.ExpModuleActive
	ui.Spectrum.SetFreqConfig(cfg.StartFreqKHZ, cfg.FreqStepHZ)
	centerFreqMHZ := float64(1000*cfg.StartFreqKHZ+cfg.FreqStepHZ*(cfg.SweepSteps-1)/2) / 1e6
	ui.centerFrequencyInput.SetText(strconv.FormatFloat(centerFreqMHZ+0.0004, 'f', 3, 64))
	freqSpanMHZ := float64((cfg.SweepSteps-1)*cfg.FreqStepHZ) / 1e6
	ui.frequencySpanInput.SetText(strconv.FormatFloat(freqSpanMHZ+0.0004, 'f', 3, 64))
	ui.Spectrum.Reset()
	// ui.WifiSpectrum.Reset()
}

func (ui *UI) setPresets(presets []*rfx.Preset) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.presets = presets
	ui.presetClicks = make([]gesture.Click, len(presets))
	if ui.presetEditor.CloseOnRefresh {
		ui.presetEditor.CloseOnRefresh = false
		ui.presetEditor.Active = false
	}
}

// Run handles window events and renders the application.
func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops
	stopCh := make(chan any)
	defer close(stopCh)

	go func() {
		for {
			var pkt rfx.Packet
			select {
			case pkt = <-ui.rfe.Chan():
			case <-stopCh:
				return
			}
			// TODO: make everything here concurrent safe
			switch pkt := pkt.(type) {
			case *rfx.SweepDataPacket:
				ui.Waterfall.w.Update(pkt.Samples)
				ui.Spectrum.Update(pkt.Samples)
				// ui.WifiSpectrum.Update(pkt.Samples)
			case *rfx.CurrentConfigPacket:
				ui.setRFEConfig(*pkt)
				fmt.Printf("%+#v\n", pkt)
			case *rfx.Preset:
				if !ui.receivingPreset {
					ui.receivingPreset = true
					ui.receivedPresets = nil
					ui.setPresets(nil)
				}
				ui.receivedPresets = append(ui.receivedPresets, pkt)
				// ui.presetClicks = make([]gesture.Click, len(ui.presets))
			case *rfx.EndOfPresetsPacket:
				filtered := ui.receivedPresets[:0]
				for _, p := range ui.receivedPresets {
					if p.Name != "" {
						filtered = append(filtered, p)
					}
				}
				ui.setPresets(filtered)
				ui.receivingPreset = false
				ui.receivedPresets = nil
				w.Invalidate()
			case *rfx.UnhandledPacket:
				fmt.Printf("Unhandled packet: %q\n", pkt.Data)
			default:
				fmt.Printf("%T\n", pkt)
			}
		}
	}()

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)

			if ui.lcdSwitch.Value != ui.lcdEnabled {
				ui.lcdEnabled = ui.lcdSwitch.Value
				if err := ui.rfe.SetLCDEnabled(ui.lcdEnabled); err != nil {
					log.Printf("Failed to set LCD enabled: %s", err)
				}
			}
			if ui.expModuleSwitch.Value != ui.expModuleActive {
				ui.expModuleActive = ui.expModuleSwitch.Value
				var err error
				if ui.expModuleActive {
					err = ui.rfe.SwitchModuleExp()
				} else {
					err = ui.rfe.SwitchModuleMain()
				}
				if err != nil {
					log.Printf("Failed to switch module: %s", err)
				}
			}

			pe := ui.presetEditor
			if pe.PendingCancel {
				pe.PendingCancel = false
				ui.mu.Lock()
				pe.Active = false
				ui.mu.Unlock()
			}
			if pe.PendingApply {
				pe.PendingApply = false
				p := pe.ToPreset()
				if p.Mainboard {
					if err := ui.rfe.SwitchModuleMain(); err != nil {
						log.Printf("Failed to switch to main module: %s", err)
					}
				} else {
					if err := ui.rfe.SwitchModuleExp(); err != nil {
						log.Printf("Failed to switch to expansion module: %s", err)
					}
				}
				if err := ui.rfe.SetAnalyzerConfig(p.MinFreqKHz, p.MaxFreqKHz, p.AmpTopDBm, p.AmpBottomDBm, 0); err != nil {
					log.Printf("Failed to apply preset: %s", err)
				}
			}
			if pe.PendingSave {
				pe.PendingSave = false
				p := pe.ToPreset()
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := ui.rfe.UpdatePreset(ctx, &p); err != nil {
						log.Printf("Failed to save preset: %s", err)
						return
					}
					ui.mu.Lock()
					pe.CloseOnRefresh = true
					ui.mu.Unlock()
					if err := ui.rfe.RequestPresets(); err != nil {
						log.Printf("Failed to request presets: %s", err)
					}
				}()
			}
			if pe.PendingDelete {
				pe.PendingDelete = false
				blank := rfx.Preset{Index: pe.Index}
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := ui.rfe.UpdatePreset(ctx, &blank); err != nil {
						log.Printf("Failed to delete preset: %s", err)
						return
					}
					ui.mu.Lock()
					pe.CloseOnRefresh = true
					ui.mu.Unlock()
					if err := ui.rfe.RequestPresets(); err != nil {
						log.Printf("Failed to request presets: %s", err)
					}
				}()
			}
		case key.Event:
			switch e.Name {
			case key.NameEscape:
				return nil
			}
		case app.DestroyEvent:
			return e.Err
		}
	}
}

func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	// TODO: should have more granular locking
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	// inset is used to add padding around the window border.
	// inset := layout.UniformInset(defaultMargin)
	// return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
	// 	return ui.LayoutMain(ui.Theme, gtx)
	// })
	return ui.LayoutMain(ui.Theme, gtx)
}

func (ui *UI) LayoutMain(th *material.Theme, gtx layout.Context) layout.Dimensions {
	paint.Fill(gtx.Ops, ui.Theme.Bg)

	for {
		e, ok := ui.centerFrequencyInput.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			centerFreqMHZ, err := strconv.ParseFloat(e.Text, 64)
			if err == nil {
				freqSpanMHZ := float64(ui.rfeConfig.FreqStepHZ*(ui.rfeConfig.SweepSteps-1)) / 1e6
				startFreqKHZ := int((centerFreqMHZ - freqSpanMHZ/2) * 1000)
				endFreqKHZ := int((centerFreqMHZ + freqSpanMHZ/2) * 1000)
				if ui.rfe.SetAnalyzerConfig(startFreqKHZ, endFreqKHZ, ui.rfeConfig.AmpTopDBM, ui.rfeConfig.AmpBottomDBM, 0); err != nil {
					log.Printf("Failed to set analyzer config: %s", err)
				}
			}
		}
	}
	for {
		e, ok := ui.frequencySpanInput.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			freqSpanMHZ, err := strconv.ParseFloat(e.Text, 64)
			if err == nil {
				centerFreqMHZ := (float64(1000*ui.rfeConfig.StartFreqKHZ) + float64(ui.rfeConfig.FreqStepHZ*(ui.rfeConfig.SweepSteps-1))/2) / 1e6
				startFreqKHZ := int(math.Round((centerFreqMHZ - freqSpanMHZ/2) * 1000))
				endFreqKHZ := int(math.Round((centerFreqMHZ + freqSpanMHZ/2) * 1000))
				fmt.Println(startFreqKHZ, endFreqKHZ, centerFreqMHZ)
				if ui.rfe.SetAnalyzerConfig(startFreqKHZ, endFreqKHZ, ui.rfeConfig.AmpTopDBM, ui.rfeConfig.AmpBottomDBM, 0); err != nil {
					log.Printf("Failed to set analyzer config: %s", err)
				}
			}
		}
	}
	for i := range ui.presetClicks {
		click := &ui.presetClicks[i]
		for {
			e, ok := click.Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				ui.presetEditor.OpenPreset(ui.presets[i])
			}
		}
	}
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			pr := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
			paint.Fill(gtx.Ops, ui.leftBg)
			pr.Pop()
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.Spectrum.Layout(gtx, th)
					// return ui.WifiSpectrum.Layout(gtx)
				}),
				// layout.Rigid(layout.Spacer{Height: defaultMargin}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							freq := strconv.FormatFloat(float64(ui.rfeConfig.StartFreqKHZ)/1000.0+0.0004, 'f', 3, 64)
							l := material.Body1(ui.Theme, freq)
							l.Color = ui.leftFg
							return l.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							endFreqHz := 1000*ui.rfeConfig.StartFreqKHZ + ui.rfeConfig.FreqStepHZ*(ui.rfeConfig.SweepSteps-1)/2
							freq := strconv.FormatFloat(float64(endFreqHz)/1000000.0+0.0004, 'f', 3, 64)
							l := material.Body1(ui.Theme, freq)
							l.Color = ui.leftFg
							l.Alignment = text.Middle
							return l.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							endFreqHz := 1000*ui.rfeConfig.StartFreqKHZ + ui.rfeConfig.FreqStepHZ*(ui.rfeConfig.SweepSteps-1)
							freq := strconv.FormatFloat(float64(endFreqHz)/1000000.0+0.0004, 'f', 3, 64)
							l := material.Body1(ui.Theme, freq)
							l.Color = ui.leftFg
							l.Alignment = text.End
							return l.Layout(gtx)
						}))
				}),
				layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
					return ui.Waterfall.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(240)
			gtx.Constraints.Min.X = gtx.Dp(240)
			inset := layout.UniformInset(defaultMargin)
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(material.Body2(th, "Center Frequency (MHz)").Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutEditorField(gtx, th, ui.centerFrequencyInput)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(material.Body2(th, "Frequency Span (MHz)").Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutEditorField(gtx, th, ui.frequencySpanInput)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(material.Switch(th, ui.lcdSwitch, "LCD On").Layout),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(material.Body1(th, "LCD On").Layout),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(material.Switch(th, ui.expModuleSwitch, "Expansion Module").Layout),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(material.Body1(th, "Expansion Module").Layout),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if ui.presetEditor.Active {
							return ui.presetEditor.Layout(gtx, th)
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(material.Body2(th, "Presets").Layout),
							layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return material.List(ui.Theme, ui.presetList).Layout(gtx, len(ui.presets), func(gtx layout.Context, index int) layout.Dimensions {
									isHovered := ui.presetClicks[index].Hovered()
									bgColor := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
									if isHovered {
										bgColor = color.NRGBA{R: 0xe0, G: 0xe8, B: 0xff, A: 0xff}
									}
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return layout.Stack{}.Layout(gtx,
										layout.Expanded(func(gtx layout.Context) layout.Dimensions {
											paint.FillShape(gtx.Ops, bgColor,
												clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Op())
											return layout.Dimensions{Size: gtx.Constraints.Min}
										}),
										layout.Stacked(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Constraints.Max.X
											dims := layout.UniformInset(unit.Dp(8)).Layout(gtx,
												material.Body1(ui.Theme, ui.presets[index].Name).Layout)
											pr := clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops)
											ui.presetClicks[index].Add(gtx.Ops)
											pr.Pop()
											return dims
										}),
									)
								})
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if ui.addPresetBtn.Clicked(gtx) {
									ui.presetEditor.OpenNew(firstUnusedPresetIndex(ui.presets))
								}
								return material.Button(th, &ui.addPresetBtn, "Add Preset").Layout(gtx)
							}),
						)
					}),
				)
			})
		}),
	)
}
