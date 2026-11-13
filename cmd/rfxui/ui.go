package main

import (
	"context"
	"image"
	"image/color"
	"log"
	"math"
	"path/filepath"
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
	"samuelks.com/rfx"
	"samuelks.com/rfx/bands"
)

// defaultMargin is a margin applied in multiple places to give
// widgets room to breathe.
var defaultMargin = unit.Dp(10)

// firstUnusedPresetIndex returns the lowest of the device's slots not already
// used by any preset.
func firstUnusedPresetIndex(presets []rfx.Preset, slots int) int {
	used := make(map[int]bool, len(presets))
	for _, p := range presets {
		used[p.Index] = true
	}
	for i := range slots {
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
		borderColor = th.ContrastBg
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
	Theme           *material.Theme
	Waterfall       *WaterfallWidget
	Spectrum        *SpectrumWidget
	ChannelOverview *ChannelOverviewWidget
	// WifiSpectrum *WifiSpectrumWidget

	mu                   sync.RWMutex
	rfe                  *rfx.Device
	centerFrequencyInput *widget.Editor
	frequencySpanInput   *widget.Editor
	presets              []rfx.Preset
	maxPresets           int
	presetList           *widget.List
	presetClicks         []gesture.Click
	rfeConfig            rfx.SweepConfig
	lcdEnabled           bool
	lcdSwitch            *widget.Bool
	expModuleActive      bool
	expModuleSwitch      *widget.Bool
	presetEditor         *presetEditor
	addPresetBtn         widget.Clickable
	bandSets             []*bands.BandSet
	uiConfig             *UIConfig
	uiConfigPath         string
	leftBg, leftFg       color.NRGBA
}

func NewUI(rfe *rfx.Device) *UI {
	theme := material.NewTheme() // gofont.Collection())
	theme.Palette = material.Palette{
		Fg:         color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		Bg:         color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
		ContrastBg: color.NRGBA{R: 0x3f, G: 0x51, B: 0xb5, A: 0xff},
		ContrastFg: color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
	}

	// Load band sets: builtins merged with any user-defined sets.
	var customSets []*bands.BandSet
	cfgDir, err := configDir()
	var uiConfigPath string
	if err != nil {
		log.Printf("Could not determine config dir: %s", err)
	} else {
		uiConfigPath = filepath.Join(cfgDir, "ui.json")
		custom, err := bands.LoadFile(filepath.Join(cfgDir, "bands.json"))
		if err != nil {
			log.Printf("Could not load custom band sets: %s", err)
		} else {
			customSets = custom
		}
	}
	bandSets := bands.MergeSets(bands.Builtins, customSets)

	cfg, err := loadUIConfig(uiConfigPath)
	if err != nil {
		log.Printf("Could not load UI config: %s", err)
		cfg = &UIConfig{}
	}

	pe := newPresetEditor()
	pe.SetBandSets(bandSets)

	ui := &UI{
		Theme: theme,

		rfe:                  rfe,
		maxPresets:           rfe.Info().MaxPresets(),
		lcdEnabled:           true,
		lcdSwitch:            &widget.Bool{Value: true},
		expModuleSwitch:      &widget.Bool{},
		presetEditor:         pe,
		centerFrequencyInput: &widget.Editor{SingleLine: true, Submit: true},
		frequencySpanInput:   &widget.Editor{SingleLine: true, Submit: true},
		presetList: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		bandSets:     bandSets,
		uiConfig:     cfg,
		uiConfigPath: uiConfigPath,
		leftBg:       color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		leftFg:       color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
	w := NewWaterfall(112, 256, -120, 0, true, true)
	ui.Waterfall = NewWaterfallWidget(w)
	ui.Spectrum = NewSpectrumWidget()
	ui.ChannelOverview = NewChannelOverviewWidget()
	// ui.WifiSpectrum = NewWifiSpectrumWidget(2400000, 2495000, ui.Theme)
	return ui
}

// bandSetByName returns the BandSet with the given name, or nil if not found.
func (ui *UI) bandSetByName(name string) *bands.BandSet {
	for _, bs := range ui.bandSets {
		if bs.Name == name {
			return bs
		}
	}
	return nil
}

// saveUIConfig persists uiConfig to disk in the background, logging any error.
func (ui *UI) saveUIConfig() {
	if ui.uiConfigPath == "" {
		return
	}
	cfg := ui.uiConfig
	path := ui.uiConfigPath
	go func() {
		if err := cfg.Save(path); err != nil {
			log.Printf("Failed to save UI config: %s", err)
		}
	}()
}

func (ui *UI) setRFEConfig(cfg rfx.SweepConfig) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	startKHz, stepHz := int(cfg.Start/rfx.KHz), int(cfg.Step)
	ui.rfeConfig = cfg
	ui.expModuleActive = cfg.ExpansionActive
	ui.expModuleSwitch.Value = cfg.ExpansionActive
	ui.Spectrum.SetFreqConfig(startKHz, stepHz)
	ui.Spectrum.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.ChannelOverview.SetFreqConfig(startKHz, stepHz)
	ui.ChannelOverview.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.Waterfall.w.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.centerFrequencyInput.SetText(formatMHz(cfg.Center()))
	ui.frequencySpanInput.SetText(formatMHz(cfg.Span()))
	ui.Spectrum.Reset()
}

// formatMHz formats a frequency in MHz to the nearest kHz.
func formatMHz(f rfx.Frequency) string {
	return strconv.FormatFloat(f.MHz(), 'f', 3, 64)
}

// parseMHz parses a frequency entered in MHz.
func parseMHz(s string) (rfx.Frequency, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return rfx.Frequency(math.Round(v * float64(rfx.MHz))), nil
}

// setPresets replaces the preset list with the used slots of presets.
func (ui *UI) setPresets(presets []rfx.Preset) {
	used := make([]rfx.Preset, 0, len(presets))
	for _, p := range presets {
		if p.Name != "" {
			used = append(used, p)
		}
	}
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.presets = used
	ui.presetClicks = make([]gesture.Click, len(used))
	if ui.presetEditor.CloseOnRefresh {
		ui.presetEditor.CloseOnRefresh = false
		ui.presetEditor.Active = false
	}
}

// refreshPresets reads the presets from the device and redraws.
func (ui *UI) refreshPresets(w *app.Window) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	presets, err := ui.rfe.Presets(ctx)
	if err != nil {
		log.Printf("Failed to read presets: %s", err)
		return
	}
	ui.setPresets(presets)
	w.Invalidate()
}

