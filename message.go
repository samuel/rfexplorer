package rfx

import (
	"fmt"
	"image"
	"image/color"
	"time"
)

// Kind identifies the type of a Message. Kinds combine as a bit set to select
// messages in SubscribeOptions.
type Kind uint32

const (
	KindSweep Kind = 1 << iota
	KindConfig
	KindDeviceInfo
	KindSerialNumber
	KindCalculator
	KindPreset
	KindPresetsEnd
	KindScreen
	KindCalibration
	KindCalibrationAvailability
	KindSnifferConfig
	KindRawData
	KindUnhandled
	KindProtocolError
	KindInputStage
	KindTracking

	// KindAll selects every message.
	KindAll Kind = 1<<iota - 1
)

// Message is something received from the device. The concrete types are the
// pointer types in this package that have a Kind method.
type Message interface {
	Kind() Kind
	message()
}

// Sweep is one pass over the configured frequency range.
type Sweep struct {
	// Time is when the sweep was received.
	Time time.Time
	// Start is the frequency of the first sample.
	Start Frequency
	// Step is the frequency distance between samples.
	Step Frequency
	// Samples are the measured levels in dBm.
	Samples []float32
}

// FrequencyAt returns the frequency of sample i.
func (s *Sweep) FrequencyAt(i int) Frequency {
	return s.Start + Frequency(i)*s.Step
}

// IndexOf returns the index of the sample nearest to f. ok is false if f is
// outside the sweep.
func (s *Sweep) IndexOf(f Frequency) (i int, ok bool) {
	if s.Step <= 0 || len(s.Samples) == 0 {
		return 0, false
	}
	i = int((f - s.Start + s.Step/2) / s.Step)
	if f < s.Start-s.Step/2 || i >= len(s.Samples) {
		return 0, false
	}
	return i, true
}

// Peak returns the index and level of the strongest sample, or -1 if the
// sweep is empty.
func (s *Sweep) Peak() (i int, dBm float32) {
	i = -1
	for j, v := range s.Samples {
		if i < 0 || v > dBm {
			i, dBm = j, v
		}
	}
	return i, dBm
}

// SweepConfig is the analyzer configuration reported by the device.
type SweepConfig struct {
	// Start is the frequency of the first sweep point.
	Start Frequency
	// Step is the frequency distance between sweep points.
	Step Frequency
	// Points is the number of points in a sweep.
	Points int
	// AmpTop and AmpBottom are the displayed amplitude range in dBm.
	AmpTop    int
	AmpBottom int
	// ExpansionActive is true if the expansion board is selected.
	ExpansionActive bool
	Mode            Mode
	// MinFreq and MaxFreq are the limits of the active board.
	MinFreq Frequency
	MaxFreq Frequency
	// MaxSpan is the widest range the active board can sweep.
	MaxSpan Frequency
	// RBW is the resolution bandwidth.
	RBW Frequency
	// AmpOffset is the amplitude offset in dB configured on the device.
	AmpOffset  int
	Calculator CalculatorMode
}

// Stop returns the frequency of the last sweep point.
func (c SweepConfig) Stop() Frequency {
	return c.Start + c.Span()
}

// Span returns the frequency range covered by a sweep.
func (c SweepConfig) Span() Frequency {
	if c.Points < 2 {
		return 0
	}
	return c.Step * Frequency(c.Points-1)
}

// Center returns the middle of the swept range.
func (c SweepConfig) Center() Frequency {
	return c.Start + c.Span()/2
}

// Module returns the selected RF board.
func (c SweepConfig) Module() Module {
	if c.ExpansionActive {
		return ModuleExpansion
	}
	return ModuleMain
}

// DeviceInfo describes the connected device.
type DeviceInfo struct {
	// Main and Expansion are the installed RF boards. Expansion is ModelNone
	// if there is none.
	Main      Model
	Expansion Model
	Firmware  string
	// Serial is empty until the device has reported it.
	Serial string
}

// MaxPresets returns the number of preset slots on the device.
func (i DeviceInfo) MaxPresets() int {
	if i.Main.IsPlus() {
		return 100
	}
	return 30
}

// isMWSUB3G reports whether the device is a WSUB3G without an expansion
// board, which has features the WSUB3G expansion board lacks.
func (i DeviceInfo) isMWSUB3G() bool {
	return i.Main == ModelWSUB3G && i.Expansion == ModelNone
}

// SerialNumber is the device serial number.
type SerialNumber struct {
	Serial string
}

// CalculatorChange reports the calculator mode after it was set.
type CalculatorChange struct {
	Mode CalculatorMode
}

// Preset is a stored analyzer configuration. Its JSON form uses frequency
// strings such as "433.92MHz" and mode names such as "MaxHold".
type Preset struct {
	// Index is the slot of the preset starting at 0, shown as 1 on the device.
	Index int `json:"index"`
	// Name is up to 12 printable 7-bit ASCII characters.
	Name  string    `json:"name"`
	Start Frequency `json:"start"`
	Stop  Frequency `json:"stop"`
	// AmpTop range is [-110, +35] dBm and at least 10 above AmpBottom.
	AmpTop int `json:"amp_top"`
	// AmpBottom range is [-120, +25] dBm.
	AmpBottom  int            `json:"amp_bottom"`
	Calculator CalculatorMode `json:"calculator"`
	// Iterations range is [1, 16].
	Iterations int `json:"iterations"`
	// Mainboard selects the main board rather than the expansion board.
	Mainboard bool       `json:"mainboard"`
	Marker    MarkerMode `json:"marker"`
}

