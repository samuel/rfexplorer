package rfx

// https://github.com/RFExplorer/RFExplorer-for-.NET/wiki/RF-Explorer-UART-API-interface-specification
//   https://web.archive.org/web/*/https://github.com/RFExplorer/RFExplorer-for-.NET/wiki/RF-Explorer-UART-API-interface-specification

// TODO https://github.com/RFExplorer/RFExplorer-for-Python/blob/master/RFExplorer/RFE6GEN_CalibrationData.py

// Calibration data: https://github.com/arocholl/rfexplorer/blob/325612c82eb44f528297c42fa2c1749d7a1cf375/RFExplorerCommunicator/RFECommunicator.cs#L1389
// #[0x0A]Cb[0x01][0x1F][0x00][0x04][0x00][0x00]
//19:28:53.090 DEBUG Sent to RFE: #[0x1E]C4-F:0440000,00510,100,00266
//19:28:53.293 Received configuration: #C4-F:0440000,1,006,00266,0,00510,100,
//19:28:53.293 DEBUG Sent to RFE: #[0x0A]Cb[0x01][0x1F][0x00][0x04][0x00][0x00]
// $Cb[0x01][0x1F][0x00][0x04][0x00][0x00][0xE7][0xE9][0xE8][0xE9][0xDD][0xEA][0xE9][0xDB][0xDE][0xE2][0xE6][0xE6][0xE5][0xE7][0xE6][0xE6][0xF1][0xF1][0xF0][0xF1][0xF2][0xF2][0xF2][0xF1][0xEE][0xE9][0xE8][0xE4][0xEB][0xEB][0xE7][0xE6][0xEB][0xF1][0xF2][0xF2][0xF1][0xF0][0xF1][0xF1][0xF1][0xF1][0xF2][0xF1][0xF1][0xF0][0xF0][0xEB][0xEB][0xEA][0xEB][0xEA][0xEA][0xEA][0xEB][0xEB][0xEB][0xEA][0xEB][0xE8][0xE6][0xE3][0xE0][0xE1][0xE3][0xE4][0xE3][0xE4][0xE4][0xE4][0xE3][0xE3][0xE2][0xE3][0xE3][0xE3][0xE3][0xE1][0xE0][0xDD][0xE2][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xEB][0xE7][0xE8][0xE8][0xE9][0xE9][0xE9][0xE9][0xE7][0xE8][0xE8][0xE8][0xE8][0xE8][0xE9][0xE8][0xF0][0xF1][0xF1][0xF1][0xF1][0xF0][0xEF][0xF0][0xEF][0xEF][0xEF][0xEF][0xEE][0xEE][0xEF][0xEE][0xEF][0xEE][0xEF][0xEE][0xED][0xEC][0xEE][0xEE][0xEF][0xF0][0xF0][0xEF][0xF0][0xEF][0xEF][0xEE][0xEE][0xED][0xEE][0xEF][0xEF][0xEF][0xEF][0xEF][0xEF][0xEF][0xEF][0xF4][0xF2][0xEE][0xEA][0xEB][0xEC][0xE2][0xE2][0xE2][0xE2][0xE3][0xE3][0xE1][0xDF][0xE1][0xDE][0xE0][0xE0][0xE2][0xE0][0xE1][0xDF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0xFF][0x0D][0x0A]
//19:29:50.905 DEBUG Sent to RFE: #[0x0A]Cb[0x01][0x1F][0x00][0x04][0x00][0x00]

// https://web.archive.org/web/*/https://github.com/RFExplorer/RFExplorer-for-.NET/wiki/RF-Explorer-UART-API-interface-specification

// https://github.com/RFExplorer/RFExplorer-for-Python/blob/ea25c90c500bdc2b78b146903ce65ff97a0abe69/RFExplorer/RFExplorer.py#L999
/*
#If it is a MODEL_WSUB3G, make sure we use the MAX HOLD mode to account for proper DSP
self.m_eCalculator = objConfiguration.eCalculator
time.sleep(0.5)
if (self.m_bUseMaxHold):
	if (self.m_eCalculator != RFE_Common.eCalculator.MAX_HOLD):
		print("Updated remote mode to Max Hold for reliable DSP calculations with fast signals")
		self.SendCommand_SetMaxHold()
else:
	if (self.m_eCalculator == RFE_Common.eCalculator.MAX_HOLD):
		print("Remote mode is not Max Hold, some fast signals may not be detected")
		self.SendCommand_Realtime()
*/

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"sync"
	"time"
)

