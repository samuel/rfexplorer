package rfx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// presetFrameLen is the minimum length of a preset frame:
// "$P" len:byte index:byte \x01 name:byte*12 \x00 \x00 minfreqkhz:uint32
// maxfreqkhz:uint32 calcmode:byte amptop:int8 ampbottom:int8 calciter:byte
// mainboard:bool markermode:byte
const presetFrameLen = 33

var errFrameLength = errors.New("frame length does not match its header")

// decode converts a frame, as produced by framer, into a packet. Frames that
// are not recognized are returned as UnhandledPacket. An error is returned
// for a recognized frame that is malformed.
func decode(b []byte) (Packet, error) {
	var pkt Packet
	var err error
	if len(b) >= 2 {
		switch b[0] {
		case '$':
			pkt, err = decodeBinary(b)
		case '#':
			pkt, err = decodeText(b)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("rfx: invalid %q frame: %w", b[:2], err)
	}
	if pkt == nil {
		pkt = &UnhandledPacket{Data: bytes.Clone(b)}
	}
	return pkt, nil
}

func decodeBinary(b []byte) (Packet, error) {
	n, isBinary, ok := binaryFrameLen(b)
	if !isBinary {
		return nil, nil
	}
	if !ok || len(b) != n {
		return nil, errFrameLength
	}
	switch b[1] {
	case 'S', 's':
		return decodeSweep(b[3:]), nil
	case 'z':
		return decodeSweep(b[4:]), nil
	case 'D':
		return &ScreenImage{Data: bytes.Clone(b[2:])}, nil
	case 'R':
		// Raw data (used for sniffer)
		return &RawData{Data: bytes.Clone(b[4:])}, nil
	case 'q':
		return &CalibrationDataPacket{Data: bytes.Clone(b[3:])}, nil
	case 'Q':
		return &CalibrationDataPacket{Data: bytes.Clone(b[4:])}, nil
	case 'P':
		return decodePreset(b)
	}
	return nil, nil
}

// decodeSweep converts sweep samples to dBm. Each sample is an unsigned byte
// holding the amplitude in units of -0.5 dBm, so 0x11 is -8.5 dBm.
func decodeSweep(b []byte) *SweepDataPacket {
	samples := make([]float64, len(b))
	for i, v := range b {
		samples[i] = -float64(v) / 2.0
	}
	return &SweepDataPacket{Samples: samples}
}

func decodePreset(b []byte) (Packet, error) {
	if len(b) < presetFrameLen {
		return nil, errFrameLength
	}
	name := b[5 : 5+12]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	return &Preset{
		Index:          int(b[3]),
		Name:           string(name),
		MinFreqKHz:     int(binary.LittleEndian.Uint32(b[19:23])),
		MaxFreqKHz:     int(binary.LittleEndian.Uint32(b[23:27])),
		CalcMode:       CalculatorMode(b[27]),
		AmpTopDBm:      int(int8(b[28])),
		AmpBottomDBm:   int(int8(b[29])),
		CalcIterations: int(b[30]),
		Mainboard:      b[31] != 0,
		MarkerMode:     MarkerMode(b[32]),
	}, nil
}

func decodeText(b []byte) (Packet, error) {
	s := string(b)
	switch {
	case len(s) == 4 && strings.HasPrefix(s, "#C+"):
		// #C+<mode:byte> - calculator mode
		return &CurrentCalculatorMode{Mode: CalculatorMode(s[3])}, nil
	case strings.HasPrefix(s, "#C2-F:"):
		// #C2-F:<Start_Freq>,<Freq_Step>,<Amp_Top>,<Amp_Bottom>,<Sweep_Steps>,
		//       <ExpModuleActive>,<CurrentMode>,<Min_Freq>,<Max_Freq>,<Max_Span>,
		//       <RBW>,<AmpOffset>,<CalculatorMode>
		// Firmware before v1.12 sends fewer fields.
		f := newFields(s[6:])
		pkt := &CurrentConfigPacket{
			StartFreqKHZ:    f.int(0),
			FreqStepHZ:      f.int(1),
			AmpTopDBM:       f.int(2),
			AmpBottomDBM:    f.int(3),
			SweepSteps:      f.int(4),
			ExpModuleActive: f.str(5) == "1",
			CurrentMode:     Mode(f.int(6)),
			MinFreqKHZ:      f.int(7),
			MaxFreqKHZ:      f.int(8),
			MaxSpan:         f.int(9),
			RBWKHZ:          f.int(10),
			AmpOffset:       f.int(11),
			CalculatorMode:  CalculatorMode(f.int(12)),
		}
		return pkt, f.err
	case strings.HasPrefix(s, "#C2-M:"):
		// #C2-M:<Main_Model>,<Expansion_Model>,<Firmware_Version>
		f := newFields(s[6:])
		pkt := &CurrentSetupPacket{
			Model:           f.model(0),
			ExpansionModel:  f.model(1),
			FirmwareVersion: strings.TrimLeft(f.str(2), "0"),
		}
		return pkt, f.err
	case strings.HasPrefix(s, "#C4-F:"):
		// #C4-F:<Start_Freq>,<ExpModuleActive>,<CurrentMode>,<Delay>,
		//       <Modulation>,<RBW>,<Threshold>
		// The sniffer baud rate is 16 MHz / Delay. Threshold is in units of
		// -0.5 dBm.
		f := newFields(s[6:])
		pkt := &CurrentSnifferConfig{
			StartFreqKHZ:    f.int(0),
			ExpModuleActive: f.str(1) == "1",
			CurrentMode:     Mode(f.int(2)),
			Delay:           f.int(3),
			Modulation:      Modulation(f.int(4)),
			RBWKHZ:          f.int(5),
			ThresholdDBM:    -0.5 * float64(f.int(6)),
		}
		return pkt, f.err
	case strings.HasPrefix(s, "#CAL:"):
		// #CAL:<mainboard:0|1><expansion:0|1>
		if len(s) < 7 {
			return nil, errFrameLength
		}
		return &CalibrationAvailabilityPacket{
			MainboardInternalCalibrationAvailable:      s[5] == '1',
			ExpansionBoardInternalCalibrationAvailable: s[6] == '1',
		}, nil
	case strings.HasPrefix(s, "#Sn"):
		return &SerialNumberPacket{SN: s[3:]}, nil
	case strings.HasPrefix(s, "#PCK"):
		return &EndOfPresetsPacket{}, nil
	}
	return nil, nil
}

// fields parses a comma separated list of values. Fields that are missing or
// empty decode to the zero value. The first parse failure is kept in err.
type fields struct {
	f   []string
	err error
}

func newFields(s string) *fields {
	return &fields{f: strings.Split(s, ",")}
}

func (f *fields) str(i int) string {
	if i >= len(f.f) {
		return ""
	}
	return strings.TrimSpace(f.f[i])
}

func (f *fields) int(i int) int {
	s := f.str(i)
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil && f.err == nil {
		f.err = fmt.Errorf("field %d: %w", i, err)
	}
	return v
}

// model parses a model code. A missing field means no board is present.
func (f *fields) model(i int) Model {
	if f.str(i) == "" {
		return ModelNone
	}
	return Model(f.int(i))
}
