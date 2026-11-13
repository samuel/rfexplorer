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
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
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

// ErrClosed is returned by operations on a closed RFExplorer.
var ErrClosed = errors.New("rfx: closed")

// configEchoTimeout is how long to wait for the device to report a changed
// configuration before asking for it. Echoes arrive within about 200 ms.
var configEchoTimeout = 1500 * time.Millisecond

type RFExplorer struct {
	port       io.ReadWriteCloser
	closeCh    chan struct{} // closed by Close
	readerDone chan struct{} // closed when the reader exits
	readCh     chan Packet
	queue      *packetQueue
	waiters    waiters
	wg         sync.WaitGroup
	closeOnce  sync.Once
	closeErr   error

	// reqMu serializes commands that wait for a response.
	reqMu sync.Mutex
	// txMu serializes writes to the port.
	txMu sync.Mutex
	// txReady is signaled when the device may have room for another command.
	txReady chan struct{}

	flowMu sync.Mutex
	flow   txWindow

	mu                                          sync.RWMutex
	err                                         error
	config                                      *CurrentConfigPacket
	mainBoardModel                              Model
	expansionBoardModel                         Model
	activeModel                                 Model
	calculatorMode                              CalculatorMode
	spectrumAnalyzerEmbeddedCalibrationOffsetDB []float64
}

// New starts communicating with the RF Explorer connected to port and
// requests its configuration. ctx bounds the wait for the device to answer.
// A device that is not in an analyzer mode may not respond.
func New(ctx context.Context, port io.ReadWriteCloser) (*RFExplorer, error) {
	rf := &RFExplorer{
		port:       port,
		closeCh:    make(chan struct{}),
		readerDone: make(chan struct{}),
		readCh:     make(chan Packet, 16),
		queue:      newPacketQueue(),
		txReady:    make(chan struct{}, 1),

		mainBoardModel:      ModelNone,
		expansionBoardModel: ModelNone,
		activeModel:         ModelNone,
		calculatorMode:      CalculatorModeNormal,
	}
	rf.wg.Add(2)
	go rf.readLoop()
	go rf.forwardLoop()

	w := rf.waiters.add(isConfig)
	err := rf.RequestConfig()
	if err == nil {
		if _, err = rf.await(ctx, w); err != nil {
			err = fmt.Errorf("rfx: no response to config request: %w", err)
		}
	}
	if err == nil {
		err = rf.RequestInternalCalibrationData()
	}
	if err != nil {
		_ = rf.Close() // returning the original error
		return nil, err
	}
	return rf, nil
}

func isConfig(pkt Packet) bool {
	_, ok := pkt.(*CurrentConfigPacket)
	return ok
}

func isEndOfPresets(pkt Packet) bool {
	_, ok := pkt.(*EndOfPresetsPacket)
	return ok
}

// Close stops communication and closes the port. The packet channel is
// closed once Close returns.
func (r *RFExplorer) Close() error {
	r.closeOnce.Do(func() {
		close(r.closeCh)
		r.closeErr = r.port.Close()
		r.wg.Wait()
	})
	return r.closeErr
}

// Chan returns the channel of packets received from the device. It is closed
// after Close or when reading from the port fails; see Err. If the consumer
// falls behind, the oldest sweeps and screen images are discarded.
func (r *RFExplorer) Chan() <-chan Packet {
	return r.readCh
}

// Err returns the error that stopped reading from the port, or nil if reading
// is still active or was stopped by Close.
func (r *RFExplorer) Err() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.err
}

// Dropped returns the number of packets discarded because the consumer of
// Chan fell behind.
func (r *RFExplorer) Dropped() uint64 {
	return r.queue.droppedCount()
}