const MaxSpectrumSteps = 65535

type Model int

const (
	Model433M           Model = 0
	Model868M           Model = 1
	Model915M           Model = 2
	ModelWSUB1G         Model = 3
	Model24G            Model = 4
	ModelWSUB3G         Model = 5
	Model6G             Model = 6
	ModelWSUB1GPlus     Model = 10
	ModelAudioPro       Model = 11 // note from Python library: this is converted internally to MODEL_WSUB3G to simplify code, but sets m_bAudioPro to true
	Model2400Plus       Model = 12
	Model4GPlus         Model = 13
	Model6GPlus         Model = 14
	ModelW5G3G          Model = 16
	ModelW5G4G          Model = 17
	ModelW5G5G          Model = 18
	ModelRFGen          Model = 60
	ModelRFGenExpansion Model = 61
	ModelNone           Model = 255
	ModelInvalid        Model = -1
)

type Mode int

const (
	ModeSpectrumAnalyzer  Mode = 0
	ModeRFGenerator       Mode = 1
	ModeWIFIAnalyzer      Mode = 2
	ModeAnalyzerTracking  Mode = 5
	ModeRFSniffer         Mode = 6
	ModeCWTransmitter     Mode = 60
	ModeSweepFrequency    Mode = 61
	ModeSweetAmplitude    Mode = 62
	ModeGeneratorTracking Mode = 63
	ModeUnknown           Mode = 255
	ModeInvalid           Mode = -1
)

type CalculatorMode int

const (
	CalculatorModeNormal        CalculatorMode = 0
	CalculatorModeMax           CalculatorMode = 1
	CalculatorModeAvg           CalculatorMode = 2
	CalculatorModeOverwrite     CalculatorMode = 3
	CalculatorModeMaxHold       CalculatorMode = 4
	CalculatorModeMaxHistorical CalculatorMode = 5
	CalculatorModeInvalid       CalculatorMode = -1
)

type WifiMode byte

const (
	WifiModeDisabled WifiMode = 0
	WifiMode24GHz    WifiMode = 1
	WifiMode5GHz     WifiMode = 2
)

type MarkerMode byte

const (
	MarkerModePeak   MarkerMode = 0
	MarkerModeNone   MarkerMode = 1
	MarkerModeManual MarkerMode = 2
)

type Modulation int

const (
	ModulationOOKRaw Modulation = 0
	ModulationPSKRaw Modulation = 1
	ModulationOOKStd Modulation = 2
	ModulationPSKStd Modulation = 3
	ModulationNone   Modulation = 0xff
)

type DSP byte

const (
	DSPAuto   DSP = 0
	DSPFilter DSP = 1
	DSPFast   DSP = 2
	DSPNoImg  DSP = 3
)

type InputStage byte

const (
	InputStageDirect         InputStage = 0
	InputStageAttenuator30dB InputStage = 1
	InputStageLNA25dB        InputStage = 2
	InputStageAttenuator60dB InputStage = 3
	InputStageLNA12dB        InputStage = 4
)

type CurrentConfigPacket struct {
	StartFreqKHZ    int
	FreqStepHZ      int
	AmpTopDBM       int
	AmpBottomDBM    int
	SweepSteps      int
	ExpModuleActive bool
	CurrentMode     Mode
	MinFreqKHZ      int
	MaxFreqKHZ      int
	MaxSpan         int
	RBWKHZ          int
	AmpOffset       int
	CalculatorMode  CalculatorMode
}

func (p *CurrentConfigPacket) Type() string {
	return "CurrentConfig"
}

type CurrentSetupPacket struct {
	Model           Model
	ExpansionModel  Model
	FirmwareVersion string
}

func (p *CurrentSetupPacket) Type() string {
	return "CurrentSetup"
}

type CalibrationAvailabilityPacket struct {
	MainboardInternalCalibrationAvailable      bool
	ExpansionBoardInternalCalibrationAvailable bool
}

func (p *CalibrationAvailabilityPacket) Type() string {
	return "CalibrationAvailability"
}

type SweepDataPacket struct {
	Samples []float64
}

func (p *SweepDataPacket) Type() string {
	return "SweepData"
}

