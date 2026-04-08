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
	var first = true
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
var Builtins = []*BandSet{&WiFi24, &WiFi5, &FPV58Video, &MURS, &GMRS}

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

// WiFi5 is the 5 GHz IEEE 802.11 channel plan (20 MHz channels), organized by
// UNII sub-band.
var WiFi5 = BandSet{
	Name: "5 GHz WiFi",
	Bands: []Band{
		{
			Name: "UNII-1",
			Channels: []Channel{
				{Name: "36", CenterKHz: 5180000, WidthKHz: 20000},
				{Name: "40", CenterKHz: 5200000, WidthKHz: 20000},
				{Name: "44", CenterKHz: 5220000, WidthKHz: 20000},
				{Name: "48", CenterKHz: 5240000, WidthKHz: 20000},
			},
		},
		{
			Name: "UNII-2A",
			Channels: []Channel{
				{Name: "52", CenterKHz: 5260000, WidthKHz: 20000},
				{Name: "56", CenterKHz: 5280000, WidthKHz: 20000},
				{Name: "60", CenterKHz: 5300000, WidthKHz: 20000},
				{Name: "64", CenterKHz: 5320000, WidthKHz: 20000},
			},
		},
		{
			Name: "UNII-2C",
			Channels: []Channel{
				{Name: "100", CenterKHz: 5500000, WidthKHz: 20000},
				{Name: "104", CenterKHz: 5520000, WidthKHz: 20000},
				{Name: "108", CenterKHz: 5540000, WidthKHz: 20000},
				{Name: "112", CenterKHz: 5560000, WidthKHz: 20000},
				{Name: "116", CenterKHz: 5580000, WidthKHz: 20000},
				{Name: "120", CenterKHz: 5600000, WidthKHz: 20000},
				{Name: "124", CenterKHz: 5620000, WidthKHz: 20000},
				{Name: "128", CenterKHz: 5640000, WidthKHz: 20000},
				{Name: "132", CenterKHz: 5660000, WidthKHz: 20000},
				{Name: "136", CenterKHz: 5680000, WidthKHz: 20000},
				{Name: "140", CenterKHz: 5700000, WidthKHz: 20000},
				{Name: "144", CenterKHz: 5720000, WidthKHz: 20000},
			},
		},
		{
			Name: "UNII-3",
			Channels: []Channel{
				{Name: "149", CenterKHz: 5745000, WidthKHz: 20000},
				{Name: "153", CenterKHz: 5765000, WidthKHz: 20000},
				{Name: "157", CenterKHz: 5785000, WidthKHz: 20000},
				{Name: "161", CenterKHz: 5805000, WidthKHz: 20000},
				{Name: "165", CenterKHz: 5825000, WidthKHz: 20000},
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

// MURS is the Multi-Use Radio Service VHF channel plan (5 channels).
// Channels 1–3 are 11.25 kHz wide; channels 4–5 are 20 kHz wide.
var MURS = BandSet{
	Name: "MURS",
	Bands: []Band{
		{
			Name: "MURS",
			Channels: []Channel{
				{Name: "1", CenterKHz: 151820, WidthKHz: 11},
				{Name: "2", CenterKHz: 151880, WidthKHz: 11},
				{Name: "3", CenterKHz: 151940, WidthKHz: 11},
				{Name: "4", CenterKHz: 154570, WidthKHz: 20},
				{Name: "5", CenterKHz: 154600, WidthKHz: 20},
			},
		},
	},
}

// GMRS is the General Mobile Radio Service UHF channel plan.
// The 462 MHz band carries main and repeater-output channels (interleaved with
// FRS channels 1–7 and 15–22); the 467 MHz band carries repeater-input
// channels (shared with FRS 8–14). Channels are listed in frequency order and
// are 25 kHz wide.
var GMRS = BandSet{
	Name: "GMRS",
	Bands: []Band{
		{
			Name: "462 MHz",
			Channels: []Channel{
				{Name: "15", CenterKHz: 462550, WidthKHz: 25},
				{Name: "1", CenterKHz: 462563, WidthKHz: 25},
				{Name: "16", CenterKHz: 462575, WidthKHz: 25},
				{Name: "2", CenterKHz: 462588, WidthKHz: 25},
				{Name: "17", CenterKHz: 462600, WidthKHz: 25},
				{Name: "3", CenterKHz: 462613, WidthKHz: 25},
				{Name: "18", CenterKHz: 462625, WidthKHz: 25},
				{Name: "4", CenterKHz: 462638, WidthKHz: 25},
				{Name: "19", CenterKHz: 462650, WidthKHz: 25},
				{Name: "5", CenterKHz: 462663, WidthKHz: 25},
				{Name: "20", CenterKHz: 462675, WidthKHz: 25},
				{Name: "6", CenterKHz: 462688, WidthKHz: 25},
				{Name: "21", CenterKHz: 462700, WidthKHz: 25},
				{Name: "7", CenterKHz: 462713, WidthKHz: 25},
				{Name: "22", CenterKHz: 462725, WidthKHz: 25},
			},
		},
		{
			Name: "467 MHz",
			Channels: []Channel{
				{Name: "8", CenterKHz: 467563, WidthKHz: 25},
				{Name: "9", CenterKHz: 467588, WidthKHz: 25},
				{Name: "10", CenterKHz: 467613, WidthKHz: 25},
				{Name: "11", CenterKHz: 467638, WidthKHz: 25},
				{Name: "12", CenterKHz: 467663, WidthKHz: 25},
				{Name: "13", CenterKHz: 467688, WidthKHz: 25},
				{Name: "14", CenterKHz: 467713, WidthKHz: 25},
			},
		},
	},
}
