package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"samuelks.com/bandplan"
	"samuelks.com/rfx"
)

// defaultMargin is a margin applied in multiple places to give
// widgets room to breathe.
var defaultMargin = unit.Dp(10)

var errorColor = color.NRGBA{R: 0xc0, G: 0x10, B: 0x10, A: 0xff}

// commandTimeout bounds a device command. Switching modules alone can take
// the device a few seconds.
const commandTimeout = 10 * time.Second

// firstUnusedPresetIndex returns the lowest of the device's slots not already
// used by any preset, or -1 if all are used.
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
	return -1
}

// matchPreset returns the preset whose range and board match cfg, allowing
// for the device rounding the range to whole sweep steps.
func matchPreset(presets []rfx.Preset, cfg rfx.SweepConfig) (rfx.Preset, bool) {
	tol := max(cfg.Step, rfx.KHz)
	near := func(a, b rfx.Frequency) bool { return max(a-b, b-a) <= tol }
	for _, p := range presets {
		if p.Mainboard == !cfg.ExpansionActive && near(p.Start, cfg.Start) && near(p.Stop, cfg.Stop()) {
			return p, true
		}
	}
	return rfx.Preset{}, false
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

// command is a device command run by the command worker.
type command struct {
	name string
	run  func(ctx context.Context) error
	// done, if set, runs on the UI goroutine with the result.
	done func(err error)
}

// UI is the desktop application. All of its state is owned by the goroutine
// that runs Run: other goroutines hand it work through post.
type UI struct {
	Theme           *material.Theme
	Waterfall       *WaterfallWidget
	Spectrum        *SpectrumWidget
	ChannelOverview *ChannelOverviewWidget

	dev     *rfx.Device
	logger  *slog.Logger
	window  *app.Window
	updates chan func()
	cmds    chan command
	stop    chan struct{}

	status    string
	statusErr bool

	config               rfx.SweepConfig
	centerFrequencyInput *widget.Editor
	frequencySpanInput   *widget.Editor
	presets              []rfx.Preset
	presetsLoaded        bool
	maxPresets           int
	presetList           *widget.List
	presetClicks         []gesture.Click
	lcdSwitch            *widget.Bool
	expModuleSwitch      *widget.Bool
	presetEditor         *presetEditor
	addPresetBtn         widget.Clickable
	bandSets             []*bandplan.BandSet
	uiConfig             *UIConfig
	saver                *configSaver
	leftBg, leftFg       color.NRGBA
}

func NewUI(dev *rfx.Device, logger *slog.Logger) *UI {
	theme := material.NewTheme()
	theme.Palette = material.Palette{
		Fg:         color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		Bg:         color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
		ContrastBg: color.NRGBA{R: 0x3f, G: 0x51, B: 0xb5, A: 0xff},
		ContrastFg: color.NRGBA{R: 0xef, G: 0xef, B: 0xef, A: 0xff},
	}

	maxPresets := dev.Info().MaxPresets()
	ui := &UI{
		Theme:           theme,
		Waterfall:       NewWaterfallWidget(256),
		Spectrum:        NewSpectrumWidget(),
		ChannelOverview: NewChannelOverviewWidget(),

		dev:     dev,
		logger:  logger,
		updates: make(chan func(), 64),
		cmds:    make(chan command, 8),
		stop:    make(chan struct{}),

		maxPresets: maxPresets,
		// The LCD state can't be read from the device; it starts on.
		lcdSwitch:            &widget.Bool{Value: true},
		expModuleSwitch:      &widget.Bool{},
		presetEditor:         newPresetEditor(maxPresets),
		centerFrequencyInput: &widget.Editor{SingleLine: true, Submit: true},
		frequencySpanInput:   &widget.Editor{SingleLine: true, Submit: true},
		presetList:           &widget.List{List: layout.List{Axis: layout.Vertical}},
		uiConfig:             &UIConfig{},
		leftBg:               color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		leftFg:               color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}

	// Load band sets: builtins merged with any user-defined sets.
	var customSets []*bandplan.BandSet
	if cfgDir, err := configDir(); err != nil {
		ui.setStatus(fmt.Errorf("finding config directory: %w", err))
	} else {
		path := filepath.Join(cfgDir, "ui.json")
		ui.saver = newConfigSaver(path, func(err error) {
			ui.post(func() { ui.setStatus(err) })
		})
		if custom, err := bandplan.LoadFile(filepath.Join(cfgDir, "bands.json")); err != nil {
			ui.setStatus(fmt.Errorf("loading custom band sets: %w", err))
		} else {
			customSets = custom
		}
		if cfg, err := loadUIConfig(path); err != nil {
			ui.setStatus(err)
		} else {
			ui.uiConfig = cfg
		}
	}
	ui.bandSets = bandplan.MergeSets(bandplan.Builtins(), customSets)
	ui.presetEditor.SetBandSets(ui.bandSets)
	return ui
}

// Close writes any pending config save. Call it after Run returns.
func (ui *UI) Close() {
	if ui.saver != nil {
		ui.saver.Close()
	}
}

// post runs f on the UI goroutine before the next frame. It is dropped once
// Run has returned.
func (ui *UI) post(f func()) {
	select {
	case ui.updates <- f:
		ui.window.Invalidate()
	case <-ui.stop:
	}
}

// setStatus shows err in the status line, or clears it if err is nil.
func (ui *UI) setStatus(err error) {
	if err == nil {
		ui.status, ui.statusErr = "", false
		return
	}
	ui.logger.Error(err.Error())
	ui.status, ui.statusErr = err.Error(), true
}

// do queues a device command for the command worker so the UI never waits
// on the device.
func (ui *UI) do(name string, run func(ctx context.Context) error, done func(err error)) {
	select {
	case ui.cmds <- command{name: name, run: run, done: done}:
		ui.status, ui.statusErr = name+"...", false
	default:
		err := fmt.Errorf("%s: device busy, try again", name)
		ui.setStatus(err)
		if done != nil {
			done(err)
		}
	}
}

// runCommands executes queued commands one at a time.
func (ui *UI) runCommands() {
	for {
		select {
		case <-ui.stop:
			return
		case c := <-ui.cmds:
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			err := c.run(ctx)
			cancel()
			ui.post(func() {
				if err != nil {
					ui.setStatus(fmt.Errorf("%s: %w", c.name, err))
				} else if ui.status == c.name+"..." {
					ui.setStatus(nil)
				}
				if c.done != nil {
					c.done(err)
				}
			})
		}
	}
}

// receive forwards device messages to the UI goroutine.
func (ui *UI) receive(sub *rfx.Subscription) {
	for msg := range sub.C() {
		ui.post(func() { ui.handleMessage(msg) })
	}
	err := ui.dev.Err()
	if err == nil {
		err = errors.New("device closed")
	}
	ui.post(func() { ui.setStatus(fmt.Errorf("connection lost: %w", err)) })
}

func (ui *UI) handleMessage(msg rfx.Message) {
	switch msg := msg.(type) {
	case *rfx.Sweep:
		ui.Waterfall.Update(msg)
		ui.Spectrum.Update(msg)
		ui.ChannelOverview.Update(msg)
	case *rfx.SweepConfig:
		ui.setConfig(*msg)
	}
}

func (ui *UI) setConfig(cfg rfx.SweepConfig) {
	ui.config = cfg
	ui.expModuleSwitch.Value = cfg.ExpansionActive
	ui.Spectrum.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.ChannelOverview.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.Waterfall.SetAmplitude(cfg.AmpBottom, cfg.AmpTop)
	ui.centerFrequencyInput.SetText(formatMHz(cfg.Center()))
	ui.frequencySpanInput.SetText(formatMHz(cfg.Span()))
}

// saveUIConfig queues the UI config to be written.
func (ui *UI) saveUIConfig() {
	if ui.saver == nil {
		return
	}
	if err := ui.saver.Save(ui.uiConfig); err != nil {
		ui.setStatus(err)
	}
}

// setOverlay shows the named band set, or none if name is empty.
func (ui *UI) setOverlay(name string) {
	ui.ChannelOverview.SetBandSet(bandplan.Find(ui.bandSets, name))
}

// formatMHz formats a frequency in MHz to the nearest kHz.
func formatMHz(f rfx.Frequency) string {
	return strconv.FormatFloat(f.MHz(), 'f', 3, 64)
}

// parseMHz parses a frequency entered in MHz.
func parseMHz(s string) (rfx.Frequency, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid frequency %q", s)
	}
	return rfx.Frequency(math.Round(v * float64(rfx.MHz))), nil
}