type SerialNumberPacket struct {
	SN string
}

func (p *SerialNumberPacket) Type() string {
	return "SerialNumber"
}

// Preset represents a stored preset.
type Preset struct {
	// Index is the index of the preset starting at 0 (equivalent to 1 in the interface).
	// The valid range is [0,29] for standard units and [0,99] for Plus units.
	Index int
	// Name should be 7-bit ascii, ideally limited to A-Z, a-z, 0-9, and simple symbols like ., -, +, _, etc. Max length of 12.
	Name       string
	MinFreqKHz int
	MaxFreqKHz int
	// AmpTopDBm range [-110, +35]. Should be at least 10 more than AmpBottomDBm.
	AmpTopDBm int
	// AmpBottomDBm range [-120, +25]. Should be at least 10 less than AmpTopDBm.
	AmpBottomDBm int
	CalcMode     CalculatorMode
	// CalcIterations range [1, 16]
	CalcIterations int
	Mainboard      bool
	MarkerMode     MarkerMode
}

func (p *Preset) Type() string {
	return "Preset"
}

type EndOfPresetsPacket struct{}

func (p *EndOfPresetsPacket) Type() string {
	return "EndOfPresets"
}

type CurrentCalculatorMode struct {
	Mode CalculatorMode
}

func (p *CurrentCalculatorMode) Type() string {
	return "CurrentCalculatorMode"
}

type CurrentSnifferConfig struct {
	StartFreqKHZ    int
	ExpModuleActive bool
	CurrentMode     Mode
	Delay           int
	Modulation      Modulation
	RBWKHZ          int
	ThresholdDBM    float64
}

func (p *CurrentSnifferConfig) Type() string {
	return "CurrentSnifferConfig"
}

// ScreenImage is a image of the LCD screen sent by the device. It implements
// the image.Image interface.
type ScreenImage struct {
	Data []byte
}

func (si *ScreenImage) Type() string {
	return "ScreenImage"
}

// ColorModel returns the Image's color model.
func (si *ScreenImage) ColorModel() color.Model {
	return color.GrayModel
}

// Bounds returns the domain for which At can return non-zero color.
// The bounds do not necessarily contain the point (0, 0).
func (si *ScreenImage) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 128, Y: 64},
	}
}

// At returns the color of the pixel at (x, y).
// At(Bounds().Min.X, Bounds().Min.Y) returns the upper-left pixel of the grid.
// At(Bounds().Max.X-1, Bounds().Max.Y-1) returns the lower-right one.
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

// UnhandledPacket is the contents of an unhandled packet sent from RF Explorer.
type UnhandledPacket struct {
	Data []byte
}

func (p *UnhandledPacket) Type() string {
	return "UnhandledPacket"
}

// CalibrationDataPacket is the embedded amplitude calibration table sent by
// the device. The layout depends on the installed boards.
type CalibrationDataPacket struct {
	Data []byte
}

func (p *CalibrationDataPacket) Type() string {
	return "CalibrationData"
}

// DecodeErrorPacket reports a frame that was recognized but malformed.
type DecodeErrorPacket struct {
	// Data is the frame without its terminator.
	Data []byte
	Err  error
}

func (p *DecodeErrorPacket) Type() string {
	return "DecodeError"
}

// RawData is a packet of raw bytes sent from RF explorer as used by the sniffer.
type RawData struct {
	Data []byte
}

func (p *RawData) Type() string {
	return "RawData"
}

func (m Model) String() string {
	switch m {
	case Model433M:
		return "433M"
	case Model868M:
		return "868M"
	case Model915M:
		return "915M"
	case ModelWSUB1G:
		return "WSUB1G"
	case Model24G:
		return "2.4G"
	case ModelWSUB3G:
		return "WSUB3G"
	case Model6G:
		return "6G"
	case ModelWSUB1GPlus:
		return "WSUB1G Plus"
	case ModelAudioPro:
		return "Audio Pro"
	case Model2400Plus:
		return "2400 Plus"
	case Model4GPlus:
		return "4G Plus"
	case Model6GPlus:
		return "6G Plus"
	case ModelW5G3G:
		return "ModelW5G3G"
	case ModelW5G4G:
		return "ModelW5G4G"
	case ModelW5G5G:
		return "ModelW5G5G"
	case ModelRFGen:
		return "RFE6GEN"
	case ModelRFGenExpansion:
		return "RFE6GEN Expansion"
	case ModelNone:
		return ""
	case ModelInvalid:
		return "INVALID"
	}
	return fmt.Sprintf("Model(%d)", int(m))
}

