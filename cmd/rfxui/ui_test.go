package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"samuelks.com/rfx"
)

func TestConfigSaver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "ui.json")
	var errs []error
	s := newConfigSaver(path, func(err error) { errs = append(errs, err) })
	cfg := &UIConfig{}
	for i := range 20 {
		cfg.SetBandSetFor(i, "set")
		if err := s.Save(cfg); err != nil {
			t.Fatal(err)
		}
	}
	// Changes after Save must not affect what is written.
	cfg.SetBandSetFor(99, "late")
	s.Close()
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	got, err := loadUIConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PresetOverlays) != 20 || got.BandSetFor(19) != "set" || got.BandSetFor(99) != "" {
		t.Fatalf("saved %+v", got.PresetOverlays)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}

func TestMatchPreset(t *testing.T) {
	presets := []rfx.Preset{
		{Index: 0, Start: 2400 * rfx.MHz, Stop: 2500 * rfx.MHz, Mainboard: true},
		{Index: 1, Start: 2400 * rfx.MHz, Stop: 2500 * rfx.MHz, Mainboard: false},
		{Index: 2, Start: 433 * rfx.MHz, Stop: 435 * rfx.MHz, Mainboard: false},
	}
	// 112 points from 2400 MHz with the stop rounded down to a whole step.
	cfg := rfx.SweepConfig{Start: 2400 * rfx.MHz, Step: 900_900, Points: 112, ExpansionActive: true}
	p, ok := matchPreset(presets, cfg)
	if !ok || p.Index != 1 {
		t.Fatalf("matched %v, %v; want preset 1", p.Index, ok)
	}
	cfg.ExpansionActive = false
	if p, ok := matchPreset(presets, cfg); !ok || p.Index != 0 {
		t.Fatalf("matched %v, %v; want preset 0", p.Index, ok)
	}
	cfg.Start += 10 * rfx.MHz
	if _, ok := matchPreset(presets, cfg); ok {
		t.Fatal("matched a preset with a different range")
	}
}

func TestFirstUnusedPresetIndex(t *testing.T) {
	presets := []rfx.Preset{{Index: 0}, {Index: 2}}
	if got := firstUnusedPresetIndex(presets, 3); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := firstUnusedPresetIndex(append(presets, rfx.Preset{Index: 1}), 3); got != -1 {
		t.Fatalf("got %d with all slots used, want -1", got)
	}
}

func TestPresetEditorCheck(t *testing.T) {
	e := newPresetEditor(30)
	e.OpenPreset(&rfx.Preset{
		Index: 3, Name: "WiFi", Start: 2400 * rfx.MHz, Stop: 2483_500 * rfx.KHz,
		AmpTop: 0, AmpBottom: -100, Iterations: 4, Calculator: rfx.CalculatorMaxHold,
		Marker: rfx.MarkerMode(200), Mainboard: true,
	}, "")
	if e.markerModeDropdown.Selected != 0 {
		t.Fatalf("out of range marker mode selected %d", e.markerModeDropdown.Selected)
	}
	if got := e.minFreqInput.Text(); got != "2400.000" {
		t.Fatalf("start shown as %q, want MHz", got)
	}
	f := e.check()
	if f.err != nil || !f.canApply {
		t.Fatalf("valid preset: err %v, canApply %v", f.err, f.canApply)
	}
	if f.preset.Stop != 2483_500*rfx.KHz || f.preset.Iterations != 4 || f.preset.Calculator != rfx.CalculatorMaxHold {
		t.Fatalf("preset %+v", f.preset)
	}

	e.maxFreqInput.SetText("24x")
	f = e.check()
	if f.fieldErrs[e.maxFreqInput] == "" || f.err == nil || f.canApply {
		t.Fatalf("bad frequency: %+v", f)
	}

	e.maxFreqInput.SetText("2483.5")
	e.nameInput.SetText("")
	f = e.check()
	if f.err == nil || !strings.Contains(f.err.Error(), "name") || !f.canApply {
		t.Fatalf("empty name: err %v, canApply %v", f.err, f.canApply)
	}

	e.nameInput.SetText("WiFi")
	e.ampBottomInput.SetText("-5")
	if f := e.check(); f.err == nil || f.canApply {
		t.Fatalf("amplitude range too small: err %v, canApply %v", f.err, f.canApply)
	}

	e.ampBottomInput.SetText("-100")
	e.Index = 30
	if f := e.check(); f.err == nil || !f.canApply {
		t.Fatalf("slot beyond device: err %v, canApply %v", f.err, f.canApply)
	}
}