// command runs a device command with a timeout, logging failures. Switching
// modules alone can take the device a few seconds.
func (ui *UI) command(name string, f func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f(ctx); err != nil {
		log.Printf("Failed to %s: %s", name, err)
	}
}

// setSweep changes the frequency and amplitude range, waiting briefly for
// the device to confirm.
func (ui *UI) setSweep(ctx context.Context, start, stop rfx.Frequency, ampTop, ampBottom int) error {
	_, err := ui.rfe.SetSweep(ctx, rfx.SweepRequest{Start: start, Stop: stop, AmpTop: ampTop, AmpBottom: ampBottom})
	return err
}

func (ui *UI) selectModule(ctx context.Context, mainboard bool) error {
	m := rfx.ModuleExpansion
	if mainboard {
		m = rfx.ModuleMain
	}
	_, err := ui.rfe.SelectModule(ctx, m)
	return err
}

// Run handles window events and renders the application.
func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops
	stopCh := make(chan any)
	defer close(stopCh)

	sub := ui.rfe.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep | rfx.KindConfig})
	defer sub.Close()
	ui.setRFEConfig(ui.rfe.Config())
	go ui.refreshPresets(w)

	go func() {
		var samples []float64
		for {
			var msg rfx.Message
			select {
			case m, ok := <-sub.C():
				if !ok {
					if err := ui.rfe.Err(); err != nil {
						log.Printf("Connection lost: %s", err)
					}
					return
				}
				msg = m
			case <-stopCh:
				return
			}
			// TODO: make everything here concurrent safe
			switch msg := msg.(type) {
			case *rfx.Sweep:
				samples = samples[:0]
				for _, v := range msg.Samples {
					samples = append(samples, float64(v))
				}
				ui.Waterfall.w.Update(samples)
				ui.Spectrum.Update(samples)
				ui.ChannelOverview.Update(samples)
			case *rfx.SweepConfig:
				ui.setRFEConfig(*msg)
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

			// Sync channel selection highlight to spectrum after layout
			// processes click events.
			if lo, hi, ok := ui.ChannelOverview.SelectedRange(); ok {
				ui.Spectrum.SetHighlight(lo, hi)
			} else {
				ui.Spectrum.ClearHighlight()
			}

			if ui.lcdSwitch.Value != ui.lcdEnabled {
				ui.lcdEnabled = ui.lcdSwitch.Value
				ui.command("set LCD", func(ctx context.Context) error { return ui.rfe.SetLCD(ctx, ui.lcdEnabled) })
			}
			if ui.expModuleSwitch.Value != ui.expModuleActive {
				ui.expModuleActive = ui.expModuleSwitch.Value
				ui.command("switch module", func(ctx context.Context) error { return ui.selectModule(ctx, !ui.expModuleActive) })
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
				ui.command("apply preset", func(ctx context.Context) error {
					if err := ui.selectModule(ctx, p.Mainboard); err != nil {
						return err
					}
					return ui.setSweep(ctx, p.Start, p.Stop, p.AmpTop, p.AmpBottom)
				})
				bandSetName := pe.SelectedBandSetName()
				ui.uiConfig.SetBandSetFor(p.Index, bandSetName)
				bs := ui.bandSetByName(bandSetName)
				ui.Spectrum.SetBandSet(bs)
				ui.ChannelOverview.SetBandSet(bs)
				ui.saveUIConfig()
			}
			if pe.PendingSave {
				pe.PendingSave = false
				p := pe.ToPreset()
				bandSetName := pe.SelectedBandSetName()
				ui.uiConfig.SetBandSetFor(p.Index, bandSetName)
				bs := ui.bandSetByName(bandSetName)
				ui.Spectrum.SetBandSet(bs)
				ui.ChannelOverview.SetBandSet(bs)
				ui.saveUIConfig()
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := ui.rfe.WritePreset(ctx, p); err != nil {
						log.Printf("Failed to save preset: %s", err)
						return
					}
					ui.mu.Lock()
					pe.CloseOnRefresh = true
					ui.mu.Unlock()
					ui.refreshPresets(w)
				}()
			}
			if pe.PendingDelete {
				pe.PendingDelete = false
				index := pe.Index
				ui.uiConfig.SetBandSetFor(pe.Index, "")
				ui.saveUIConfig()
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := ui.rfe.DeletePreset(ctx, index); err != nil {
						log.Printf("Failed to delete preset: %s", err)
						return
					}
					ui.mu.Lock()
					pe.CloseOnRefresh = true
					ui.mu.Unlock()
					ui.refreshPresets(w)
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
			if center, err := parseMHz(e.Text); err == nil {
				span := ui.rfeConfig.Span()
				ui.command("set center frequency", func(ctx context.Context) error {
					return ui.setSweep(ctx, center-span/2, center+span/2, ui.rfeConfig.AmpTop, ui.rfeConfig.AmpBottom)
				})
			}
		}
	}
	for {
		e, ok := ui.frequencySpanInput.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			if span, err := parseMHz(e.Text); err == nil {
				center := ui.rfeConfig.Center()
				ui.command("set frequency span", func(ctx context.Context) error {
					return ui.setSweep(ctx, center-span/2, center+span/2, ui.rfeConfig.AmpTop, ui.rfeConfig.AmpBottom)
				})
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
				p := &ui.presets[i]
				ui.presetEditor.OpenPreset(p, ui.uiConfig.BandSetFor(p.Index))
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
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.ChannelOverview.Layout(gtx, th)
				}),
				// layout.Rigid(layout.Spacer{Height: defaultMargin}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Body1(ui.Theme, formatMHz(ui.rfeConfig.Start))
							l.Color = ui.leftFg
							return l.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Body1(ui.Theme, formatMHz(ui.rfeConfig.Center()))
							l.Color = ui.leftFg
							l.Alignment = text.Middle
							return l.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Body1(ui.Theme, formatMHz(ui.rfeConfig.Stop()))
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
									ui.presetEditor.OpenNew(firstUnusedPresetIndex(ui.presets, ui.maxPresets), "")
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