func (m Mode) String() string {
	switch m {
	case ModeSpectrumAnalyzer:
		return "SpectrumAnalyzer"
	case ModeRFGenerator:
		return "RFGenerator"
	case ModeWIFIAnalyzer:
		return "WIFIAnalyzer"
	case ModeAnalyzerTracking:
		return "AnalyzerTracking"
	case ModeRFSniffer:
		return "RFSniffer"
	case ModeCWTransmitter:
		return "CWTransmitter"
	case ModeSweepFrequency:
		return "SweepFrequency"
	case ModeSweetAmplitude:
		return "SweetAmplitude"
	case ModeGeneratorTracking:
		return "GeneratorTracking"
	case ModeUnknown:
		return "Unknown"
	case ModeInvalid:
		return "Invalid"
	}
	return fmt.Sprintf("Mode(%d)", int(m))
}

func (m CalculatorMode) String() string {
	switch m {
	case CalculatorModeNormal:
		return "Normal"
	case CalculatorModeMax:
		return "Max"
	case CalculatorModeAvg:
		return "Avg"
	case CalculatorModeOverwrite:
		return "Overwrite"
	case CalculatorModeMaxHold:
		return "MaxHold"
	case CalculatorModeMaxHistorical:
		return "MaxHistorical"
	case CalculatorModeInvalid:
		return "Invalid"
	}
	return fmt.Sprintf("CalculatorMode(%d)", int(m))
}

func (m MarkerMode) String() string {
	switch m {
	case MarkerModePeak:
		return "Peak"
	case MarkerModeNone:
		return "None"
	case MarkerModeManual:
		return "Manual"
	}
	return fmt.Sprintf("MarkerMode(%d)", int(m))
}

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

type Packet interface {
	Type() string
}

type RFExplorer struct {
	port          io.ReadWriteCloser
	closeCh       chan struct{}
	readCh        chan Packet
	endOfPresetCh chan struct{}

	txMu     sync.Mutex
	writeBuf []byte

	mu                                          sync.RWMutex
	mainBoardModel                              Model
	expansionBoardModel                         Model
	activeModel                                 Model
	calculatorMode                              CalculatorMode
	spectrumAnalyzerEmbeddedCalibrationOffsetDB []float64
}

// New initiates a connection to the RF Explorer over the provided device.
func New(port io.ReadWriteCloser) (*RFExplorer, error) {
	rf := &RFExplorer{
		port:          port,
		writeBuf:      make([]byte, 256),
		closeCh:       make(chan struct{}),
		readCh:        make(chan Packet, 16),
		endOfPresetCh: make(chan struct{}, 1),

		mainBoardModel:      ModelNone,
		expansionBoardModel: ModelNone,
		activeModel:         ModelNone,
		calculatorMode:      CalculatorModeNormal,
	}
	go rf.readLoop()

	// Get the initial config
	// TODO: this fails depending on mode
	if err := rf.RequestConfig(); err != nil {
		_ = rf.Close() // returning the original error
		return nil, err
	}
setupLoop:
	for {
		// TODO: timeout
		pkt, ok := <-rf.Chan()
		if !ok {
			_ = rf.Close() // returning the original error
			return nil, errors.New("rfx: failed to get current config")
		}
		switch pkt.(type) {
		case *CurrentConfigPacket:
			if err := rf.RequestInternalCalibrationData(); err != nil {
				_ = rf.Close() // returning the original error
				return nil, err
			}
			// TODO: wait for calibration data
			break setupLoop
		}
	}
	return rf, nil
}

// Close close the communucation device.
func (r *RFExplorer) Close() error {
	close(r.closeCh)
	close(r.readCh)
	return r.port.Close()
}

func (r *RFExplorer) Chan() chan Packet {
	return r.readCh
}

// SetLCDEnabled requests RF Explorer to turn the LCD on or off.
func (r *RFExplorer) SetLCDEnabled(enabled bool) error {
	if enabled {
		return r.SendCommand("L1")
	}
	return r.SendCommand("L0")
}

