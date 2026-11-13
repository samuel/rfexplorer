package rfx

import (
	"fmt"
	"strings"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.52.0 -linecomment -type=Model,Mode,CalculatorMode,WifiMode,MarkerMode,Modulation,DSP,InputStage,Module,Kind -trimprefix=Kind -output=enum_string.go

// MaxSweepPoints is the largest number of points in a sweep.
const MaxSweepPoints = 65535

// Model identifies an RF Explorer board.
type Model int

const (
	Model433M           Model = 0   // 433M
	Model868M           Model = 1   // 868M
	Model915M           Model = 2   // 915M
	ModelWSUB1G         Model = 3   // WSUB1G
	Model24G            Model = 4   // 2.4G
	ModelWSUB3G         Model = 5   // WSUB3G
	Model6G             Model = 6   // 6G
	ModelWSUB1GPlus     Model = 10  // WSUB1G+
	ModelAudioPro       Model = 11  // Audio Pro
	Model2400Plus       Model = 12  // 2.4G+
	Model4GPlus         Model = 13  // 4G+
	Model6GPlus         Model = 14  // 6G+
	ModelW5G3G          Model = 16  // W5G3G
	ModelW5G4G          Model = 17  // W5G4G
	ModelW5G5G          Model = 18  // W5G5G
	ModelRFGen          Model = 60  // RFE6GEN
	ModelRFGenExpansion Model = 61  // RFE6GEN Expansion
	ModelNone           Model = 255 // None
	ModelInvalid        Model = -1  // Invalid
)

// IsGenerator reports whether m is a signal generator, which this package
// does not support.
func (m Model) IsGenerator() bool {
	return m == ModelRFGen || m == ModelRFGenExpansion
}

// IsPlus reports whether m is a Plus series board.
func (m Model) IsPlus() bool {
	switch m {
	case ModelWSUB1GPlus, ModelAudioPro, Model2400Plus, Model4GPlus, Model6GPlus, ModelW5G3G, ModelW5G4G, ModelW5G5G:
		return true
	}
	return false
}

// Mode is the operating mode of the device.
type Mode int

const (
	ModeSpectrumAnalyzer  Mode = 0   // SpectrumAnalyzer
	ModeRFGenerator       Mode = 1   // RFGenerator
	ModeWifiAnalyzer      Mode = 2   // WifiAnalyzer
	ModeAnalyzerTracking  Mode = 5   // AnalyzerTracking
	ModeRFSniffer         Mode = 6   // RFSniffer
	ModeCWTransmitter     Mode = 60  // CWTransmitter
	ModeSweepFrequency    Mode = 61  // SweepFrequency
	ModeSweepAmplitude    Mode = 62  // SweepAmplitude
	ModeGeneratorTracking Mode = 63  // GeneratorTracking
	ModeUnknown           Mode = 255 // Unknown
	ModeInvalid           Mode = -1  // Invalid
)

// CalculatorMode is how the device combines consecutive sweeps.
type CalculatorMode int

const (
	CalculatorNormal        CalculatorMode = 0  // Normal
	CalculatorMax           CalculatorMode = 1  // Max
	CalculatorAverage       CalculatorMode = 2  // Average
	CalculatorOverwrite     CalculatorMode = 3  // Overwrite
	CalculatorMaxHold       CalculatorMode = 4  // MaxHold
	CalculatorMaxHistorical CalculatorMode = 5  // MaxHistorical
	CalculatorInvalid       CalculatorMode = -1 // Invalid
)

func (m CalculatorMode) valid() bool {
	return m >= CalculatorNormal && m <= CalculatorMaxHistorical
}

// MarshalText implements encoding.TextMarshaler using the mode's name.
func (m CalculatorMode) MarshalText() ([]byte, error) {
	if !m.valid() {
		return nil, fmt.Errorf("rfx: invalid calculator mode %d", int(m))
	}
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. Names are case
// insensitive.
func (m *CalculatorMode) UnmarshalText(b []byte) error {
	for v := CalculatorNormal; v <= CalculatorMaxHistorical; v++ {
		if strings.EqualFold(v.String(), string(b)) {
			*m = v
			return nil
		}
	}
	return fmt.Errorf("rfx: unknown calculator mode %q", b)
}

// WifiMode selects the WiFi analyzer band.
type WifiMode byte

const (
	WifiDisabled WifiMode = 0 // Disabled
	Wifi24GHz    WifiMode = 1 // 2.4GHz
	Wifi5GHz     WifiMode = 2 // 5GHz
)

// MarkerMode is the marker shown on the device for a preset.
type MarkerMode byte

const (
	MarkerPeak   MarkerMode = 0 // Peak
	MarkerNone   MarkerMode = 1 // None
	MarkerManual MarkerMode = 2 // Manual
)

// MarshalText implements encoding.TextMarshaler using the mode's name.
func (m MarkerMode) MarshalText() ([]byte, error) {
	if m > MarkerManual {
		return nil, fmt.Errorf("rfx: invalid marker mode %d", m)
	}
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. Names are case
// insensitive.
func (m *MarkerMode) UnmarshalText(b []byte) error {
	for v := MarkerPeak; v <= MarkerManual; v++ {
		if strings.EqualFold(v.String(), string(b)) {
			*m = v
			return nil
		}
	}
	return fmt.Errorf("rfx: unknown marker mode %q", b)
}

// Modulation is the sniffer demodulation.
type Modulation int

const (
	ModulationOOKRaw Modulation = 0    // OOK Raw
	ModulationPSKRaw Modulation = 1    // PSK Raw
	ModulationOOKStd Modulation = 2    // OOK Std
	ModulationPSKStd Modulation = 3    // PSK Std
	ModulationNone   Modulation = 0xff // None
)

// DSP is the onboard signal processing mode.
type DSP byte

const (
	DSPAuto   DSP = 0 // Auto
	DSPFilter DSP = 1 // Filter
	DSPFast   DSP = 2 // Fast
	DSPNoImg  DSP = 3 // NoImg
)

// InputStage is the RF input path on models that have a selectable one.
type InputStage byte

const (
	InputStageDirect         InputStage = 0 // Direct
	InputStageAttenuator30dB InputStage = 1 // Attenuator 30dB
	InputStageLNA25dB        InputStage = 2 // LNA 25dB
	InputStageAttenuator60dB InputStage = 3 // Attenuator 60dB
	InputStageLNA12dB        InputStage = 4 // LNA 12dB
)

// Module selects one of the two RF boards in the device.
type Module byte

const (
	ModuleMain      Module = 0 // Main
	ModuleExpansion Module = 1 // Expansion
)

// BaudRate is the serial communications baud rate configured on the RF Explorer.
type BaudRate int

// Supported baud rates.
const (
	BaudRate1200   BaudRate = 1200
	BaudRate2400   BaudRate = 2400
	BaudRate4800   BaudRate = 4800
	BaudRate9600   BaudRate = 9600
	BaudRate19200  BaudRate = 19200
	BaudRate38400  BaudRate = 38400
	BaudRate57600  BaudRate = 57600
	BaudRate115200 BaudRate = 115200
	BaudRate500000 BaudRate = 500000
)

// DefaultBaudRate is the baud rate RF Explorer uses unless reconfigured.
const DefaultBaudRate = BaudRate500000