// PresetsEnd marks the end of a preset listing and acknowledges a preset
// write.
type PresetsEnd struct{}

// Calibration is the embedded amplitude calibration table as sent by the
// device. See Device.Calibration for the offsets of the active board.
type Calibration struct {
	Data []byte
}

// CalibrationAvailability reports which boards have embedded calibration.
type CalibrationAvailability struct {
	Mainboard bool
	Expansion bool
}

// SnifferConfig is the configuration reported in sniffer mode.
type SnifferConfig struct {
	Start           Frequency
	ExpansionActive bool
	Mode            Mode
	// Delay sets the sample rate: 16 MHz / Delay.
	Delay        int
	Modulation   Modulation
	RBW          Frequency
	ThresholdDBm float64
}

// RawData is a sniffer capture: the demodulated signal sampled at the rate
// set by SnifferConfig.Delay, one bit per sample. For OOK a set bit means the
// signal was above the threshold; for PSK it is the demodulated phase.
type RawData struct {
	// Time is when the capture was received.
	Time time.Time
	Data []byte
}

// Len returns the number of samples in the capture.
func (r *RawData) Len() int {
	return len(r.Data) * 8
}

// Bit returns sample i. Samples are stored most significant bit first. Bit
// order has not been verified against hardware.
func (r *RawData) Bit(i int) bool {
	return r.Data[i/8]&(0x80>>(i%8)) != 0
}

// Pulse is a run of equal samples in a capture.
type Pulse struct {
	High    bool
	Samples int
}

// Pulses returns the capture as runs of equal samples, the usual starting
// point for decoding OOK remotes and sensors.
func (r *RawData) Pulses() []Pulse {
	var pulses []Pulse
	for i := range r.Len() {
		b := r.Bit(i)
		if n := len(pulses); n > 0 && pulses[n-1].High == b {
			pulses[n-1].Samples++
			continue
		}
		pulses = append(pulses, Pulse{High: b, Samples: 1})
	}
	return pulses
}

// InputStageChange reports the selected input stage ("#QA").
type InputStageChange struct {
	Stage InputStage
}

// TrackingChange reports that the device entered ("#K1") or left ("#K0")
// tracking mode.
type TrackingChange struct {
	Enabled bool
}

// Unhandled is a frame the library doesn't interpret, or line noise.
type Unhandled struct {
	Data []byte
}

// ProtocolError reports a frame that was recognized but malformed.
type ProtocolError struct {
	// Frame is the frame without its terminator.
	Frame []byte
	Err   error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("rfx: invalid %q frame: %s", e.Frame[:min(len(e.Frame), 2)], e.Err)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}

// ScreenImage is an image of the LCD screen sent by the device. It implements
// image.Image.
type ScreenImage struct {
	Data []byte
}

// ColorModel returns the Image's color model.
func (si *ScreenImage) ColorModel() color.Model {
	return color.GrayModel
}

// Bounds returns the size of the LCD.
func (si *ScreenImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, 128, 64)
}

// At returns the color of the pixel at (x, y).
func (si *ScreenImage) At(x, y int) color.Color {
	return si.AtGray(x, y)
}

// AtGray returns the color.Gray of the pixel at (x, y). Points outside the
// bounds are black and pixels missing from Data are unlit.
func (si *ScreenImage) AtGray(x, y int) color.Gray {
	if !(image.Point{X: x, Y: y}).In(si.Bounds()) {
		return color.Gray{}
	}
	i := (y/8)*128 + x
	if i >= len(si.Data) {
		return color.Gray{Y: 255}
	}
	return color.Gray{Y: 255 ^ (255 * ((si.Data[i] >> (uint(y) % 8)) & 1))}
}

func (*Sweep) Kind() Kind                   { return KindSweep }
func (*SweepConfig) Kind() Kind             { return KindConfig }
func (*DeviceInfo) Kind() Kind              { return KindDeviceInfo }
func (*SerialNumber) Kind() Kind            { return KindSerialNumber }
func (*CalculatorChange) Kind() Kind        { return KindCalculator }
func (*Preset) Kind() Kind                  { return KindPreset }
func (*PresetsEnd) Kind() Kind              { return KindPresetsEnd }
func (*ScreenImage) Kind() Kind             { return KindScreen }
func (*Calibration) Kind() Kind             { return KindCalibration }
func (*CalibrationAvailability) Kind() Kind { return KindCalibrationAvailability }
func (*SnifferConfig) Kind() Kind           { return KindSnifferConfig }
func (*RawData) Kind() Kind                 { return KindRawData }
func (*Unhandled) Kind() Kind               { return KindUnhandled }
func (*ProtocolError) Kind() Kind           { return KindProtocolError }
func (*InputStageChange) Kind() Kind        { return KindInputStage }
func (*TrackingChange) Kind() Kind          { return KindTracking }

func (*Sweep) message()                   {}
func (*SweepConfig) message()             {}
func (*DeviceInfo) message()              {}
func (*SerialNumber) message()            {}
func (*CalculatorChange) message()        {}
func (*Preset) message()                  {}
func (*PresetsEnd) message()              {}
func (*ScreenImage) message()             {}
func (*Calibration) message()             {}
func (*CalibrationAvailability) message() {}
func (*SnifferConfig) message()           {}
func (*RawData) message()                 {}
func (*Unhandled) message()               {}
func (*ProtocolError) message()           {}
func (*InputStageChange) message()        {}
func (*TrackingChange) message()          {}
