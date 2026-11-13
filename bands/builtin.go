package bands

import (
	"fmt"
	"strconv"

	"samuelks.com/rfx"
)

// Names of the built-in band sets.
const (
	NameWiFi24 = "2.4 GHz WiFi"
	NameWiFi5  = "5 GHz WiFi"
	NameFPV58  = "FPV Video (5.8 GHz)"
	NameZigbee = "Zigbee (2.4 GHz)"
	NameMURS   = "MURS"
	NameGMRS   = "GMRS"

	NameBluetooth   = "Bluetooth (2.4 GHz)"
	NameLoRaUS915   = "LoRaWAN US915"
	NameLoRaEU868   = "LoRaWAN EU868"
	NameLPD433      = "LPD433"
	NamePMR446      = "PMR446"
	NameMarineVHF   = "Marine VHF"
	NameNOAAWeather = "NOAA Weather Radio"
	NameAmateurVHF  = "Amateur VHF/UHF (US)"
)

const (
	// wifiWidth is the nominal 802.11 OFDM channel width.
	wifiWidth = 20 * rfx.MHz
	// fpvWidth approximates the occupied bandwidth of an analog FPV video
	// transmitter.
	fpvWidth = 20 * rfx.MHz
	// zigbeeWidth is the 802.15.4 channel width at 2.4 GHz.
	zigbeeWidth = 2 * rfx.MHz
	// fmNarrowWidth is the occupied bandwidth of 25 kHz-spaced FM voice
	// (16K0F3E).
	fmNarrowWidth = 16 * rfx.KHz
)

// Builtins returns a copy of the built-in band sets.
func Builtins() []*BandSet {
	sets := make([]*BandSet, len(builtins))
	for i, bs := range builtins {
		sets[i] = bs.Clone()
	}
	return sets
}

// Builtin returns a copy of the built-in band set named name, or nil if
// there is none.
func Builtin(name string) *BandSet {
	if bs := Find(builtins, name); bs != nil {
		return bs.Clone()
	}
	return nil
}

var builtins = []*BandSet{
	&wifi24, &wifi5, &fpv58, &zigbee, &bluetooth,
	&loraUS915, &loraEU868, &lpd433, &pmr446,
	&murs, &gmrs, &marineVHF, &noaaWeather, &amateurVHF,
}

// spaced returns count channels named first, first+1, ... with centers
// starting at start and step apart.
func spaced(first, count int, start, step, width rfx.Frequency) []Channel {
	chs := make([]Channel, count)
	for i := range chs {
		chs[i] = Channel{
			Name:   strconv.Itoa(first + i),
			Center: start + rfx.Frequency(i)*step,
			Width:  width,
		}
	}
	return chs
}

// numbered returns channels named "1", "2", ... at the given centers in MHz.
func numbered(width rfx.Frequency, centersMHz ...int) []Channel {
	chs := make([]Channel, len(centersMHz))
	for i, c := range centersMHz {
		chs[i] = Channel{Name: strconv.Itoa(i + 1), Center: rfx.Frequency(c) * rfx.MHz, Width: width}
	}
	return chs
}

// wifi24 is the 2.4 GHz IEEE 802.11 channel plan.
var wifi24 = BandSet{
	Name: NameWiFi24,
	Bands: []Band{
		{Name: "WiFi", Channels: numbered(wifiWidth,
			2412, 2417, 2422, 2427, 2432, 2437, 2442, 2447, 2452, 2457, 2462, 2467, 2472, 2484)},
	},
}

// wifi5 is the 5 GHz IEEE 802.11 channel plan (20 MHz channels), organized by
// UNII sub-band. Channel numbers follow f = 5000 MHz + 5 MHz × n. Channel
// 169 straddles the UNII-3/UNII-4 boundary at 5850 MHz.
var wifi5 = BandSet{
	Name: NameWiFi5,
	Bands: []Band{
		{Name: "UNII-1", Channels: wifi5Channels(36, 40, 44, 48)},
		{Name: "UNII-2A", Channels: wifi5Channels(52, 56, 60, 64)},
		{Name: "UNII-2C", Channels: wifi5Channels(100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144)},
		{Name: "UNII-3", Channels: wifi5Channels(149, 153, 157, 161, 165)},
		{Name: "UNII-4", Channels: wifi5Channels(169, 173, 177)},
	},
}

