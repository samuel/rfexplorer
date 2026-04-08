// Package rfxbands defines frequency band and channel data structures for RF
// spectrum analysis, along with built-in band sets for common RF standards.
package rfxbands

// Channel is a single RF channel defined by its center frequency and bandwidth.
type Channel struct {
	Name      string `json:"name"`
	CenterKHz int    `json:"center_khz"`
	WidthKHz  int    `json:"width_khz"`
}

// Band is a named group of channels.
type Band struct {
	Name     string    `json:"name"`
	Channels []Channel `json:"channels"`
}

// BandSet is a named collection of bands covering a theme or standard.
// Bands and channels within a set may overlap in frequency.
type BandSet struct {
	Name  string `json:"name"`
	Bands []Band `json:"bands"`
}

// SpanKHz returns the minimum and maximum frequency extent across all channels
// in all bands. Returns (0, 0) if the BandSet has no channels.
func (bs *BandSet) SpanKHz() (minKHz, maxKHz int) {
	var first bool = true
	for _, band := range bs.Bands {
		for _, ch := range band.Channels {
			lo := ch.CenterKHz - ch.WidthKHz/2
			hi := ch.CenterKHz + ch.WidthKHz/2
			if first {
				minKHz = lo
				maxKHz = hi
				first = false
				continue
			}
			if lo < minKHz {
				minKHz = lo
			}
			if hi > maxKHz {
				maxKHz = hi
			}
		}
	}
	return minKHz, maxKHz
}

// Builtins is the list of built-in band sets provided by the package.
var Builtins = []*BandSet{&WiFi24, &FPV58Video}

// WiFi24 is the 2.4 GHz IEEE 802.11 channel plan (14 channels, 22 MHz wide).
var WiFi24 = BandSet{
	Name: "2.4 GHz WiFi",
	Bands: []Band{
		{
			Name: "WiFi",
			Channels: []Channel{
				{Name: "1", CenterKHz: 2412000, WidthKHz: 22000},
				{Name: "2", CenterKHz: 2417000, WidthKHz: 22000},
				{Name: "3", CenterKHz: 2422000, WidthKHz: 22000},
				{Name: "4", CenterKHz: 2427000, WidthKHz: 22000},
				{Name: "5", CenterKHz: 2432000, WidthKHz: 22000},
				{Name: "6", CenterKHz: 2437000, WidthKHz: 22000},
				{Name: "7", CenterKHz: 2442000, WidthKHz: 22000},
				{Name: "8", CenterKHz: 2447000, WidthKHz: 22000},
				{Name: "9", CenterKHz: 2452000, WidthKHz: 22000},
				{Name: "10", CenterKHz: 2457000, WidthKHz: 22000},
				{Name: "11", CenterKHz: 2462000, WidthKHz: 22000},
				{Name: "12", CenterKHz: 2467000, WidthKHz: 22000},
				{Name: "13", CenterKHz: 2472000, WidthKHz: 22000},
				{Name: "14", CenterKHz: 2484000, WidthKHz: 22000},
			},
		},
	},
}

// FPV58Video is the standard analog FPV 5.8 GHz video band plan (A/B/E/F/R bands,
// 8 channels each, ~26 MHz wide).
var FPV58Video = BandSet{
	Name: "FPV Video (5.8 GHz)",
	Bands: []Band{
		{
			Name: "A",
			Channels: []Channel{
				{Name: "1", CenterKHz: 5865000, WidthKHz: 26000},
				{Name: "2", CenterKHz: 5845000, WidthKHz: 26000},
				{Name: "3", CenterKHz: 5825000, WidthKHz: 26000},
				{Name: "4", CenterKHz: 5805000, WidthKHz: 26000},
				{Name: "5", CenterKHz: 5785000, WidthKHz: 26000},
				{Name: "6", CenterKHz: 5765000, WidthKHz: 26000},
				{Name: "7", CenterKHz: 5745000, WidthKHz: 26000},
				{Name: "8", CenterKHz: 5725000, WidthKHz: 26000},
			},
		},
		{
			Name: "B",
			Channels: []Channel{
				{Name: "1", CenterKHz: 5733000, WidthKHz: 26000},
				{Name: "2", CenterKHz: 5752000, WidthKHz: 26000},
				{Name: "3", CenterKHz: 5771000, WidthKHz: 26000},
				{Name: "4", CenterKHz: 5790000, WidthKHz: 26000},
				{Name: "5", CenterKHz: 5809000, WidthKHz: 26000},
				{Name: "6", CenterKHz: 5828000, WidthKHz: 26000},
				{Name: "7", CenterKHz: 5847000, WidthKHz: 26000},
				{Name: "8", CenterKHz: 5866000, WidthKHz: 26000},
			},
		},
		{
			Name: "E",
			Channels: []Channel{
				{Name: "1", CenterKHz: 5705000, WidthKHz: 26000},
				{Name: "2", CenterKHz: 5685000, WidthKHz: 26000},
				{Name: "3", CenterKHz: 5665000, WidthKHz: 26000},
				{Name: "4", CenterKHz: 5645000, WidthKHz: 26000},
				{Name: "5", CenterKHz: 5885000, WidthKHz: 26000},
				{Name: "6", CenterKHz: 5905000, WidthKHz: 26000},
				{Name: "7", CenterKHz: 5925000, WidthKHz: 26000},
				{Name: "8", CenterKHz: 5945000, WidthKHz: 26000},
			},
		},
		{
			Name: "F",
			Channels: []Channel{
				{Name: "1", CenterKHz: 5740000, WidthKHz: 26000},
				{Name: "2", CenterKHz: 5760000, WidthKHz: 26000},
				{Name: "3", CenterKHz: 5780000, WidthKHz: 26000},
				{Name: "4", CenterKHz: 5800000, WidthKHz: 26000},
				{Name: "5", CenterKHz: 5820000, WidthKHz: 26000},
				{Name: "6", CenterKHz: 5840000, WidthKHz: 26000},
				{Name: "7", CenterKHz: 5860000, WidthKHz: 26000},
				{Name: "8", CenterKHz: 5880000, WidthKHz: 26000},
			},
		},
		{
			Name: "R",
			Channels: []Channel{
				{Name: "1", CenterKHz: 5658000, WidthKHz: 26000},
				{Name: "2", CenterKHz: 5695000, WidthKHz: 26000},
				{Name: "3", CenterKHz: 5732000, WidthKHz: 26000},
				{Name: "4", CenterKHz: 5769000, WidthKHz: 26000},
				{Name: "5", CenterKHz: 5806000, WidthKHz: 26000},
				{Name: "6", CenterKHz: 5843000, WidthKHz: 26000},
				{Name: "7", CenterKHz: 5880000, WidthKHz: 26000},
				{Name: "8", CenterKHz: 5917000, WidthKHz: 26000},
			},
		},
	},
}
