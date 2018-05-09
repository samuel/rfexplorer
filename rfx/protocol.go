package rfx

// https://github.com/RFExplorer/RFExplorer-for-.NET/wiki/RF-Explorer-UART-API-interface-specification

import (
	"bytes"
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

type CurrentSetupPacket struct {
	Model           Model
	ExpansionModel  Model
	FirmwareVersion string
}

type CalibrationAvailabilityPacket struct {
	MainboardInternalCalibrationAvailable      bool
	ExpansionBoardInternalCalibrationAvailable bool
}

type SweepDataPacket struct {
	Samples []float64
}

type SerialNumberPacket struct {
	SN string
}

type ScreenImage struct {
	Data []byte
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

type RawPacket struct {
	Data []byte
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
	return fmt.Sprintf("Mode(%d)", int(m))
}

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

type Packet interface{}

type RFExplorer struct {
	port     io.ReadWriteCloser
	writeBuf []byte
	closeCh  chan struct{}
	readCh   chan Packet
	config   atomic.Value // *CurrentConfigPacket
}

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
		port:     port,
		writeBuf: make([]byte, 256),
		closeCh:  make(chan struct{}),
		readCh:   make(chan Packet, 16),
	}
	go rf.readLoop()

	// Get the initial config
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

// RequestSerialNumber requests the serial number from the RF Explorer
func (r *RFExplorer) RequestSerialNumber() error {
	return r.SendCommand("Cn")
}

// RequestConfig requests RF Explorer to send the currnet configuration
func (r *RFExplorer) RequestConfig() error {
	return r.SendCommand("C0")
}

// RequestInternalCalibrationData requests RF Explorer to send the currnet configuration
func (r *RFExplorer) RequestInternalCalibrationData() error {
	return r.SendCommand("Cq")
}

// SwitchModuleMain request RF Explorer to enable Mainboard module
func (r *RFExplorer) SwitchModuleMain() error {
	return r.SendCommand("CM\x00")
}

// Hold stops receiving samples. Use RequestConfig to resume receving samples.
func (r *RFExplorer) Hold() error {
	return r.SendCommand("CH")
}

// SwitchModuleExp request RF Explorer to enable Expansion module
func (r *RFExplorer) SwitchModuleExp() error {
	return r.SendCommand("CM\x01")
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

// func (r *RFExplorer) SetSteps(steps int) error {
// 	// Not sure about this.. found it in https://github.com/RFExplorer/RFExplorer_3GP_IoT_Arduino/blob/cff0e6abb31a77a54aefc3803b5b99a62cb897fe/src/RFExplorer_3GP_IoT.cpp
//	// Or steps/16
// 	switch steps {
// 	case 112:
// 		return r.SendCommand("CP\x06")
// 	case 240:
// 		return r.SendCommand("CP\x0E")
// 	case 512:
// 		return r.SendCommand("CP\xFF")
// 	}
// 	return fmt.Errorf("rfx: unsupported number of steps %d", steps)
// }

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
	buf := make([]byte, 2048)
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
			eolIdx := bytes.IndexByte(buf[:off], '\r')
			if eolIdx < 0 || len(buf)-1 == eolIdx || buf[eolIdx+1] != '\n' {
				eolIdx = -1
			}
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
				case 'S':
					// Sweep_data - $S<Sample_Steps> <AdBm>… <AdBm> <EOL> - Send all dBm sample points to PC client, in binary
					if eolIdx < 0 {
						break decodeLoop
					}
					if len(b) > 3 && b[1] == 'S' {
						nSamples := int(b[2])
						if len(b) < 3+nSamples {
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
					// TODO: #C3- ??
					if len(b) > 6 {
						switch b[2] {
						case '2':
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
						r.handlePacket(&SerialNumberPacket{SN: string(buf[:eolIdx])})
						handled = true
					}
				}
			}
			if eolIdx < 0 {
				break
			}
			if eolIdx >= 0 && !handled {
				// Need to copy the data as we reuse the buffer
				b2 := make([]byte, eolIdx)
				copy(b2, b[:eolIdx])
				r.handlePacket(&RawPacket{Data: b2})
			}
			copy(buf, buf[eolIdx+2:])
			off -= eolIdx + 2
		}
	}
}