func wifi5Channels(numbers ...int) []Channel {
	chs := make([]Channel, len(numbers))
	for i, n := range numbers {
		chs[i] = Channel{Name: strconv.Itoa(n), Center: 5000*rfx.MHz + rfx.Frequency(n)*5*rfx.MHz, Width: wifiWidth}
	}
	return chs
}

// fpv58 is the analog FPV 5.8 GHz video channel plan.
var fpv58 = BandSet{
	Name: NameFPV58,
	Bands: []Band{
		// Team BlackSheep, RangeVideo, SpyHawk, FlyCamOne USA
		{Name: "A", Channels: numbered(fpvWidth, 5865, 5845, 5825, 5805, 5785, 5765, 5745, 5725)},
		// FlyCamOne Europe
		{Name: "B", Channels: numbered(fpvWidth, 5733, 5752, 5771, 5790, 5809, 5828, 5847, 5866)},
		// HobbyKing, Foxtech
		{Name: "E", Channels: numbered(fpvWidth, 5705, 5685, 5665, 5645, 5885, 5905, 5925, 5945)},
		// Airwave: ImmersionRC, Iftron
		{Name: "F", Channels: numbered(fpvWidth, 5740, 5760, 5780, 5800, 5820, 5840, 5860, 5880)},
		// Raceband
		{Name: "R", Channels: numbered(fpvWidth, 5658, 5695, 5732, 5769, 5806, 5843, 5880, 5917)},
		// Diatone
		{Name: "D", Channels: numbered(fpvWidth, 5362, 5399, 5436, 5473, 5510, 5547, 5584, 5621)},
		{Name: "U", Channels: numbered(fpvWidth, 5325, 5348, 5366, 5384, 5402, 5420, 5438, 5456)},
		{Name: "O", Channels: numbered(fpvWidth, 5474, 5492, 5510, 5528, 5546, 5564, 5582, 5600)},
		// Low band
		{Name: "L", Channels: numbered(fpvWidth, 5333, 5373, 5413, 5453, 5493, 5533, 5573, 5613)},
		// High band
		{Name: "H", Channels: numbered(fpvWidth, 5653, 5693, 5733, 5773, 5813, 5853, 5893, 5933)},
	},
}

// zigbee is the IEEE 802.15.4 2.4 GHz channel plan (channels 11–26).
var zigbee = BandSet{
	Name: NameZigbee,
	Bands: []Band{
		{Name: "802.15.4", Channels: spaced(11, 16, 2405*rfx.MHz, 5*rfx.MHz, zigbeeWidth)},
	},
}

// murs is the Multi-Use Radio Service VHF channel plan. Channels 1–3 are
// 11.25 kHz wide; channels 4–5 are 20 kHz wide.
var murs = BandSet{
	Name: NameMURS,
	Bands: []Band{
		{
			Name: "MURS",
			Channels: []Channel{
				{Name: "1", Center: 151_820_000, Width: 11_250},
				{Name: "2", Center: 151_880_000, Width: 11_250},
				{Name: "3", Center: 151_940_000, Width: 11_250},
				{Name: "4", Center: 154_570_000, Width: 20_000},
				{Name: "5", Center: 154_600_000, Width: 20_000},
			},
		},
	},
}