// refreshPresets reads the presets from the device. done, if set, runs
// after they have been shown.
func (ui *UI) refreshPresets(done func()) {
	var presets []rfx.Preset
	ui.do("read presets", func(ctx context.Context) error {
		var err error
		presets, err = ui.dev.Presets(ctx)
		return err
	}, func(err error) {
		if err != nil {
			return
		}
		ui.setPresets(presets)
		if done != nil {
			done()
		}
	})
}

// setPresets replaces the preset list with the used slots of presets. The
// first time, it also restores the overlay of the preset matching the
// device's configuration.
func (ui *UI) setPresets(presets []rfx.Preset) {
	used := make([]rfx.Preset, 0, len(presets))
	for _, p := range presets {
		if p.Name != "" {
			used = append(used, p)
		}
	}
	ui.presets = used
	ui.presetClicks = make([]gesture.Click, len(used))
	if !ui.presetsLoaded {
		ui.presetsLoaded = true
		if p, ok := matchPreset(used, ui.config); ok {
			ui.setOverlay(ui.uiConfig.BandSetFor(p.Index))
		}
	}
}

func (ui *UI) setSweep(start, stop rfx.Frequency) {
	req := rfx.SweepRequest{Start: start, Stop: stop, AmpTop: ui.config.AmpTop, AmpBottom: ui.config.AmpBottom}
	ui.do("set frequency range", func(ctx context.Context) error {
		_, err := ui.dev.SetSweep(ctx, req)
		return err
	}, nil)
}

