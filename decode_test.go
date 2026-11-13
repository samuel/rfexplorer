package rfx

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	screen := payload(screenDataLen)
	for _, c := range []struct {
		name  string
		frame []byte
		want  Packet
	}{
		{"sweep", cat("$S", 3, 0x00, 0x11, 0xf0), &SweepDataPacket{Samples: []float64{0, -8.5, -120}}},
		{"sweep ext", cat("$s", 1, bytes.Repeat([]byte{0x64}, 16)), &SweepDataPacket{Samples: repeatFloat(-50, 16)}},
		{"sweep ext 4096", cat("$s", 0, bytes.Repeat([]byte{0x02}, 4096)), &SweepDataPacket{Samples: repeatFloat(-1, 4096)}},
		{"sweep large", cat("$z", 0x01, 0x2c, bytes.Repeat([]byte{0xc8}, 300)), &SweepDataPacket{Samples: repeatFloat(-100, 300)}},
		{"sweep empty", cat("$S", 0), &SweepDataPacket{Samples: []float64{}}},
		{"screen", cat("$D", screen), &ScreenImage{Data: screen}},
		{"raw", cat("$R", 3, 0, "abc"), &RawData{Data: []byte("abc")}},
		{"calibration", cat("$q", 3, 0x02, 0xfe, 0x00), &CalibrationDataPacket{Data: []byte{0x02, 0xfe, 0x00}}},
		{"calibration large", cat("$Q", 3, 0, 0x02, 0xfe, 0x00), &CalibrationDataPacket{Data: []byte{0x02, 0xfe, 0x00}}},
		{"preset", presetFrame, &Preset{
			Index: 0, Name: "RC Remote", MinFreqKHz: 27000, MaxFreqKHz: 28000, AmpTopDBm: -20, AmpBottomDBm: -120,
			CalcMode: CalculatorModeNormal, CalcIterations: 10, Mainboard: false, MarkerMode: MarkerModeNone,
		}},
		{"calculator mode", []byte("#C+\x04"), &CurrentCalculatorMode{Mode: CalculatorModeMaxHold}},
		{"config", []byte("#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000"), &CurrentConfigPacket{
			StartFreqKHZ: 2400000, FreqStepHZ: 855854, AmpTopDBM: 0, AmpBottomDBM: -120, SweepSteps: 112,
			ExpModuleActive: true, CurrentMode: ModeSpectrumAnalyzer, MinFreqKHZ: 15000, MaxFreqKHZ: 2700000,
			MaxSpan: 600000, RBWKHZ: 510, AmpOffset: 0, CalculatorMode: CalculatorModeNormal,
		}},
		{"config with spaces", []byte("#C2-F:5350000, 0005357, -020, -100, 0112, 0, 002, 4850000, 6100000, 0600000, 00110, -010, 004"), &CurrentConfigPacket{
			StartFreqKHZ: 5350000, FreqStepHZ: 5357, AmpTopDBM: -20, AmpBottomDBM: -100, SweepSteps: 112,
			CurrentMode: ModeWIFIAnalyzer, MinFreqKHZ: 4850000, MaxFreqKHZ: 6100000,
			MaxSpan: 600000, RBWKHZ: 110, AmpOffset: -10, CalculatorMode: CalculatorModeMaxHold,
		}},
		{"config old firmware", []byte("#C2-F:0433000,0001000,-050,-120,0112,0,000"), &CurrentConfigPacket{
			StartFreqKHZ: 433000, FreqStepHZ: 1000, AmpTopDBM: -50, AmpBottomDBM: -120, SweepSteps: 112,
		}},
		{"config empty", []byte("#C2-F:"), &CurrentConfigPacket{}},
		{"setup", []byte("#C2-M:006,005,01.34"), &CurrentSetupPacket{Model: Model6G, ExpansionModel: ModelWSUB3G, FirmwareVersion: "1.34"}},
		{"setup no expansion", []byte("#C2-M:003,255,01.12B26"), &CurrentSetupPacket{Model: ModelWSUB1G, ExpansionModel: ModelNone, FirmwareVersion: "1.12B26"}},
		{"setup short", []byte("#C2-M:004"), &CurrentSetupPacket{Model: Model24G, ExpansionModel: ModelNone}},
		{"sniffer config", []byte("#C4-F:0440000,1,006,00266,0,00510,100,"), &CurrentSnifferConfig{
			StartFreqKHZ: 440000, ExpModuleActive: true, CurrentMode: ModeRFSniffer, Delay: 266,
			Modulation: ModulationOOKRaw, RBWKHZ: 510, ThresholdDBM: -50,
		}},
		{"calibration availability", []byte("#CAL:10"), &CalibrationAvailabilityPacket{MainboardInternalCalibrationAvailable: true}},
		{"serial number", []byte("#Sn0123456789ABCDEF"), &SerialNumberPacket{SN: "0123456789ABCDEF"}},
		{"serial number empty", []byte("#Sn"), &SerialNumberPacket{}},
		{"end of presets", []byte("#PCK"), &EndOfPresetsPacket{}},

		{"unknown text", []byte("#QA:0"), &UnhandledPacket{Data: []byte("#QA:0")}},
		{"unknown binary", []byte("$X123"), &UnhandledPacket{Data: []byte("$X123")}},
		{"banner", []byte("RF Explorer 01.34"), &UnhandledPacket{Data: []byte("RF Explorer 01.34")}},
		{"bare marker", []byte("#"), &UnhandledPacket{Data: []byte("#")}},
		{"short text", []byte("#S"), &UnhandledPacket{Data: []byte("#S")}},
		{"calculator mode without value", []byte("#C+"), &UnhandledPacket{Data: []byte("#C+")}},
		{"empty", nil, &UnhandledPacket{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := decode(c.frame)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %T %+v, want %T %+v", got, got, c.want, c.want)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	for _, c := range []struct {
		name  string
		frame []byte
	}{
		{"sweep shorter than count", cat("$S", 5, "AB")},
		{"sweep longer than count", cat("$S", 1, "AB")},
		{"sweep without count", []byte("$S")},
		{"large sweep without count", cat("$z", 1)},
		{"screen short", cat("$D", make([]byte, 100))},
		{"raw short", cat("$R", 9, 0, "abc")},
		{"preset short", cat("$P", 2, "ab")},
		{"calibration short", cat("$q", 9, "abc")},
		{"config not a number", []byte("#C2-F:2400000,abc")},
		{"setup not a number", []byte("#C2-M:x")},
		{"sniffer config not a number", []byte("#C4-F:0440000,1,006,zz")},
		{"calibration availability short", []byte("#CAL:1")},
	} {
		t.Run(c.name, func(t *testing.T) {
			pkt, err := decode(c.frame)
			if err == nil {
				t.Fatalf("got %T %+v, want an error", pkt, pkt)
			}
			if pkt != nil {
				t.Errorf("got packet %T with error %q", pkt, err)
			}
		})
	}
}

// Decoded packets must not alias the frame, which the framer reuses.
func TestDecodeCopies(t *testing.T) {
	frame := cat("$R", 3, 0, "abc")
	pkt, err := decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	clear(frame)
	if got := pkt.(*RawData).Data; string(got) != "abc" {
		t.Errorf("packet data changed to %q", got)
	}
}

func TestCalibrationOffsets(t *testing.T) {
	data := make([]byte, 154)
	data[0] = 0x02   // +1 dB
	data[134] = 0x08 // +4 dB
	data[153] = 0xfa // -3 dB
	for _, c := range []struct {
		name        string
		model       Model
		data        []byte
		n           int
		first, last float64
	}{
		{"6G", Model6G, data, 20, 4, -3},
		{"WSUB3G", ModelWSUB3G, data, 154, 1, -3},
		{"WSUB1G", ModelWSUB1G, data, 43, 0, 0},
		{"WSUB1G Plus short data", ModelWSUB1GPlus, data, 154, 1, -3},
		{"6G short data", Model6G, data[:100], 0, 0, 0},
		{"no data", ModelWSUB3G, nil, 0, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := calibrationOffsetsDB(c.model, c.data)
			if len(got) != c.n {
				t.Fatalf("got %d offsets, want %d", len(got), c.n)
			}
			if c.n == 0 {
				return
			}
			if got[0] != c.first || got[len(got)-1] != c.last {
				t.Errorf("got first %v last %v, want %v and %v", got[0], got[len(got)-1], c.first, c.last)
			}
		})
	}
}

func repeatFloat(v float64, n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func FuzzDecode(f *testing.F) {
	for _, frame := range testFrames() {
		f.Add(frame)
	}
	f.Add([]byte("#C2-F:,,,,,,,,,,,,,,,"))
	f.Add([]byte("#CAL:"))
	f.Add([]byte("$P\x20"))
	f.Fuzz(func(t *testing.T, frame []byte) {
		pkt, err := decode(frame)
		if (pkt == nil) == (err == nil) {
			t.Fatalf("got packet %v and error %v", pkt, err)
		}
	})
}
