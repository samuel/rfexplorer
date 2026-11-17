package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"samuelks.com/bandplan"
	"samuelks.com/rfx"
)

var (
	calcModeLabels = []string{
		"Normal", "Max", "Average", "Overwrite", "Max Hold", "Max Historical",
	}
	markerModeLabels = []string{"Peak", "None", "Manual"}
)

// enumDropdown is a simple inline expandable enum selector.
type enumDropdown struct {
	labels   []string
	Selected int
	Changed  bool
	open     bool
	header   widget.Clickable
	items    []widget.Clickable
}

func newEnumDropdown(labels []string, selected int) *enumDropdown {
	return &enumDropdown{
		labels:   labels,
		Selected: selected,
		items:    make([]widget.Clickable, len(labels)),
	}
}

func (d *enumDropdown) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.Changed = false
	if d.header.Clicked(gtx) {
		d.open = !d.open
	}
	for i := range d.items {
		if d.items[i].Clicked(gtx) {
			if d.Selected != i {
				d.Selected = i
				d.Changed = true
			}
			d.open = false
		}
	}

	borderColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}

	var children []layout.FlexChild

	// Header button showing current selection.
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return d.header.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
					arrow := " ▾"
					if d.open {
						arrow = " ▴"
					}
					return layout.UniformInset(unit.Dp(6)).Layout(gtx,
						material.Body1(th, d.labels[d.Selected]+arrow).Layout)
				}),
			)
		})
	}))

	// Option rows shown when open.
	if d.open {
		for i := range d.items {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return d.items[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					bg := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					switch {
					case i == d.Selected:
						bg = color.NRGBA{R: 0xd0, G: 0xdc, B: 0xff, A: 0xff}
					case d.items[i].Hovered():
						bg = color.NRGBA{R: 0xe8, G: 0xec, B: 0xff, A: 0xff}
					}
					return layout.Stack{}.Layout(gtx,
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							paint.FillShape(gtx.Ops, bg,
								clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Op())
							return layout.Dimensions{Size: gtx.Constraints.Min}
						}),
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Inset{
								Left: unit.Dp(16), Top: unit.Dp(6),
								Bottom: unit.Dp(6), Right: unit.Dp(6),
							}.Layout(gtx, material.Body1(th, d.labels[i]).Layout)
						}),
					)
				})
			}))
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// presetEditor is the form for creating and editing presets.
// Callers must check the Pending* flags after Layout and handle them.
type presetEditor struct {
	Active bool
	IsNew  bool
	Index  int
	// MaxPresets is the number of preset slots on the device.
	MaxPresets int

	nameInput           *widget.Editor
	minFreqInput        *widget.Editor
	maxFreqInput        *widget.Editor
	ampTopInput         *widget.Editor
	ampBottomInput      *widget.Editor
	calcIterationsInput *widget.Editor
	mainboardSwitch     *widget.Bool
	calcModeDropdown    *enumDropdown
	markerModeDropdown  *enumDropdown
	bandSets            []*bandplan.BandSet
	bandSetDropdown     *enumDropdown

	applyBtn  widget.Clickable
	saveBtn   widget.Clickable
	deleteBtn widget.Clickable
	cancelBtn widget.Clickable

	PendingApply  bool
	PendingSave   bool
	PendingDelete bool
	PendingCancel bool

	list widget.List
}