// Run handles window events and renders the application until the window
// is closed.
func (ui *UI) Run(w *app.Window) error {
	ui.window = w
	defer close(ui.stop)

	sub := ui.dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep | rfx.KindConfig})
	defer sub.Close()
	go ui.receive(sub)
	go ui.runCommands()

	ui.setConfig(ui.dev.Config())
	ui.refreshPresets(nil)

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.FrameEvent:
			ui.drainUpdates()
			gtx := app.NewContext(&ops, e)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
			ui.handleEditor()
		case app.DestroyEvent:
			return e.Err
		}
	}
}

// drainUpdates runs the work other goroutines posted since the last frame.
func (ui *UI) drainUpdates() {
	for {
		select {
		case f := <-ui.updates:
			f()
		default:
			return
		}
	}
}

// handleEditor acts on the preset editor's buttons after a frame.
func (ui *UI) handleEditor() {
	pe := ui.presetEditor
	if pe.PendingCancel {
		pe.PendingCancel = false
		pe.Active = false
	}
	if pe.PendingApply {
		pe.PendingApply = false
		p := pe.Preset()
		module := rfx.ModuleExpansion
		if p.Mainboard {
			module = rfx.ModuleMain
		}
		ui.do("apply preset", func(ctx context.Context) error {
			if _, err := ui.dev.SelectModule(ctx, module); err != nil {
				return err
			}
			if _, err := ui.dev.SetSweep(ctx, rfx.SweepRequest{Start: p.Start, Stop: p.Stop, AmpTop: p.AmpTop, AmpBottom: p.AmpBottom}); err != nil {
				return err
			}
			return ui.dev.SetCalculator(ctx, p.Calculator)
		}, nil)
		ui.setOverlay(pe.SelectedBandSetName())
		if !pe.IsNew {
			ui.uiConfig.SetBandSetFor(p.Index, pe.SelectedBandSetName())
			ui.saveUIConfig()
		}
	}
	if pe.PendingSave {
		pe.PendingSave = false
		p := pe.Preset()
		bandSetName := pe.SelectedBandSetName()
		ui.do("save preset", func(ctx context.Context) error {
			return ui.dev.WritePreset(ctx, p)
		}, func(err error) {
			if err != nil {
				return
			}
			ui.uiConfig.SetBandSetFor(p.Index, bandSetName)
			ui.saveUIConfig()
			ui.setOverlay(bandSetName)
			ui.refreshPresets(func() { pe.Active = false })
		})
	}
	if pe.PendingDelete {
		pe.PendingDelete = false
		index := pe.Index
		ui.do("delete preset", func(ctx context.Context) error {
			return ui.dev.DeletePreset(ctx, index)
		}, func(err error) {
			if err != nil {
				return
			}
			ui.uiConfig.SetBandSetFor(index, "")
			ui.saveUIConfig()
			ui.refreshPresets(func() { pe.Active = false })
		})
	}
}

