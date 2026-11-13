package rfx

import (
	"encoding/json"
	"math"
	"testing"
)

func TestFrequencyString(t *testing.T) {
	for _, c := range []struct {
		f    Frequency
		want string
	}{
		{0, "0Hz"},
		{12, "12Hz"},
		{5357, "5.357kHz"},
		{433920 * KHz, "433.92MHz"},
		{2400 * MHz, "2.4GHz"},
		{5 * GHz, "5GHz"},
		{855854, "855.854kHz"},
		{-10 * MHz, "-10MHz"},
		{2401500 * KHz, "2.4015GHz"},
	} {
		if got := c.f.String(); got != c.want {
			t.Errorf("%d: got %q, want %q", int64(c.f), got, c.want)
		}
		back, err := ParseFrequency(c.want)
		if err != nil || back != c.f {
			t.Errorf("ParseFrequency(%q) = %d, %v; want %d", c.want, int64(back), err, int64(c.f))
		}
	}
}

func TestParseFrequency(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Frequency
	}{
		{"2.4GHz", 2400 * MHz},
		{"433.92 MHz", 433920 * KHz},
		{" 500k ", 500 * KHz},
		{"1000", 1000},
		{"2412mhz", 2412 * MHz},
		{"5.8G", 5800 * MHz},
		{"0.5m", 500 * KHz},
		{"12.5 kHz", 12500},
		{"11.25kHz", 11250},
	} {
		got, err := ParseFrequency(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseFrequency(%q) = %d, %v; want %d", c.in, int64(got), err, int64(c.want))
		}
	}
	for _, in := range []string{"", "MHz", "abc", "1.2.3MHz", "12 parsecs", "NaN", "InfHz"} {
		if got, err := ParseFrequency(in); err == nil {
			t.Errorf("ParseFrequency(%q) = %d, want an error", in, int64(got))
		}
	}
}

func TestFrequencyJSON(t *testing.T) {
	type channel struct {
		Center Frequency `json:"center"`
	}
	b, err := json.Marshal(channel{Center: 462562500})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"center":"462.5625MHz"}` {
		t.Errorf("got %s", b)
	}
	var c channel
	if err := json.Unmarshal(b, &c); err != nil || c.Center != 462562500 {
		t.Errorf("got %d, %v", int64(c.Center), err)
	}
	if err := json.Unmarshal([]byte(`{"center":"fast"}`), &c); err == nil {
		t.Error("invalid frequency accepted")
	}
}

func TestPowerConversions(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if got := MilliwattsToDBm(1); !near(got, 0) {
		t.Errorf("1 mW = %v dBm", got)
	}
	if got := DBmToMilliwatts(-30); !near(got, 0.001) {
		t.Errorf("-30 dBm = %v mW", got)
	}
	if got := WattsToDBm(1); !near(got, 30) {
		t.Errorf("1 W = %v dBm", got)
	}
	if got := DBmToWatts(30); !near(got, 1) {
		t.Errorf("30 dBm = %v W", got)
	}
	if got := DBuVToDBm(DBmToDBuV(-47)); !near(got, -47) {
		t.Errorf("dBuV round trip = %v", got)
	}
	if got := FormatSerialNumber("0123456789ABCDEF"); got != "0123-4567-89AB-CDEF" {
		t.Errorf("serial %q", got)
	}
}