// SetScreenDumpEnabled requests RF Explorer to dump all screen data
func (r *RFExplorer) SetScreenDumpEnabled(enabled bool) error {
	if enabled {
		return r.SendCommand("D1")
	}
	return r.SendCommand("D0")
}

func (r *RFExplorer) SetTrackingStep(n int) error {
	// return r.SendCommand("k" + )
	// this.SendCommand("k" + (object) Convert.ToChar(Convert.ToByte((int) nStep >> 8)) + (object) Convert.ToChar(Convert.ToByte((int) nStep & (int) byte.MaxValue)));
	return nil // TODO
}

func (r *RFExplorer) ResetInternalBuffers() error {
	return r.SendCommand("Cr")
}

// RequestSerialNumber requests the serial number from the RF Explorer.
func (r *RFExplorer) RequestSerialNumber() error {
	return r.SendCommand("Cn")
}

// RequestConfig requests RF Explorer to send the current configuration.
func (r *RFExplorer) RequestConfig() error {
	return r.SendCommand("C0")
}

// RequestPresets requests RF explorer to send the presents.
func (r *RFExplorer) RequestPresets() error {
	return r.SendCommand("CP\x00")
}

// UpdatePreset updates a stored preset.
func (r *RFExplorer) UpdatePreset(ctx context.Context, p *Preset) error {
	// "#$CP" \x01 index:byte name:byte*12 \x00 \x00 minfreqkhz:uint32 maxfeqkhz:uint32 calcmode:byte amptop:int8 ampbottom:int8 calciter:byte mainboard:bool markermode:byte \x42 \x00
	buf := make([]byte, 36)
	buf[0] = '#'
	buf[1] = 0x24
	buf[2] = 'C'
	buf[3] = 'P'
	buf[4] = 0x01

	// TODO: filter invalid characters from name
	// TODO: validate / clamp all parameters
	buf[5] = byte(p.Index)
	name := p.Name
	if len(name) > 12 {
		name = name[:12]
	}
	copy(buf[6:], name)
	buf[6+len(name)] = 0
	buf[18] = 0
	buf[19] = 0
	binary.LittleEndian.PutUint32(buf[20:24], uint32(p.MinFreqKHz))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(p.MaxFreqKHz))
	buf[28] = byte(p.CalcMode)
	buf[29] = byte(int8(p.AmpTopDBm))
	buf[30] = byte(int8(p.AmpBottomDBm))
	buf[31] = byte(p.CalcIterations)
	if p.Mainboard {
		buf[32] = 1
	} else {
		buf[32] = 0
	}
	buf[33] = byte(p.MarkerMode)
	buf[34] = 0x42
	buf[35] = 0
	// Clear end of preset channel so we can know if we receive one.
	select {
	case <-r.endOfPresetCh:
	default:
	}
	if err := r.write(buf[:36]); err != nil {
		return err

	}
	// Wait for end of presets
	select {
	case <-r.endOfPresetCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// RequestInternalCalibrationData requests RF Explorer to send the current configuration.
func (r *RFExplorer) RequestInternalCalibrationData() error {
	return r.SendCommand("Cq")
}

// SwitchModuleMain request RF Explorer to enable Mainboard module.
func (r *RFExplorer) SwitchModuleMain() error {
	return r.SendCommand("CM\x00")
}

// Hold stops receiving samples. Use RequestConfig to resume receiving samples.
func (r *RFExplorer) Hold() error {
	return r.SendCommand("CH")
}

// SwitchModuleExp request RF Explorer to enable Expansion module.
func (r *RFExplorer) SwitchModuleExp() error {
	return r.SendCommand("CM\x01")
}

// SetBaudRate requets RF Explorer to set the serial baud rate.
func (r *RFExplorer) SetBaudRate(br BaudRate) error {
	switch br {
	case BaudRate1200:
		return r.SendCommand("c1")
	case BaudRate2400:
		return r.SendCommand("c2")
	case BaudRate4800:
		return r.SendCommand("c3")
	case BaudRate9600:
		return r.SendCommand("c4")
	case BaudRate19200:
		return r.SendCommand("c5")
	case BaudRate38400:
		return r.SendCommand("c6")
	case BaudRate57600:
		return r.SendCommand("c7")
	case BaudRate115200:
		return r.SendCommand("c8")
	case BaudRate500000:
		return r.SendCommand("c0")
	}
	return fmt.Errorf("rfx: unknown baud rate %d", br)
}

// Realtime sets RF Explorer SA device in Calculator:Normal, this is useful to minimize spikes and spurs produced by unwanted signals
func (r *RFExplorer) Realtime() error {
	return r.SendCommand("C+\x00")
}

func (r *RFExplorer) SetMaxHold() error {
	return r.SendCommand("C+\x04")
}

func (r *RFExplorer) SetCalculatorMode(mode CalculatorMode) error {
	return r.SendCommand(fmt.Sprintf("C+%c", byte(mode)))
}

func (r *RFExplorer) SetWifiMode(mode WifiMode) error {
	return r.SendCommand(fmt.Sprintf("CW%d", byte(mode)))
}

// TODO: SetDSP	#<Size>Cp <DSP_Mode>	Request RF Explorer to set onboard DSP mode <Size>=5 bytes	1.12

func (r *RFExplorer) Shutdown() error {
	return r.SendCommand("S") // or is it "CS"
}

func (r *RFExplorer) SetGeneratorPower(on bool) error {
	if on {
		return r.SendCommand("CP1")
	}
	return r.SendCommand("CP0")
}

// TODO: SetOffsetDB	#<Size>CO <OffsetDB>	Request RF Explorer to set onboard Amplitude Offset in dB <Size>=5 bytes
// TODO: SetInputStage	#<Size>a <InputStage>	Request RF Explorer to set onboard input stage mode, available in WSUB1G+ and IoT models only <Size>=4 bytes
// TODO: SetSweepPointsLarge	#<Size>Cj <Sample_points_large>	Request RF Explorer to change to new data point sweep size <Size>=6 bytes - this mode support sweep sizes up to 65536 data points

// SetSweepPoints sets the number of sweep data points (16-4096, multiple of 16).
// May only work with 112, 240, and 512 or may not do anything at all.
func (r *RFExplorer) SetSweepPoints(steps int) error {
	if steps < 16 {
		steps = 16
	}
	if steps > 4096 {
		steps = 4096
	}
	return r.SendCommand("CJ" + string([]byte{byte((steps - 16) / 16)}))
}

// SetSweepPointsEx sets the number of sweep data points (112-65536, multiple of 2).
func (r *RFExplorer) SetSweepPointsEx(steps int) error {
	if steps < 112 {
		steps = 112
	}
	if steps > 65536 {
		steps = 65536
	}
	return r.SendCommand("Cj" + string([]byte{byte((steps & 0xff00) >> 8), byte(steps & 0xff)}))
}

// SetAnalyzerConfig will change current configuration for RF Explorer and send current Spectrum Analyzer configuration data back to PC.
// rbwKHZ is currently ignored by the firmware.
func (r *RFExplorer) SetAnalyzerConfig(startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm, rbwKHZ int) error {
	// #<Size>C2-F: <Start_Freq>, <End_Freq>, <Amp_Top>, <Amp_Bottom>, <RBW_KHZ>
	// <Start_Freq>, <End_Freq> = 7 ascii digits, decimal
	// <Amp_Top>, <Amp_Bottom> = 4 ascii digits, decimal
	// <RBW_KHZ> = 5 ascii digits, decimal
	if startFreqKHZ < 0 || endFreqKHZ < 0 || startFreqKHZ > 9999999 || endFreqKHZ > 9999999 {
		return errors.New("rfx: SetAnalyzerConfig startFreqKHZ and endFreqKHZ must be in the range [0,9999999]")
	}
	if ampTopDBm > 0 {
		ampTopDBm = 0
	}
	if ampTopDBm < -120 {
		ampTopDBm = -120
	}
	if ampBottomDBm >= ampTopDBm || ampBottomDBm < -120 {
		ampBottomDBm = -120
	}

	var rbwKHZStr string
	if rbwKHZ > 0 && rbwKHZ >= 3 && rbwKHZ <= 670 {
		steps := min(max((endFreqKHZ-startFreqKHZ+rbwKHZ/2)/rbwKHZ, 112), MaxSpectrumSteps)
		rbwKHZ = (endFreqKHZ - startFreqKHZ + steps/2) / steps
		if rbwKHZ >= 3 && rbwKHZ < 620 {
			rbwKHZStr = fmt.Sprintf(",%05d", rbwKHZ)
		}
	}

	cmd := fmt.Sprintf("C2-F:%07d,%07d,%04d,%04d%s", startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm, rbwKHZStr)
	if err := r.SendCommand(cmd); err != nil {
		return err
	}
	// wait some time for the unit to process changes, otherwise may get a different command too soon
	time.Sleep(time.Millisecond * 500)
	return nil
}

// Sample rate value should be in range 20,000 – 500,000 for OOK RAW modulation modes usually found
// in commercial devices, but some experimentation may be needed. This is the sample rate at which
// the internal decoder will detect activity – the higher this value the better capture resolution
// but at the cost of a shorter capture time lapse.
func (r *RFExplorer) SetSnifferConfig(centerFreqKHZ int, sampleRate int) error {
	return nil // TODO
}

// SendCommand sends a "#" command to the RF Explorer
func (r *RFExplorer) SendCommand(cmd string) error {
	r.txMu.Lock()
	defer r.txMu.Unlock()
	if len(cmd) > 253 {
		return fmt.Errorf("rfx: command may not exceed a length of 253, got %d", len(cmd))
	}
	if cap(r.writeBuf) < len(cmd)+2 {
		r.writeBuf = make([]byte, len(cmd)+2)
	}
	r.writeBuf[0] = '#'
	r.writeBuf[1] = byte(2 + len(cmd))
	copy(r.writeBuf[2:], cmd)
	return r.write(r.writeBuf[:2+len(cmd)])
}

// ActiveModel returns the currently active board model.
func (r *RFExplorer) ActiveModel() Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeModel
}

