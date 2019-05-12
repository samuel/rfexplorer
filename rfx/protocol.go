package rfx

// https://github.com/RFExplorer/RFExplorer-for-.NET/wiki/RF-Explorer-UART-API-interface-specification

// TODO https://github.com/RFExplorer/RFExplorer-for-Python/blob/master/RFExplorer/RFE6GEN_CalibrationData.py

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

const MaxSpectrumSteps = 65535

type Model int

const (
	Model433M    Model = 0
	Model868M    Model = 1
	Model915M    Model = 2
	ModelWSUB1G  Model = 3
	Model24G     Model = 4
	ModelWSUB3G  Model = 5
	Model6G      Model = 6
	ModelRFGen   Model = 60
	ModelNone    Model = 255
	ModelInvalid Model = -1
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
	CalculatorModeNormal    CalculatorMode = 0
	CalculatorModeMax       CalculatorMode = 1
	CalculatorModeAvg       CalculatorMode = 2
	CalculatorModeOverwrite CalculatorMode = 3
	CalculatorModeMaxHold   CalculatorMode = 4
	CalculatorModeInvalid   CalculatorMode = -1
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

func parseModulation(s string) Modulation {
	i, _ := strconv.Atoi(s)
	return Modulation(i)
}

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

// AtGray returns the color.Gray of the pixel at (x, y).
// At(Bounds().Min.X, Bounds().Min.Y) returns the upper-left pixel of the grid.
// At(Bounds().Max.X-1, Bounds().Max.Y-1) returns the lower-right one.
func (si *ScreenImage) AtGray(x, y int) color.Gray {
	return color.Gray{Y: 255 ^ (255 * ((si.Data[(y/8)*128+x] >> (uint(y) % 8)) & 1))}
}

// UnhandledPacket is the contents of an unhandled packet sent from RF Explorer.
type UnhandledPacket struct {
	Data []byte
}

func (p *UnhandledPacket) Type() string {
	return "UnhandledPacket"
}

// RawData is a packet of raw bytes sent from RF explorer as used by the sniffer.
type RawData struct {
	Data []byte
}

func (p *RawData) Type() string {
	return "RawData"
}

func parseASCIIDecimal(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(strings.TrimLeft(s, "0"))
	return i
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
	case ModelRFGen:
		return "RFE6GEN"
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

// class eDSP(Enum):
//     """All possible DSP values
// 	"""
//     DSP_AUTO = 0
//     DSP_FILTER = 1
//     DSP_FAST = 2
//     DSP_NO_IMG = 3

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

func parseModel(m string) Model {
	if m == "" {
		return ModelNone
	}
	m = strings.TrimLeft(m, "0")
	if m == "" {
		return Model(0)
	}
	i, err := strconv.Atoi(m)
	if err != nil {
		return ModelInvalid
	}
	return Model(i)
}

func parseMode(m string) Mode {
	if m == "" {
		return ModeInvalid
	}
	m = strings.TrimLeft(m, "0")
	if m == "" {
		return Mode(0)
	}
	i, err := strconv.Atoi(m)
	if err != nil {
		return ModeInvalid
	}
	return Mode(i)
}

func parseCalculatorMode(m string) CalculatorMode {
	if m == "" {
		return CalculatorModeInvalid
	}
	m = strings.TrimLeft(m, "0")
	if m == "" {
		return CalculatorMode(0)
	}
	i, err := strconv.Atoi(m)
	if err != nil {
		return CalculatorModeInvalid
	}
	return CalculatorMode(i)
}

type Packet interface {
	Type() string
}

type RFExplorer struct {
	port          io.ReadWriteCloser
	writeBuf      []byte
	closeCh       chan struct{}
	readCh        chan Packet
	config        atomic.Value // *CurrentConfigPacket
	endOfPresetCh chan struct{}
}

// New initiates a connection to the RF Explorer over the provided device.
// TODO: currently a baud rate of 500,000 is assumed.
func New(device string) (*RFExplorer, error) {
	options := serial.OpenOptions{
		PortName:        device,
		BaudRate:        500000,
		DataBits:        8,
		ParityMode:      serial.PARITY_NONE,
		StopBits:        1,
		MinimumReadSize: 1,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		return nil, err
	}

	rf := &RFExplorer{
		port:          port,
		writeBuf:      make([]byte, 256),
		closeCh:       make(chan struct{}),
		readCh:        make(chan Packet, 16),
		endOfPresetCh: make(chan struct{}, 1),
	}
	go rf.readLoop()

	// Get the initial config
	// TODO: this fails depending on mode
	if err := rf.RequestConfig(); err != nil {
		return nil, err
	}
setupLoop:
	for {
		pkt, ok := <-rf.Chan()
		if !ok {
			rf.Close()
			return nil, fmt.Errorf("rfx: failed to get current config")
		}
		switch pkt := pkt.(type) {
		case *CurrentConfigPacket:
			rf.config.Store(pkt)
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

func (r *RFExplorer) Config() *CurrentConfigPacket {
	return r.config.Load().(*CurrentConfigPacket)
}

// SetLCDEnabled requests RF Explorer to turn the LCD on or off.
func (r *RFExplorer) SetLCDEnabled(enabled bool) error {
	// #<Size>C(0|1)
	r.writeBuf[0] = '#'
	r.writeBuf[1] = 4
	r.writeBuf[2] = 'L'
	if enabled {
		r.writeBuf[3] = '1'
	} else {
		r.writeBuf[3] = '0'
	}
	return r.write(r.writeBuf[:4])
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
	// Way for end of presets
	select {
	case <-r.endOfPresetCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// RequestInternalCalibrationData requests RF Explorer to send the currnet configuration.
func (r *RFExplorer) RequestInternalCalibrationData() error {
	return r.SendCommand("Cq")
}

// SwitchModuleMain request RF Explorer to enable Mainboard module.
func (r *RFExplorer) SwitchModuleMain() error {
	return r.SendCommand("CM\x00")
}

// Hold stops receiving samples. Use RequestConfig to resume receving samples.
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

func (r *RFExplorer) Realtime() error {
	return r.SendCommand("C+\x00")
}

func (r *RFExplorer) SetMaxHold() error {
	return r.SendCommand("C+\x04")
}

func (r *RFExplorer) Shutdown() error {
	return r.SendCommand("CS")
}

func (r *RFExplorer) SetGeneratorPower(on bool) error {
	if on {
		return r.SendCommand("CP1")
	}
	return r.SendCommand("CP0")
}

// TODO: SetCalculator	#<Size>C+<CalcMode>	Request RF Explorer to set onboard calculator mode <Size>=5 bytes
// TODO: SetDSP	#<Size>Cp <DSP_Mode>	Request RF Explorer to set onboard DSP mode <Size>=5 bytes	1.12
// TODO: SetOffsetDB	#<Size>CO <OffsetDB>	Request RF Explorer to set onboard Amplitude Offset in dB <Size>=5 bytes
// TODO: SetInputStage	#<Size>a <InputStage>	Request RF Explorer to set onboard input stage mode, available in WSUB1G+ and IoT models only <Size>=4 bytes
// TODO: SetSweepPointsLarge	#<Size>Cj <Sample_points_large>	Request RF Explorer to change to new data point sweep size <Size>=6 bytes - this mode support sweep sizes up to 65536 data points

// SetSweepPoints sets the number of sweep data points (16-4096, multiple of 16).
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
func (r *RFExplorer) SetAnalyzerConfig(startFreqKHZ, endFreqKHZ, ampTopDBm, ampBottomDBm, rbwKHZ int) error {
	// #<Size>C2-F: <Start_Freq>, <End_Freq>, <Amp_Top>, <Amp_Bottom>, <RBW_KHZ>
	// <Start_Freq>, <End_Freq> = 7 ascii digits, decimal
	// <Amp_Top>, <Amp_Bottom> = 4 ascii digits, decimal
	// <RBW_KHZ> = 5 ascii digits, decimal
	if startFreqKHZ < 0 || endFreqKHZ < 0 || startFreqKHZ > 9999999 || endFreqKHZ > 9999999 {
		return fmt.Errorf("rfx: SetAnalyzerConfig startFreqKHZ and endFreqKHZ must be in the range [0,9999999]")
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
		steps := (endFreqKHZ - startFreqKHZ + rbwKHZ/2) / rbwKHZ
		if steps < 112 {
			steps = 112
		}
		if steps > MaxSpectrumSteps {
			steps = MaxSpectrumSteps
		}
		rbwKHZ = (endFreqKHZ - startFreqKHZ + steps/2) / steps
		if rbwKHZ >= 3 && rbwKHZ < 620 {
			rbwKHZStr = fmt.Sprintf(",%05d", rbwKHZ)
		} else {
			fmt.Printf("Ignored RBW %d Khz", rbwKHZ)
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

// Sample rate value should be in range 20,000 – 500,000 for OOK RAW modulation modes usually found in commercial devices, but some experimentation may be needed. This is the sample rate at which the internal decoder will detect activity – the higher this value the better capture resolution but at the cost of a shorter capture time lapse.
func (r *RFExplorer) SetSnifferConfig(centerFreqKHZ int, sampleRate int) error {
	return nil // TODO
}

// SendCommand sends a "#" command to the RF Explorer
func (r *RFExplorer) SendCommand(cmd string) error {
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

func (r *RFExplorer) write(b []byte) error {
	if n, err := r.port.Write(b); err != nil {
		return fmt.Errorf("rfx: failed to write to port: %s", err)
	} else if n != len(b) {
		return fmt.Errorf("rfx: expected to write %d bytes but wrote %d", len(b), n)
	}
	return nil
}

func (r *RFExplorer) handlePacket(pkt Packet) {
	r.readCh <- pkt
}

// var logFile *os.File

// func init() {
// 	var err error
// 	logFile, err = os.Create("log.bin")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func (r *RFExplorer) readLoop() {
	buf := make([]byte, 8192)
	off := 0
	for {
		if off >= len(buf)-1 {
			// TODO
			off = 0
		}
		n, err := r.port.Read(buf[off:])
		if err != nil {
			// TODO
			log.Fatal(err)
		}
		// logFile.Write(buf[off : off+n])
		select {
		case <-r.closeCh:
			return
		default:
		}
		if n == 0 {
			continue
		}
		off += n
	decodeLoop:
		for off > 2 {
			// See if there's an EOL
			eolIdx := bytes.Index(buf[:off], []byte{0x0d, 0x0a})
			// The buffer is guaranteed to be at least 3 bytes long now
			b := buf[:off]
			handled := false
			switch b[0] {
			case '$':
				// TODO: $C?
				switch b[1] {
				case 'D':
					if len(b) < 0x404 {
						break decodeLoop
					}
					data := make([]byte, 0x400)
					copy(data, b[2:0x402])
					r.handlePacket(&ScreenImage{
						Data: data,
					})
					eolIdx = 0x402
					handled = true
				case 'R':
					// Raw data (used for sniffer)
					nBytes := int(buf[2]) | (int(buf[3]) << 8)
					if len(b) < nBytes+4 {
						break decodeLoop
					}
					data := make([]byte, nBytes)
					copy(data, b[4:4+nBytes])
					r.handlePacket(&RawData{
						Data: data,
					})
					eolIdx = 4 + nBytes
					handled = true
				case 'S':
					// Sweep_data - $S<Sample_Steps> <AdBm>… <AdBm> <EOL> - Send all dBm sample points to PC client, in binary
					if eolIdx < 0 {
						break decodeLoop
					}
					if len(b) > 3 {
						nSamples := int(b[2])
						if len(b) < 3+nSamples {
							// TODO: insert error into packet stream
							fmt.Printf("SHORT\n")
						} else {
							if eolIdx < 3+nSamples {
								eolIdx = 3 + nSamples
								if eolIdx > len(b) {
									// TODO: handle this better
									fmt.Printf("LONG\n")
									eolIdx = len(b)
								}
							}
							samples := make([]float64, nSamples)
							for i, adbm := range b[3 : 3+nSamples] {
								// Sampled value in dBm, repeated n times one per sample. To get the real value in dBm, consider this an
								// unsigned byte, divide it by two and change sign to negative. For instance a byte=0x11 (17 decimal)
								// will be -17/2= -8.5dBm. This is now normalized and consistent for all modules and setups
								samples[i] = -float64(adbm) / 2.0
							}
							r.handlePacket(&SweepDataPacket{
								Samples: samples,
							})
							handled = true
						}
					}
				case 'P':
					// "$P " index:byte \x01 name:byte*12 \x00 \x00 minfreqkhz:uint32 maxfeqkhz:uint32 calcmode:byte amptop:int8 ampbottom:int8 calciter:byte mainboard:bool markermode:byte \x42 \x00
					nameBytes := buf[5 : 5+12]
					if ix := bytes.IndexByte(nameBytes, 0); ix >= 0 {
						nameBytes = nameBytes[:ix]
					}
					r.handlePacket(&Preset{
						Index:          int(buf[3]),
						Name:           string(nameBytes),
						MinFreqKHz:     int(binary.LittleEndian.Uint32(buf[19:23])),
						MaxFreqKHz:     int(binary.LittleEndian.Uint32(buf[23:27])),
						CalcMode:       CalculatorMode(buf[27]),
						AmpTopDBm:      int(int8(buf[28])),
						AmpBottomDBm:   int(int8(buf[29])),
						CalcIterations: int(buf[30]),
						Mainboard:      buf[31] != 0,
						MarkerMode:     MarkerMode(buf[32]),
					})
					handled = true
				}
			case '#':
				if eolIdx < 0 {
					break decodeLoop
				}
				b = buf[:eolIdx]
				// TODO: #QA:0 is received once on startup (TODO?)
				// TODO: #K1 & #K0 -- thread tracking something or other

				switch b[1] {
				case 'C':

					if len(b) > 6 {
						switch b[2] {
						case '2': // Spectrum Analyzer mode
							if b[3] == '-' && b[5] == ':' {
								switch b[4] {
								case 'F':
									// Current_config - #C2-F:<Start_Freq>, <Freq_Step>, <Amp_Top>, <Amp_Bottom>, <Sweep_Steps>,
									//                  <ExpModuleActive>, <CurrentMode>, <Min_Freq>, <Max_Freq>, <Max_Span>, <RBW>,
									//                  <AmpOffset>, <CalculatorMode> <EOL>
									// Send current Spectrum Analyzer configuration data. From RFE to PC, will be used
									// by the PC to control PC client GUI. Note this has been updated in v1.12
									p := strings.Split(string(b[6:]), ",")
									config := &CurrentConfigPacket{
										StartFreqKHZ:    parseASCIIDecimal(p[0]),
										FreqStepHZ:      parseASCIIDecimal(p[1]),
										AmpTopDBM:       parseASCIIDecimal(p[2]),
										AmpBottomDBM:    parseASCIIDecimal(p[3]),
										SweepSteps:      parseASCIIDecimal(p[4]),
										ExpModuleActive: p[5] == "1",
										CurrentMode:     parseMode(p[6]),
										MinFreqKHZ:      parseASCIIDecimal(p[7]),
										MaxFreqKHZ:      parseASCIIDecimal(p[8]),
										MaxSpan:         parseASCIIDecimal(p[9]),
										RBWKHZ:          parseASCIIDecimal(p[10]),
										AmpOffset:       parseASCIIDecimal(p[11]),
										CalculatorMode:  parseCalculatorMode(p[12]),
									}
									r.handlePacket(config)
									handled = true
								case 'M':
									// Current_Setup - #C2-M:<Main_Model>, <Expansion_Model>, <Firmware_Version> <EOL>
									// Send current Spectrum Analyzer model setup and firmware version	1.06
									p := strings.Split(string(b[6:]), ",")
									setup := &CurrentSetupPacket{
										// <Main_Model> - Codified values are 433M:0, 868M:1, 915M:2, WSUB1G:3, 2.4G:4, WSUB3G:5, 6G:6
										Model: parseModel(p[0]),
									}
									// <Expansion_Model> - Codified values are 433M:0, 868M:1, 915M:2, WSUB1G:3, 2.4G:4, WSUB3G:5, 6G:6, NONE:255
									if len(p) >= 2 {
										setup.ExpansionModel = parseModel(p[1])
									}
									if len(p) >= 3 {
										setup.FirmwareVersion = strings.TrimLeft(p[2], "0")
									}
									r.handlePacket(setup)
									handled = true
								}
							}
						// case '3': // Signal generator CW, SweepFreq and SweepAmp modes // TODO: #C3- https://github.com/RFExplorer/RFExplorer-for-Python/blob/master/RFExplorer/RFEConfiguration.py#L136
						case '4': // Sniffer mode
							// TODO: #C4- https://github.com/RFExplorer/RFExplorer-for-Python/blob/master/RFExplorer/RFEConfiguration.py#L190
							// self.fStartMHZ = int(sLine[6:13]) / 1000.0 #note it comes in KHZ
							// self.bExpansionBoardActive = (sLine[14] == '1')
							// self.m_eMode = RFE_Common.eMode(int(sLine[16:19]))
							// nDelay = int(sLine[20:25])
							// self.nBaudrate = int(round(float(RFE_Common.CONST_FCY_CLOCK) / nDelay))   #FCY_CLOCK = 16 * 1000 * 1000
							// self.eModulations = RFE_Common.eModulation(int(sLine[26:27]))
							// ... use Modulation type
							// self.fRBWKHZ = int(sLine[28:33])
							// self.fThresholdDBM = (float)(-0.5 * float(sLine[34:37]))
							if b[3] == '-' && b[4] == 'F' && b[5] == ':' {
								p := strings.Split(string(b[6:]), ",")
								r.handlePacket(&CurrentSnifferConfig{
									StartFreqKHZ:    parseASCIIDecimal(p[0]),
									ExpModuleActive: p[1] == "1",
									CurrentMode:     parseMode(p[2]),
									Delay:           parseASCIIDecimal(p[3]), // baudrate = (FCY_CLOCK=16*1000*1000)/delay,
									Modulation:      parseModulation(p[4]),
									RBWKHZ:          parseASCIIDecimal(p[5]),
									ThresholdDBM:    -0.5 * float64(parseASCIIDecimal(p[6])),
								})
								handled = true
							}
						case 'A':
							if b[3] == 'L' && b[4] == ':' {
								r.handlePacket(&CalibrationAvailabilityPacket{
									MainboardInternalCalibrationAvailable:      b[5] == '1',
									ExpansionBoardInternalCalibrationAvailable: b[6] == '1',
								})
								handled = true
							}
						}
					}
				case 'S':
					// Serial_Number - #Sn<SerialNumber> - device serial number
					if b[2] == 'n' {
						r.handlePacket(&SerialNumberPacket{SN: string(buf[3:eolIdx])})
						handled = true
					}
				case 'P':
					if len(b) >= 4 && string(b[:4]) == "#PCK" {
						select {
						case r.endOfPresetCh <- struct{}{}:
						default:
						}
						r.handlePacket(&EndOfPresetsPacket{})
						handled = true
					}
				}
			}
			if !handled && eolIdx >= 0 {
				// Need to copy the data as we reuse the buffer
				b2 := make([]byte, eolIdx)
				copy(b2, b[:eolIdx])
				r.handlePacket(&UnhandledPacket{Data: b2})
				handled = true
			}
			if !handled {
				break
			}
			copy(buf, buf[eolIdx+2:])
			off -= eolIdx + 2
		}
	}
}
