package bands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"samuelks.com/rfx"
)

func TestBuiltinsValid(t *testing.T) {
	names := map[string]bool{}
	for _, bs := range Builtins() {
		if err := bs.Validate(); err != nil {
			t.Error(err)
		}
		if names[bs.Name] {
			t.Errorf("duplicate band set %q", bs.Name)
		}
		names[bs.Name] = true
	}
	for _, n := range []string{NameWiFi24, NameWiFi5, NameFPV58, NameZigbee, NameMURS, NameGMRS} {
		if Builtin(n) == nil {
			t.Errorf("no builtin %q", n)
		}
	}
}

func TestBuiltinsReturnsCopies(t *testing.T) {
	a := Builtins()
	a[0].Name = "changed"
	a[0].Bands[0].Channels[0].Center = 1
	b := Builtin(NameWiFi24)
	if b.Name != NameWiFi24 || b.Bands[0].Channels[0].Center != 2412*rfx.MHz {
		t.Fatal("modifying a returned set changed the builtins")
	}
}

func TestBuiltinValues(t *testing.T) {
	gmrs := Builtin(NameGMRS)
	if ch := gmrs.Bands[0].Channels[1]; ch.Name != "1" || ch.Center != 462_562_500 {
		t.Errorf("GMRS 1 = %+v", ch)
	}
	murs := Builtin(NameMURS)
	if ch := murs.Bands[0].Channels[0]; ch.Width != 11_250 {
		t.Errorf("MURS 1 width = %v", ch.Width)
	}
	if lo, hi := murs.Span(); lo != 151_814_375 || hi != 154_610_000 {
		t.Errorf("MURS span = %v..%v", lo, hi)
	}
	wifi5 := Builtin(NameWiFi5)
	if ch := wifi5.Bands[3].Channels[4]; ch.Name != "165" || ch.Center != 5825*rfx.MHz {
		t.Errorf("WiFi 165 = %+v", ch)
	}
	zigbee := Builtin(NameZigbee)
	chs := zigbee.Bands[0].Channels
	if chs[0].Name != "11" || chs[0].Center != 2405*rfx.MHz || chs[15].Name != "26" || chs[15].Center != 2480*rfx.MHz {
		t.Errorf("Zigbee = %+v .. %+v", chs[0], chs[15])
	}
	if len(Builtin(NameFPV58).Bands) != 10 {
		t.Errorf("FPV bands = %d", len(Builtin(NameFPV58).Bands))
	}
}

func TestChannelEdges(t *testing.T) {
	ch := Channel{Center: 151_820_000, Width: 11_250}
	if ch.Low() != 151_814_375 || ch.High() != 151_825_625 || ch.High()-ch.Low() != ch.Width {
		t.Fatalf("edges %v..%v", ch.Low(), ch.High())
	}
	if !ch.Contains(151_814_375) || !ch.Contains(151_825_625) || ch.Contains(151_825_626) {
		t.Fatal("Contains is not inclusive of exactly the edges")
	}
}

func TestSpanEmpty(t *testing.T) {
	if lo, hi := (&BandSet{}).Span(); lo != 0 || hi != 0 {
		t.Fatalf("empty span %v..%v", lo, hi)
	}
}

func TestValidate(t *testing.T) {
	ok := Channel{Name: "1", Center: rfx.MHz, Width: rfx.KHz}
	tests := []struct {
		name string
		bs   BandSet
		want string
	}{
		{"no set name", BandSet{}, "no name"},
		{"no band name", BandSet{Name: "s", Bands: []Band{{}}}, "band has no name"},
		{"no channel name", BandSet{Name: "s", Bands: []Band{{Name: "b", Channels: []Channel{{Center: 1, Width: 1}}}}}, "channel has no name"},
		{"zero width", BandSet{Name: "s", Bands: []Band{{Name: "b", Channels: []Channel{{Name: "1", Center: 1}}}}}, "width"},
		{"zero center", BandSet{Name: "s", Bands: []Band{{Name: "b", Channels: []Channel{{Name: "1", Width: 1}}}}}, "center"},
		{"valid", BandSet{Name: "s", Bands: []Band{{Name: "b", Channels: []Channel{ok}}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bs.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "bands.json")

	sets, err := LoadFile(path)
	if err != nil || sets != nil {
		t.Fatalf("missing file: %v, %v", sets, err)
	}

	want := []*BandSet{Builtin(NameMURS)}
	if err := SaveFile(path, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"11.25kHz"`) {
		t.Errorf("frequencies not written as text:\n%s", data)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	gj, _ := json.Marshal(got)
	wj, _ := json.Marshal(want)
	if string(gj) != string(wj) {
		t.Fatalf("round trip\n got %s\nwant %s", gj, wj)
	}

	if err := os.WriteFile(path, []byte(`[{"name":"x","bands":[{"name":"b","channels":[{"name":"1","center":"1MHz","width":"0"}]}]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "width") {
		t.Fatalf("invalid file err = %v", err)
	}
	if err := os.WriteFile(path, []byte(`[null]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("null entry accepted")
	}
}

func TestMergeSets(t *testing.T) {
	custom := []*BandSet{{Name: NameMURS}, {Name: "Mine"}}
	merged := MergeSets(Builtins(), custom)
	if Find(merged, NameMURS) != custom[0] || Find(merged, "Mine") != custom[1] || Find(merged, NameGMRS) == nil {
		t.Fatal("merge did not replace or keep sets")
	}
	if len(merged) != len(builtins)+1 {
		t.Fatalf("merged %d sets", len(merged))
	}
}
