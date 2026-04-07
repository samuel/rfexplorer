package main

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/samuel/rfexplorer/rfx"
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
	if d.header.Clicked(gtx) {
		d.open = !d.open
	}
	for i := range d.items {
		if d.items[i].Clicked(gtx) {
			d.Selected = i
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
			i := i
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

	nameInput           *widget.Editor
	minFreqInput        *widget.Editor
	maxFreqInput        *widget.Editor
	ampTopInput         *widget.Editor
	ampBottomInput      *widget.Editor
	calcIterationsInput *widget.Editor
	mainboardSwitch     *widget.Bool
	calcModeDropdown    *enumDropdown
	markerModeDropdown  *enumDropdown

	applyBtn  widget.Clickable
	saveBtn   widget.Clickable
	deleteBtn widget.Clickable
	cancelBtn widget.Clickable

	PendingApply   bool
	PendingSave    bool
	PendingDelete  bool
	PendingCancel  bool
	CloseOnRefresh bool

	list widget.List
}

func newPresetEditor() *presetEditor {
	return &presetEditor{
		nameInput:           &widget.Editor{SingleLine: true, Submit: true},
		minFreqInput:        &widget.Editor{SingleLine: true, Submit: true},
		maxFreqInput:        &widget.Editor{SingleLine: true, Submit: true},
		ampTopInput:         &widget.Editor{SingleLine: true, Submit: true},
		ampBottomInput:      &widget.Editor{SingleLine: true, Submit: true},
		calcIterationsInput: &widget.Editor{SingleLine: true, Submit: true},
		mainboardSwitch:     &widget.Bool{},
		calcModeDropdown:    newEnumDropdown(calcModeLabels, 0),
		markerModeDropdown:  newEnumDropdown(markerModeLabels, 0),
		list:                widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

func calcModeIndex(m rfx.CalculatorMode) int {
	if m < 0 || int(m) >= len(calcModeLabels) {
		return 0
	}
	return int(m)
}

func (e *presetEditor) OpenPreset(p *rfx.Preset) {
	e.Active = true
	e.IsNew = false
	e.Index = p.Index
	e.nameInput.SetText(p.Name)
	e.minFreqInput.SetText(strconv.Itoa(p.MinFreqKHz))
	e.maxFreqInput.SetText(strconv.Itoa(p.MaxFreqKHz))
	e.ampTopInput.SetText(strconv.Itoa(p.AmpTopDBm))
	e.ampBottomInput.SetText(strconv.Itoa(p.AmpBottomDBm))
	e.calcIterationsInput.SetText(strconv.Itoa(p.CalcIterations))
	e.mainboardSwitch.Value = p.Mainboard
	e.calcModeDropdown.Selected = calcModeIndex(p.CalcMode)
	e.calcModeDropdown.open = false
	e.markerModeDropdown.Selected = int(p.MarkerMode)
	e.markerModeDropdown.open = false
}

func (e *presetEditor) OpenNew(index int) {
	e.Active = true
	e.IsNew = true
	e.Index = index
	e.nameInput.SetText("")
	e.minFreqInput.SetText("0")
	e.maxFreqInput.SetText("0")
	e.ampTopInput.SetText("0")
	e.ampBottomInput.SetText("-100")
	e.calcIterationsInput.SetText("1")
	e.mainboardSwitch.Value = true
	e.calcModeDropdown.Selected = 0
	e.calcModeDropdown.open = false
	e.markerModeDropdown.Selected = 0
	e.markerModeDropdown.open = false
}

// ToPreset reads the current form state into a Preset value.
func (e *presetEditor) ToPreset() rfx.Preset {
	p := rfx.Preset{Index: e.Index}
	p.Name = e.nameInput.Text()
	p.MinFreqKHz, _ = strconv.Atoi(e.minFreqInput.Text())
	p.MaxFreqKHz, _ = strconv.Atoi(e.maxFreqInput.Text())
	p.AmpTopDBm, _ = strconv.Atoi(e.ampTopInput.Text())
	p.AmpBottomDBm, _ = strconv.Atoi(e.ampBottomInput.Text())
	p.CalcIterations, _ = strconv.Atoi(e.calcIterationsInput.Text())
	p.Mainboard = e.mainboardSwitch.Value
	p.CalcMode = rfx.CalculatorMode(e.calcModeDropdown.Selected)
	p.MarkerMode = rfx.MarkerMode(e.markerModeDropdown.Selected)
	return p
}

// Layout renders the editor form. After each call, check the Pending* flags
// and clear them once the corresponding action has been taken.
func (e *presetEditor) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if e.applyBtn.Clicked(gtx) {
		e.PendingApply = true
	}
	if e.saveBtn.Clicked(gtx) {
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
	}

	rows := []formRow{
		{"", func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(th, title)
			l.Font.Weight = 700
			return l.Layout(gtx)
		}},
		{"Name", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.nameInput)
		}},
		{"Min Freq (kHz)", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.minFreqInput)
		}},
		{"Max Freq (kHz)", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.maxFreqInput)
		}},
		{"Amp Top (dBm)", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.ampTopInput)
		}},
		{"Amp Bottom (dBm)", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.ampBottomInput)
		}},
		{"Calc Iterations", func(gtx layout.Context) layout.Dimensions {
			return layoutEditorField(gtx, th, e.calcIterationsInput)
		}},
		{"Calc Mode", func(gtx layout.Context) layout.Dimensions {
			return e.calcModeDropdown.Layout(gtx, th)
		}},
		{"Marker Mode", func(gtx layout.Context) layout.Dimensions {
			return e.markerModeDropdown.Layout(gtx, th)
		}},
		{"", func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(material.Switch(th, e.mainboardSwitch, "Mainboard").Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(material.Body1(th, "Mainboard").Layout),
			)
		}},
	}

	// Total list items: field rows + 2 button rows.
	const buttonRows = 2
	total := len(rows) + buttonRows

	return material.List(th, &e.list).Layout(gtx, total, func(gtx layout.Context, index int) layout.Dimensions {
		switch index {
		case total - 2: // Apply + Save
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, material.Button(th, &e.applyBtn, "Apply").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, material.Button(th, &e.saveBtn, "Save").Layout),
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
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		)
	})
}