func newPresetEditor(maxPresets int) *presetEditor {
	return &presetEditor{
		MaxPresets:          maxPresets,
		nameInput:           &widget.Editor{SingleLine: true, Submit: true},
		minFreqInput:        &widget.Editor{SingleLine: true, Submit: true},
		maxFreqInput:        &widget.Editor{SingleLine: true, Submit: true},
		ampTopInput:         &widget.Editor{SingleLine: true, Submit: true},
		ampBottomInput:      &widget.Editor{SingleLine: true, Submit: true},
		calcIterationsInput: &widget.Editor{SingleLine: true, Submit: true},
		mainboardSwitch:     &widget.Bool{},
		calcModeDropdown:    newEnumDropdown(calcModeLabels, 0),
		markerModeDropdown:  newEnumDropdown(markerModeLabels, 0),
		bandSetDropdown:     newEnumDropdown([]string{"(none)"}, 0),
		list:                widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

// SetBandSets updates the available band sets shown in the channel overlay
// dropdown. Call this once after loading band sets at startup.
func (e *presetEditor) SetBandSets(sets []*bandplan.BandSet) {
	labels := make([]string, 1+len(sets))
	labels[0] = "(none)"
	for i, bs := range sets {
		labels[i+1] = bs.Name
	}
	e.bandSets = sets
	e.bandSetDropdown = newEnumDropdown(labels, 0)
}

// SelectedBandSetName returns the name of the currently selected BandSet, or
// "" if none is selected.
func (e *presetEditor) SelectedBandSetName() string {
	if e.bandSetDropdown.Selected == 0 {
		return ""
	}
	return e.bandSets[e.bandSetDropdown.Selected-1].Name
}

func (e *presetEditor) setBandSetByName(name string) {
	for i, bs := range e.bandSets {
		if bs.Name == name {
			e.bandSetDropdown.Selected = i + 1
			return
		}
	}
	e.bandSetDropdown.Selected = 0
}

func calcModeIndex(m rfx.CalculatorMode) int {
	if m < 0 || int(m) >= len(calcModeLabels) {
		return 0
	}
	return int(m)
}

func markerModeIndex(m rfx.MarkerMode) int {
	if int(m) >= len(markerModeLabels) {
		return 0
	}
	return int(m)
}

func (e *presetEditor) OpenPreset(p *rfx.Preset, bandSetName string) {
	e.Active = true
	e.IsNew = false
	e.Index = p.Index
	e.nameInput.SetText(p.Name)
	e.minFreqInput.SetText(formatMHz(p.Start))
	e.maxFreqInput.SetText(formatMHz(p.Stop))
	e.ampTopInput.SetText(strconv.Itoa(p.AmpTop))
	e.ampBottomInput.SetText(strconv.Itoa(p.AmpBottom))
	e.calcIterationsInput.SetText(strconv.Itoa(p.Iterations))
	e.mainboardSwitch.Value = p.Mainboard
	e.calcModeDropdown.Selected = calcModeIndex(p.Calculator)
	e.calcModeDropdown.open = false
	e.markerModeDropdown.Selected = markerModeIndex(p.Marker)
	e.markerModeDropdown.open = false
	e.bandSetDropdown.open = false
	e.setBandSetByName(bandSetName)
}

func (e *presetEditor) OpenNew(index int, bandSetName string) {
	e.Active = true
	e.IsNew = true
	e.Index = index
	e.nameInput.SetText("")
	e.minFreqInput.SetText("")
	e.maxFreqInput.SetText("")
	e.ampTopInput.SetText("0")
	e.ampBottomInput.SetText("-100")
	e.calcIterationsInput.SetText("1")
	e.mainboardSwitch.Value = true
	e.calcModeDropdown.Selected = 0
	e.calcModeDropdown.open = false
	e.markerModeDropdown.Selected = 0
	e.markerModeDropdown.open = false
	e.bandSetDropdown.open = false
	e.setBandSetByName(bandSetName)
}

// presetForm is the result of checking the editor fields.
type presetForm struct {
	preset rfx.Preset
	// fieldErrs holds a message for each field that could not be parsed.
	fieldErrs map[*widget.Editor]string
	// err is the first problem with the preset as a whole, or nil.
	err error
	// canApply is true if the range and amplitude can be sent to the device,
	// even if the preset can't be saved.
	canApply bool
}

// check parses the form into a preset and validates it.
func (e *presetEditor) check() presetForm {
	f := presetForm{fieldErrs: map[*widget.Editor]string{}}
	p := rfx.Preset{
		Index:      e.Index,
		Name:       e.nameInput.Text(),
		Mainboard:  e.mainboardSwitch.Value,
		Calculator: rfx.CalculatorMode(e.calcModeDropdown.Selected),
		Marker:     rfx.MarkerMode(e.markerModeDropdown.Selected),
	}
	freq := func(ed *widget.Editor) rfx.Frequency {
		v, err := parseMHz(ed.Text())
		if err != nil {
			f.fieldErrs[ed] = "Enter a frequency in MHz"
			return 0
		}
		// The device stores whole kHz.
		return (v + rfx.KHz/2) / rfx.KHz * rfx.KHz
	}
	integer := func(ed *widget.Editor) int {
		v, err := strconv.Atoi(strings.TrimSpace(ed.Text()))
		if err != nil {
			f.fieldErrs[ed] = "Enter a whole number"
		}
		return v
	}
	p.Start = freq(e.minFreqInput)
	p.Stop = freq(e.maxFreqInput)
	p.AmpTop = integer(e.ampTopInput)
	p.AmpBottom = integer(e.ampBottomInput)
	p.Iterations = integer(e.calcIterationsInput)
	f.preset = p

	if len(f.fieldErrs) > 0 {
		f.err = errors.New("fix the highlighted fields")
	} else if err := p.Validate(); err != nil {
		f.err = err
	} else if p.Index >= e.MaxPresets {
		f.err = fmt.Errorf("preset slot %d is beyond the device's %d slots", p.Index+1, e.MaxPresets)
	}

	// Applying uses only the range and amplitude, so check those with the
	// other fields set to valid values.
	q := p
	q.Index, q.Name, q.Iterations = 0, "x", 1
	_, freqErr := f.fieldErrs[e.minFreqInput]
	_, freqErr2 := f.fieldErrs[e.maxFreqInput]
	_, ampErr := f.fieldErrs[e.ampTopInput]
	_, ampErr2 := f.fieldErrs[e.ampBottomInput]
	f.canApply = !freqErr && !freqErr2 && !ampErr && !ampErr2 && q.Validate() == nil
	return f
}

// Preset returns the preset described by the form. It is valid only if
// check reports no error.
func (e *presetEditor) Preset() rfx.Preset {
	return e.check().preset
}

// Layout renders the editor form. After each call, check the Pending* flags
// and clear them once the corresponding action has been taken.
func (e *presetEditor) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	form := e.check()
	canSave := form.err == nil
	if form.canApply && e.applyBtn.Clicked(gtx) {
		e.PendingApply = true
	}
	if canSave && e.saveBtn.Clicked(gtx) {
		e.PendingSave = true
	}
	if e.deleteBtn.Clicked(gtx) {
		e.PendingDelete = true
	}
	if e.cancelBtn.Clicked(gtx) {
		e.PendingCancel = true
	}

	title := "Edit Preset"
	if e.IsNew {
		title = "New Preset"
	}

	type formRow struct {
		label  string
		widget func(gtx layout.Context) layout.Dimensions
		// editor, if set, has its parse error shown below the row.
		editor *widget.Editor
	}
	field := func(label string, ed *widget.Editor) formRow {
		return formRow{label, func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, ed)
		}, ed}
	}

	rows := []formRow{
		{"", func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(th, title)
			l.Font.Weight = 700
			return l.Layout(gtx)
		}, nil},
		{"", func(gtx layout.Context) layout.Dimensions {
			return material.Caption(th, fmt.Sprintf("Slot %d of %d", e.Index+1, e.MaxPresets)).Layout(gtx)
		}, nil},
		field("Name", e.nameInput),
		field("Min Freq (MHz)", e.minFreqInput),
		field("Max Freq (MHz)", e.maxFreqInput),
		field("Amp Top (dBm)", e.ampTopInput),
		field("Amp Bottom (dBm)", e.ampBottomInput),
		field("Calc Iterations (1-16)", e.calcIterationsInput),
		{"Calc Mode", func(gtx layout.Context) layout.Dimensions {
			return e.calcModeDropdown.Layout(gtx, th)
		}, nil},
		{"Marker Mode", func(gtx layout.Context) layout.Dimensions {
			return e.markerModeDropdown.Layout(gtx, th)
		}, nil},
		{"Channel Overlay", func(gtx layout.Context) layout.Dimensions {
			dims := e.bandSetDropdown.Layout(gtx, th)
			if e.bandSetDropdown.Changed && e.bandSetDropdown.Selected > 0 {
				bs := e.bandSets[e.bandSetDropdown.Selected-1]
				lo, hi := bs.Span()
				e.minFreqInput.SetText(formatMHz(rfx.Frequency(lo)))
				e.maxFreqInput.SetText(formatMHz(rfx.Frequency(hi)))
			}
			return dims
		}, nil},
		{"", func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(material.Switch(th, e.mainboardSwitch, "Mainboard").Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(material.Body1(th, "Mainboard").Layout),
			)
		}, nil},
	}

	errText := func(gtx layout.Context, msg string) layout.Dimensions {
		l := material.Caption(th, msg)
		l.Color = errorColor
		return l.Layout(gtx)
	}
	enabled := func(ok bool, w layout.Widget) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			if !ok {
				gtx = gtx.Disabled()
			}
			return w(gtx)
		}
	}

	// Total list items: field rows, the form error and 2 button rows.
	const extraRows = 3
	total := len(rows) + extraRows

	return material.List(th, &e.list).Layout(gtx, total, func(gtx layout.Context, index int) layout.Dimensions {
		switch index {
		case total - 3: // Form error
			if form.err == nil || len(form.fieldErrs) > 0 {
				return layout.Dimensions{}
			}
			return errText(gtx, strings.TrimPrefix(form.err.Error(), "rfx: "))
		case total - 2: // Apply + Save
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, enabled(form.canApply, material.Button(th, &e.applyBtn, "Apply").Layout)),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, enabled(canSave, material.Button(th, &e.saveBtn, "Save").Layout)),
				)
			})
		case total - 1: // Delete (existing only) + Cancel
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if e.IsNew {
							return layout.Dimensions{}
						}
						return material.Button(th, &e.deleteBtn, "Delete").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, material.Button(th, &e.cancelBtn, "Cancel").Layout),
				)
			})
		}

		r := rows[index]
		if r.label == "" {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, r.widget)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.Body2(th, r.label).Layout),
			layout.Rigid(r.widget),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if msg, ok := form.fieldErrs[r.editor]; ok && r.editor != nil {
					return errText(gtx, msg)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		)
	})
}