// gmrs is the General Mobile Radio Service UHF channel plan in frequency
// order. The 462 MHz band carries the main and interstitial channels shared
// with FRS 1–7 and 15–22 (20 kHz). The 467 MHz band carries the interstitial
// channels shared with FRS 8–14 (12.5 kHz).
var gmrs = BandSet{
	Name: NameGMRS,
	Bands: []Band{
		{
			Name: "462 MHz",
			Channels: []Channel{
				{Name: "15", Center: 462_550_000, Width: 20_000},
				{Name: "1", Center: 462_562_500, Width: 20_000},
				{Name: "16", Center: 462_575_000, Width: 20_000},
				{Name: "2", Center: 462_587_500, Width: 20_000},
				{Name: "17", Center: 462_600_000, Width: 20_000},
				{Name: "3", Center: 462_612_500, Width: 20_000},
				{Name: "18", Center: 462_625_000, Width: 20_000},
				{Name: "4", Center: 462_637_500, Width: 20_000},
				{Name: "19", Center: 462_650_000, Width: 20_000},
				{Name: "5", Center: 462_662_500, Width: 20_000},
				{Name: "20", Center: 462_675_000, Width: 20_000},
				{Name: "6", Center: 462_687_500, Width: 20_000},
				{Name: "21", Center: 462_700_000, Width: 20_000},
				{Name: "7", Center: 462_712_500, Width: 20_000},
				{Name: "22", Center: 462_725_000, Width: 20_000},
			},
		},
		{
			Name: "467 MHz",
			Channels: []Channel{
				{Name: "8", Center: 467_562_500, Width: 12_500},
				{Name: "9", Center: 467_587_500, Width: 12_500},
				{Name: "10", Center: 467_612_500, Width: 12_500},
				{Name: "11", Center: 467_637_500, Width: 12_500},
				{Name: "12", Center: 467_662_500, Width: 12_500},
				{Name: "13", Center: 467_687_500, Width: 12_500},
				{Name: "14", Center: 467_712_500, Width: 12_500},
			},
		},
	},
}

// bluetooth is the 2.4 GHz Bluetooth channel plan. Bluetooth LE uses 40
// channels at 2402 + 2k MHz; the advertising channels 37, 38 and 39 sit at
// 2402, 2426 and 2480 MHz and the data channels fill the gaps. Bluetooth
// Classic hops over 79 channels at 2402 + k MHz.
var bluetooth = BandSet{
	Name: NameBluetooth,
	Bands: []Band{
		{Name: "LE advertising", Channels: []Channel{
			{Name: "37", Center: 2402 * rfx.MHz, Width: 2 * rfx.MHz},
			{Name: "38", Center: 2426 * rfx.MHz, Width: 2 * rfx.MHz},
			{Name: "39", Center: 2480 * rfx.MHz, Width: 2 * rfx.MHz},
		}},
		{Name: "LE data", Channels: append(
			spaced(0, 11, 2404*rfx.MHz, 2*rfx.MHz, 2*rfx.MHz),
			spaced(11, 26, 2428*rfx.MHz, 2*rfx.MHz, 2*rfx.MHz)...)},
		{Name: "Classic", Channels: spaced(0, 79, 2402*rfx.MHz, rfx.MHz, rfx.MHz)},
	},
}

// loraUS915 is the LoRaWAN US902-928 channel plan. Uplink channels 0–63 are
// 125 kHz wide, 64–71 are 500 kHz wide; the eight downlink channels are
// 500 kHz wide.
var loraUS915 = BandSet{
	Name: NameLoRaUS915,
	Bands: []Band{
		{Name: "Uplink 125 kHz", Channels: spaced(0, 64, 902_300*rfx.KHz, 200*rfx.KHz, 125*rfx.KHz)},
		{Name: "Uplink 500 kHz", Channels: spaced(64, 8, 903_000*rfx.KHz, 1600*rfx.KHz, 500*rfx.KHz)},
		{Name: "Downlink", Channels: spaced(0, 8, 923_300*rfx.KHz, 600*rfx.KHz, 500*rfx.KHz)},
	},
}