func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	// Esc closes the preset editor, or the window if it is not open.
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			if ui.presetEditor.Active {
				ui.presetEditor.Active = false
			} else {
				ui.window.Perform(system.ActionClose)
			}
		}
	}

	// Sync the channel selection highlight to the spectrum.
	if lo, hi, ok := ui.ChannelOverview.SelectedRange(); ok {
		ui.Spectrum.SetHighlight(lo, hi)
	} else {
		ui.Spectrum.ClearHighlight()
	}

	return ui.LayoutMain(ui.Theme, gtx)
}

// handleControls acts on the sidebar inputs.
func (ui *UI) handleControls(gtx layout.Context) {
	for {
		e, ok := ui.centerFrequencyInput.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			center, err := parseMHz(e.Text)
			if err != nil {
				ui.setStatus(err)
				continue
			}
			span := ui.config.Span()
			ui.setSweep(center-span/2, center+span/2)
		}
	}
	for {
		e, ok := ui.frequencySpanInput.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			span, err := parseMHz(e.Text)
			if err != nil {
				ui.setStatus(err)
				continue
			}
			center := ui.config.Center()
			ui.setSweep(center-span/2, center+span/2)
		}
	}
	if ui.lcdSwitch.Update(gtx) {
		on := ui.lcdSwitch.Value
		ui.do("set LCD", func(ctx context.Context) error { return ui.dev.SetLCD(ctx, on) }, nil)
	}
	if ui.expModuleSwitch.Update(gtx) {
		module := rfx.ModuleMain
		if ui.expModuleSwitch.Value {
			module = rfx.ModuleExpansion
		}
		ui.do("switch module", func(ctx context.Context) error {
			_, err := ui.dev.SelectModule(ctx, module)
			return err
		}, func(err error) {
			if err != nil {
				ui.expModuleSwitch.Value = ui.config.ExpansionActive
			}
		})
	}
	for i := range ui.presetClicks {
		for {
			e, ok := ui.presetClicks[i].Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				p := &ui.presets[i]
				ui.presetEditor.OpenPreset(p, ui.uiConfig.BandSetFor(p.Index))
			}
		}
	}
}

func (ui *UI) LayoutMain(th *material.Theme, gtx layout.Context) layout.Dimensions {
	paint.Fill(gtx.Ops, ui.Theme.Bg)
	ui.handleControls(gtx)

	freqLabel := func(f rfx.Frequency, align text.Alignment) layout.FlexChild {
		return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(ui.Theme, formatMHz(f))
			l.Color = ui.leftFg
			l.Alignment = align
			return l.Layout(gtx)
		})
	}

	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			pr := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
			paint.Fill(gtx.Ops, ui.leftBg)
			pr.Pop()
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.Spectrum.Layout(gtx, th)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.ChannelOverview.Layout(gtx, th)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						freqLabel(ui.config.Start, text.Start),
						freqLabel(ui.config.Center(), text.Middle),
						freqLabel(ui.config.Stop(), text.End))
				}),
				layout.Flexed(3, ui.Waterfall.Layout),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(240)
			gtx.Constraints.Min.X = gtx.Dp(240)
			return layout.UniformInset(defaultMargin).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.layoutSidebar(gtx, th)
			})
		}),
	)
}

func (ui *UI) layoutSidebar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	switchRow := func(sw *widget.Bool, label string) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(material.Switch(th, sw, label).Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(material.Body1(th, label).Layout),
			)
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
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
		layout.Rigid(switchRow(ui.lcdSwitch, "LCD On")),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(switchRow(ui.expModuleSwitch, "Expansion Module")),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if ui.presetEditor.Active {
				return ui.presetEditor.Layout(gtx, th)
			}
			return ui.layoutPresetList(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.status == "" {
				return layout.Dimensions{}
			}
			l := material.Caption(th, ui.status)
			if ui.statusErr {
				l.Color = errorColor
			}
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, l.Layout)
		}),
	)
}

func (ui *UI) layoutPresetList(gtx layout.Context, th *material.Theme) layout.Dimensions {
	next := firstUnusedPresetIndex(ui.presets, ui.maxPresets)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body2(th, "Presets").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(ui.Theme, ui.presetList).Layout(gtx, len(ui.presets), func(gtx layout.Context, index int) layout.Dimensions {
				bgColor := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				if ui.presetClicks[index].Hovered() {
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
			label := "Add Preset"
			if next < 0 {
				gtx = gtx.Disabled()
				label = "All preset slots used"
			} else if ui.addPresetBtn.Clicked(gtx) {
				ui.presetEditor.OpenNew(next, "")
			}
			return material.Button(th, &ui.addPresetBtn, label).Layout(gtx)
		}),
	)
}