// MainBoardModel returns the model of the main board.
func (r *RFExplorer) MainBoardModel() Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mainBoardModel
}

// ExpansionBoardModel returns the model of the expansion board.
func (r *RFExplorer) ExpansionBoardModel() Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.expansionBoardModel
}

// CalculatorMode returns the current calculator mode.
func (r *RFExplorer) CalculatorMode() CalculatorMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calculatorMode
}

func (r *RFExplorer) write(b []byte) error {
	if n, err := r.port.Write(b); err != nil {
		return fmt.Errorf("rfx: failed to write to port: %w", err)
	} else if n != len(b) {
		return fmt.Errorf("rfx: expected to write %d bytes but wrote %d", len(b), n)
	}
	return nil
}

func (r *RFExplorer) handlePacket(pkt Packet) {
	r.readCh <- pkt
}

func (r *RFExplorer) readLoop() {
	var f framer
	buf := make([]byte, 4096)
	for {
		n, err := r.port.Read(buf)
		if err != nil {
			// TODO
			log.Fatal(err)
		}
		select {
		case <-r.closeCh:
			return
		default:
		}
		f.Write(buf[:n])
		for {
			frame, ok := f.Next()
			if !ok {
				break
			}
			pkt, err := decode(frame)
			if err != nil {
				pkt = &DecodeErrorPacket{Data: bytes.Clone(frame), Err: err}
			}
			r.updateState(pkt)
			r.handlePacket(pkt)
		}
	}
}

// updateState tracks the device state reported by pkt.
func (r *RFExplorer) updateState(pkt Packet) {
	switch pkt := pkt.(type) {
	case *CurrentCalculatorMode:
		r.mu.Lock()
		r.calculatorMode = pkt.Mode
		r.mu.Unlock()
	case *CurrentConfigPacket:
		r.mu.Lock()
		if pkt.ExpModuleActive {
			r.activeModel = r.expansionBoardModel
		} else {
			r.activeModel = r.mainBoardModel
		}
		r.calculatorMode = pkt.CalculatorMode
		r.mu.Unlock()
	case *CurrentSetupPacket:
		r.mu.Lock()
		r.mainBoardModel = pkt.Model
		r.expansionBoardModel = pkt.ExpansionModel
		r.mu.Unlock()
	case *CalibrationDataPacket:
		r.mu.Lock()
		r.spectrumAnalyzerEmbeddedCalibrationOffsetDB = calibrationOffsetsDB(r.activeModel, pkt.Data)
		r.mu.Unlock()
	case *EndOfPresetsPacket:
		// Wake a pending UpdatePreset. A signal is already queued if nothing
		// is waiting.
		select {
		case r.endOfPresetCh <- struct{}{}:
		default:
		}
	}
}