// loraEU868 is the LoRaWAN EU863-870 channel plan: the three mandatory
// channels, the five additional channels used by The Things Network, and the
// RX2 downlink channel.
var loraEU868 = BandSet{
	Name: NameLoRaEU868,
	Bands: []Band{
		{Name: "Default", Channels: spaced(0, 3, 868_100*rfx.KHz, 200*rfx.KHz, 125*rfx.KHz)},
		{Name: "Additional", Channels: spaced(3, 5, 867_100*rfx.KHz, 200*rfx.KHz, 125*rfx.KHz)},
		{Name: "RX2", Channels: []Channel{{Name: "RX2", Center: 869_525 * rfx.KHz, Width: 125 * rfx.KHz}}},
	},
}

// lpd433 is the European low-power device plan in the 433 MHz ISM band:
// 69 channels 25 kHz apart from 433.075 MHz.
var lpd433 = BandSet{
	Name: NameLPD433,
	Bands: []Band{
		{Name: "LPD433", Channels: spaced(1, 69, 433_075*rfx.KHz, 25*rfx.KHz, 25*rfx.KHz)},
	},
}

// pmr446 is the European license-free analog PMR446 plan: 16 channels
// 12.5 kHz apart from 446.00625 MHz.
var pmr446 = BandSet{
	Name: NamePMR446,
	Bands: []Band{
		{Name: "PMR446", Channels: spaced(1, 16, 446_006_250, 12_500, 12_500)},
	},
}

// marineVHF is the international maritime VHF plan. The ship band holds the
// ship transmit frequency of channels 60, 01, 61, 02 ... 28, 88, 25 kHz
// apart from 156.025 MHz; duplex channels have their coast transmit
// frequency 4.6 MHz higher.
var marineVHF = BandSet{
	Name: NameMarineVHF,
	Bands: []Band{
		{Name: "Ship", Channels: func() []Channel {
			chs := make([]Channel, 57)
			for i := range chs {
				n := 60 + i/2
				if i%2 == 1 {
					n = 1 + i/2
				}
				chs[i] = Channel{
					Name:   fmt.Sprintf("%02d", n),
					Center: 156_025*rfx.KHz + rfx.Frequency(i)*25*rfx.KHz,
					Width:  fmNarrowWidth,
				}
			}
			return chs
		}()},
		{Name: "AIS", Channels: []Channel{
			{Name: "AIS 1", Center: 161_975 * rfx.KHz, Width: 25 * rfx.KHz},
			{Name: "AIS 2", Center: 162_025 * rfx.KHz, Width: 25 * rfx.KHz},
		}},
	},
}

// noaaWeather is the NOAA Weather Radio plan: seven channels 25 kHz apart
// from 162.400 MHz, numbered in the order used by most receivers.
var noaaWeather = BandSet{
	Name: NameNOAAWeather,
	Bands: []Band{
		{Name: "WX", Channels: []Channel{
			{Name: "WX1", Center: 162_550 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX2", Center: 162_400 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX3", Center: 162_475 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX4", Center: 162_425 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX5", Center: 162_450 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX6", Center: 162_500 * rfx.KHz, Width: fmNarrowWidth},
			{Name: "WX7", Center: 162_525 * rfx.KHz, Width: fmNarrowWidth},
		}},
	},
}

// amateurVHF covers the US amateur 2 m, 1.25 m and 70 cm allocations, each
// as one channel, plus the North American APRS frequency.
var amateurVHF = BandSet{
	Name: NameAmateurVHF,
	Bands: []Band{
		{Name: "2 m", Channels: []Channel{{Name: "2 m", Center: 146 * rfx.MHz, Width: 4 * rfx.MHz}}},
		{Name: "1.25 m", Channels: []Channel{{Name: "1.25 m", Center: 223_500 * rfx.KHz, Width: 3 * rfx.MHz}}},
		{Name: "70 cm", Channels: []Channel{{Name: "70 cm", Center: 435 * rfx.MHz, Width: 30 * rfx.MHz}}},
		{Name: "APRS", Channels: []Channel{{Name: "APRS", Center: 144_390 * rfx.KHz, Width: fmNarrowWidth}}},
	},
}
