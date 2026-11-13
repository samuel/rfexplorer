package rfx

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Frequency is a frequency or frequency span in hertz.
type Frequency int64

const (
	Hz  Frequency = 1
	KHz           = 1000 * Hz
	MHz           = 1000 * KHz
	GHz           = 1000 * MHz
)

// MHz returns f in megahertz.
func (f Frequency) MHz() float64 {
	return float64(f) / float64(MHz)
}

// String formats f with the largest unit that keeps the value at 1 or above,
// such as "433.92MHz".
func (f Frequency) String() string {
	unit, name := Hz, "Hz"
	switch a := max(f, -f); {
	case a >= GHz:
		unit, name = GHz, "GHz"
	case a >= MHz:
		unit, name = MHz, "MHz"
	case a >= KHz:
		unit, name = KHz, "kHz"
	}
	return strconv.FormatFloat(float64(f)/float64(unit), 'f', -1, 64) + name
}

// ParseFrequency parses a frequency such as "2.4GHz", "433.92 MHz", "500k"
// or "1000". Units are case insensitive and a bare number is in hertz.
func ParseFrequency(s string) (Frequency, error) {
	num := strings.TrimSpace(s)
	lower := strings.ToLower(num)
	unit := Hz
	for _, u := range []struct {
		suffix string
		unit   Frequency
	}{
		{"ghz", GHz}, {"mhz", MHz}, {"khz", KHz}, {"hz", Hz}, {"g", GHz}, {"m", MHz}, {"k", KHz},
	} {
		if strings.HasSuffix(lower, u.suffix) {
			unit = u.unit
			num = strings.TrimSpace(num[:len(num)-len(u.suffix)])
			break
		}
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("rfx: invalid frequency %q", s)
	}
	return Frequency(math.Round(v * float64(unit))), nil
}

// MarshalText implements encoding.TextMarshaler.
func (f Frequency) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (f *Frequency) UnmarshalText(b []byte) error {
	v, err := ParseFrequency(string(b))
	if err != nil {
		return err
	}
	*f = v
	return nil
}

// MilliwattsToDBm converts power in milliwatts to dBm.
func MilliwattsToDBm(mW float64) float64 {
	return 10.0 * math.Log10(mW)
}

// DBmToMilliwatts converts power in dBm to milliwatts.
func DBmToMilliwatts(dBm float64) float64 {
	return math.Pow(10, dBm/10.0)
}

// WattsToDBm converts power in watts to dBm.
func WattsToDBm(w float64) float64 {
	return 30.0 + MilliwattsToDBm(w)
}

// DBmToWatts converts power in dBm to watts.
func DBmToWatts(dBm float64) float64 {
	return DBmToMilliwatts(dBm) / 1000.0
}

// DBmToDBuV converts dBm to dBµV for a 50 ohm system.
func DBmToDBuV(dBm float64) float64 {
	return dBm + 107.0
}

// DBuVToDBm converts dBµV to dBm for a 50 ohm system.
func DBuVToDBm(dBuV float64) float64 {
	return dBuV - 107.0
}

// FormatSerialNumber returns a 16 character serial number formatted as
// xxxx-xxxx-xxxx-xxxx. Other lengths are returned unchanged.
func FormatSerialNumber(sn string) string {
	if len(sn) != 16 {
		return sn
	}
	return sn[:4] + "-" + sn[4:8] + "-" + sn[8:12] + "-" + sn[12:16]
}