// await waits for the packet w was registered for.
func (r *RFExplorer) await(ctx context.Context, w *waiter) (Packet, error) {
	defer r.waiters.remove(w)
	select {
	case pkt := <-w.ch:
		return pkt, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.readerDone:
		if err := r.Err(); err != nil {
			return nil, err
		}
		return nil, ErrClosed
	}
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

// SetTrackingStep requests the device to jump to a tracking step frequency
// and measure it immediately.
func (r *RFExplorer) SetTrackingStep(step uint16) error {
	return r.SendCommand(string([]byte{'k', byte(step >> 8), byte(step)}))
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

// UpdatePreset stores a preset on the device and waits for it to be
// acknowledged.
func (r *RFExplorer) UpdatePreset(ctx context.Context, p *Preset) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return r.writePreset(ctx, p)
}

// DeletePreset clears the preset slot at index and waits for it to be
// acknowledged.
func (r *RFExplorer) DeletePreset(ctx context.Context, index int) error {
	if index < 0 || index >= MaxPresets {
		return fmt.Errorf("rfx: preset index %d out of range [0,%d]", index, MaxPresets-1)
	}
	// A slot with an empty name is unused.
	return r.writePreset(ctx, &Preset{Index: index})
}

func (r *RFExplorer) writePreset(ctx context.Context, p *Preset) error {
	r.reqMu.Lock()
	defer r.reqMu.Unlock()
	w := r.waiters.add(isEndOfPresets)
	if err := r.write(encodePreset(p)); err != nil {
		r.waiters.remove(w)
		return err
	}
	_, err := r.await(ctx, w)
	return err
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

// Realtime sets the calculator to normal mode, which minimizes spikes and
// spurs produced by unwanted signals.
func (r *RFExplorer) Realtime() error {
	return r.SetCalculatorMode(CalculatorModeNormal)
}

// SetMaxHold sets the calculator to max hold mode, which captures fast
// transient signals.
func (r *RFExplorer) SetMaxHold() error {
	return r.SetCalculatorMode(CalculatorModeMaxHold)
}

func (r *RFExplorer) SetCalculatorMode(mode CalculatorMode) error {
	if mode < CalculatorModeNormal || mode > CalculatorModeMaxHistorical {
		return fmt.Errorf("rfx: invalid calculator mode %d", int(mode))
	}
	return r.SendCommand(string([]byte{'C', '+', byte(mode)}))
}

// SetWifiMode enables or disables the WiFi analyzer. The mode is sent as a
// raw byte; firmware 1.34 ignores an ASCII digit. Leaving WiFi mode does not
// restore the previous frequency range or calculator mode.
func (r *RFExplorer) SetWifiMode(mode WifiMode) error {
	if mode > WifiMode5GHz {
		return fmt.Errorf("rfx: invalid WiFi mode %d", mode)
	}
	return r.SendCommand(string([]byte{'C', 'W', byte(mode)}))
}

// TODO: SetDSP	#<Size>Cp <DSP_Mode>	Request RF Explorer to set onboard DSP mode <Size>=5 bytes	1.12

// Shutdown powers the device off. Per the UART specification (firmware
// 1.11+); not exercised against hardware.
func (r *RFExplorer) Shutdown() error {
	return r.SendCommand("S")
}

func (r *RFExplorer) SetGeneratorPower(on bool) error {
	if on {
		return r.SendCommand("CP1")
	}
	return r.SendCommand("CP0")
}

// TODO: SetOffsetDB	#<Size>CO <OffsetDB>	Request RF Explorer to set onboard Amplitude Offset in dB <Size>=5 bytes
// TODO: SetInputStage	#<Size>a <InputStage>	Request RF Explorer to set onboard input stage mode, available in WSUB1G+ and IoT models only <Size>=4 bytes

// SetSweepPoints sets the number of sweep data points (16-4096, multiple of
// 16). Only models with selectable resolution act on it; a 6G + WSUB3G unit
// on firmware 1.34 stays at 112 points.
func (r *RFExplorer) SetSweepPoints(steps int) error {
	steps = min(max(steps, 16), 4096)
	return r.SendCommand(string([]byte{'C', 'J', byte((steps - 16) / 16)}))
}

// SetSweepPointsEx sets the number of sweep data points (112-65535). See
// SetSweepPoints for model support.
func (r *RFExplorer) SetSweepPointsEx(steps int) error {
	steps = min(max(steps, 112), MaxSpectrumSteps)
	return r.SendCommand(string([]byte{'C', 'j', byte(steps >> 8), byte(steps)}))
}

// SetAnalyzerConfig changes the frequency range and displayed amplitude range
// and returns the configuration the device reports afterwards, which may
// differ from the request. rbwKHZ of 0 leaves the resolution bandwidth to the
// device.
func (r *RFExplorer) SetAnalyzerConfig(ctx context.Context, startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm, rbwKHZ int) (*CurrentConfigPacket, error) {
	// #<Size>C2-F:<Start_Freq>,<End_Freq>,<Amp_Top>,<Amp_Bottom>[,<RBW_KHZ>]
	// <Start_Freq>, <End_Freq> = 7 ascii digits, decimal
	// <Amp_Top>, <Amp_Bottom> = 4 ascii digits, decimal
	// <RBW_KHZ> = 5 ascii digits, decimal
	if startFreqKHZ < 0 || endFreqKHZ > maxFreqKHz || startFreqKHZ >= endFreqKHZ {
		return nil, fmt.Errorf("rfx: frequency range %d to %d kHz is invalid", startFreqKHZ, endFreqKHZ)
	}
	if err := validateAmplitude(ampTopDBm, ampBottomDBm); err != nil {
		return nil, err
	}

	var rbwKHZStr string
	if rbwKHZ >= 3 && rbwKHZ <= 670 {
		steps := min(max((endFreqKHZ-startFreqKHZ+rbwKHZ/2)/rbwKHZ, 112), MaxSpectrumSteps)
		rbwKHZ = (endFreqKHZ - startFreqKHZ + steps/2) / steps
		if rbwKHZ >= 3 && rbwKHZ < 620 {
			rbwKHZStr = fmt.Sprintf(",%05d", rbwKHZ)
		}
	}

	r.reqMu.Lock()
	defer r.reqMu.Unlock()

	// The device only reports its configuration when the command changed it.
	if !r.configMatches(startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm) || rbwKHZStr != "" {
		w := r.waiters.add(func(pkt Packet) bool {
			cfg, ok := pkt.(*CurrentConfigPacket)
			return ok && cfg.StartFreqKHZ == startFreqKHZ
		})
		cmd := fmt.Sprintf("C2-F:%07d,%07d,%04d,%04d%s", startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm, rbwKHZStr)
		if err := r.SendCommand(cmd); err != nil {
			r.waiters.remove(w)
			return nil, err
		}
		echoCtx, cancel := context.WithTimeout(ctx, configEchoTimeout)
		pkt, err := r.await(echoCtx, w)
		cancel()
		if err == nil {
			return pkt.(*CurrentConfigPacket), nil
		}
		if ctx.Err() != nil || !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		// No echo: the device adjusted the range or considered it unchanged.
	}

	w := r.waiters.add(isConfig)
	if err := r.RequestConfig(); err != nil {
		r.waiters.remove(w)
		return nil, err
	}
	pkt, err := r.await(ctx, w)
	if err != nil {
		return nil, err
	}
	return pkt.(*CurrentConfigPacket), nil
}

// configMatches reports whether the last configuration reported by the
// device already has the given ranges.
func (r *RFExplorer) configMatches(startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm int) bool {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()
	if cfg == nil || cfg.StartFreqKHZ != startFreqKHZ || cfg.AmpTopDBM != ampTopDBm || cfg.AmpBottomDBM != ampBottomDBm {
		return false
	}
	// The device reports a step size rather than the end frequency.
	end := cfg.StartFreqKHZ + cfg.FreqStepHZ*(cfg.SweepSteps-1)/1000
	return end >= endFreqKHZ-1 && end <= endFreqKHZ+1
}

// SendCommand sends a "#" command to the RF Explorer. The device buffers 64
// bytes of commands and handles about one per sweep, so SendCommand blocks
// when more than that is pending.
func (r *RFExplorer) SendCommand(cmd string) error {
	if len(cmd) > 253 {
		return fmt.Errorf("rfx: command may not exceed a length of 253, got %d", len(cmd))
	}
	buf := make([]byte, 0, 2+len(cmd))
	buf = append(buf, '#', byte(2+len(cmd)))
	buf = append(buf, cmd...)
	return r.write(buf)
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
	r.txMu.Lock()
	defer r.txMu.Unlock()
	// Wait for room in the device's receive buffer.
	for {
		r.flowMu.Lock()
		wait, ok := r.flow.reserve(len(b), time.Now())
		r.flowMu.Unlock()
		if ok {
			break
		}
		timer := time.NewTimer(wait)
		select {
		case <-r.txReady:
		case <-timer.C:
		case <-r.closeCh:
			timer.Stop()
			return ErrClosed
		}
		timer.Stop()
	}
	if n, err := r.port.Write(b); err != nil {
		return fmt.Errorf("rfx: failed to write to port: %w", err)
	} else if n != len(b) {
		return fmt.Errorf("rfx: expected to write %d bytes but wrote %d", len(b), n)
	}
	return nil
}

// readLoop reads from the port until it fails or is closed. It never blocks
// on the consumer.
func (r *RFExplorer) readLoop() {
	defer r.wg.Done()
	defer close(r.readerDone)
	var f framer
	buf := make([]byte, 4096)
	for {
		n, err := r.port.Read(buf)
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
			r.waiters.deliver(pkt)
			r.queue.push(pkt)
		}
		if err != nil {
			select {
			case <-r.closeCh:
			default:
				r.mu.Lock()
				r.err = fmt.Errorf("rfx: failed to read from port: %w", err)
				r.mu.Unlock()
			}
			return
		}
	}
}

// forwardLoop moves packets from the queue to the packet channel and closes
// the channel once reading has stopped and the queue is drained, or on Close.
func (r *RFExplorer) forwardLoop() {
	defer r.wg.Done()
	defer close(r.readCh)
	for {
		pkt, ok := r.queue.pop()
		if !ok {
			select {
			case <-r.closeCh:
				return
			case <-r.queue.notify:
			case <-r.readerDone:
				// The reader may have queued packets just before exiting.
				if pkt, ok = r.queue.pop(); !ok {
					return
				}
			}
			if !ok {
				continue
			}
		}
		select {
		case r.readCh <- pkt:
		case <-r.closeCh:
			return
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
	case *SweepDataPacket:
		r.flowMu.Lock()
		r.flow.sweep(time.Now())
		r.flowMu.Unlock()
		select {
		case r.txReady <- struct{}{}:
		default:
		}
	case *CurrentConfigPacket:
		r.mu.Lock()
		r.config = pkt
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
	}
}
