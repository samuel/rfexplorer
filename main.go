package main

// http://j3.rf-explorer.com/menu-the-news/164-rf-explorer-support-for-wifi-analyzer-5ghz-band
// https://en.wikipedia.org/wiki/List_of_WLAN_channels#5.C2.A0GHz_.28802.11a.2Fh.2Fj.2Fn.2Fac.29.5B18.5D

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	termbox "github.com/nsf/termbox-go"
	"github.com/samuel/rfexplorer/rfx"
)

type channel struct {
	name         string
	centerFreqHz int
	widthHZ      int
	note         string
}

var wifi24Channels = []channel{
	{name: "1", centerFreqHz: 2412000000, widthHZ: 20000000},
	{name: "2", centerFreqHz: 2417000000, widthHZ: 20000000},
	{name: "3", centerFreqHz: 2422000000, widthHZ: 20000000},
	{name: "4", centerFreqHz: 2427000000, widthHZ: 20000000},
	{name: "5", centerFreqHz: 2432000000, widthHZ: 20000000},
	{name: "6", centerFreqHz: 2437000000, widthHZ: 20000000},
	{name: "7", centerFreqHz: 2442000000, widthHZ: 20000000},
	{name: "8", centerFreqHz: 2447000000, widthHZ: 20000000},
	{name: "9", centerFreqHz: 2452000000, widthHZ: 20000000},
	{name: "10", centerFreqHz: 2457000000, widthHZ: 20000000},
	{name: "11", centerFreqHz: 2462000000, widthHZ: 20000000},
	{name: "12", centerFreqHz: 2467000000, widthHZ: 20000000},
	{name: "13", centerFreqHz: 2472000000, widthHZ: 20000000},
	{name: "14", centerFreqHz: 2484000000, widthHZ: 20000000},
}

const vtx58ChannelWidth = 10000000

var vtx58Channels = []channel{
	// Band A: Team BlackSheep (TBS), RangeVideo, SpyHawk, FlyCamOne USA
	{name: "A1", centerFreqHz: 5865000000, widthHZ: vtx58ChannelWidth},
	{name: "A2", centerFreqHz: 5845000000, widthHZ: vtx58ChannelWidth},
	{name: "A3", centerFreqHz: 5825000000, widthHZ: vtx58ChannelWidth},
	{name: "A4", centerFreqHz: 5805000000, widthHZ: vtx58ChannelWidth},
	{name: "A5", centerFreqHz: 5785000000, widthHZ: vtx58ChannelWidth},
	{name: "A6", centerFreqHz: 5765000000, widthHZ: vtx58ChannelWidth},
	{name: "A7", centerFreqHz: 5745000000, widthHZ: vtx58ChannelWidth},
	{name: "A8", centerFreqHz: 5725000000, widthHZ: vtx58ChannelWidth},

	// Band B: FlyCamOne Europe
	{name: "B1", centerFreqHz: 5733000000, widthHZ: vtx58ChannelWidth},
	{name: "B2", centerFreqHz: 5752000000, widthHZ: vtx58ChannelWidth},
	{name: "B3", centerFreqHz: 5771000000, widthHZ: vtx58ChannelWidth},
	{name: "B4", centerFreqHz: 5790000000, widthHZ: vtx58ChannelWidth},
	{name: "B5", centerFreqHz: 5809000000, widthHZ: vtx58ChannelWidth},
	{name: "B6", centerFreqHz: 5828000000, widthHZ: vtx58ChannelWidth},
	{name: "B7", centerFreqHz: 5847000000, widthHZ: vtx58ChannelWidth},
	{name: "B8", centerFreqHz: 5866000000, widthHZ: vtx58ChannelWidth},

	// Band E: HobbyKing, Foxtech
	{name: "E1", centerFreqHz: 5705000000, widthHZ: vtx58ChannelWidth},
	{name: "E2", centerFreqHz: 5685000000, widthHZ: vtx58ChannelWidth},
	{name: "E3", centerFreqHz: 5665000000, widthHZ: vtx58ChannelWidth},
	{name: "E4", centerFreqHz: 5645000000, widthHZ: vtx58ChannelWidth},
	{name: "E5", centerFreqHz: 5885000000, widthHZ: vtx58ChannelWidth},
	{name: "E6", centerFreqHz: 5905000000, widthHZ: vtx58ChannelWidth},
	{name: "E7", centerFreqHz: 5925000000, widthHZ: vtx58ChannelWidth},
	{name: "E8", centerFreqHz: 5945000000, widthHZ: vtx58ChannelWidth},

	// Band F (Airwave): ImmersionRC, Iftron
	{name: "F1", centerFreqHz: 5740000000, widthHZ: vtx58ChannelWidth},
	{name: "F2", centerFreqHz: 5760000000, widthHZ: vtx58ChannelWidth},
	{name: "F3", centerFreqHz: 5780000000, widthHZ: vtx58ChannelWidth},
	{name: "F4", centerFreqHz: 5800000000, widthHZ: vtx58ChannelWidth},
	{name: "F5", centerFreqHz: 5820000000, widthHZ: vtx58ChannelWidth},
	{name: "F6", centerFreqHz: 5840000000, widthHZ: vtx58ChannelWidth},
	{name: "F7", centerFreqHz: 5860000000, widthHZ: vtx58ChannelWidth},
	{name: "F8", centerFreqHz: 5880000000, widthHZ: vtx58ChannelWidth},

	// Band C (R): Raceband
	{name: "C1", centerFreqHz: 5658000000, widthHZ: vtx58ChannelWidth},
	{name: "C2", centerFreqHz: 5695000000, widthHZ: vtx58ChannelWidth},
	{name: "C3", centerFreqHz: 5732000000, widthHZ: vtx58ChannelWidth},
	{name: "C4", centerFreqHz: 5769000000, widthHZ: vtx58ChannelWidth},
	{name: "C5", centerFreqHz: 5806000000, widthHZ: vtx58ChannelWidth},
	{name: "C6", centerFreqHz: 5843000000, widthHZ: vtx58ChannelWidth},
	{name: "C7", centerFreqHz: 5880000000, widthHZ: vtx58ChannelWidth},
	{name: "C8", centerFreqHz: 5917000000, widthHZ: vtx58ChannelWidth},

	// Band D: Diatone
	{name: "D1", centerFreqHz: 5362000000, widthHZ: vtx58ChannelWidth},
	{name: "D2", centerFreqHz: 5399000000, widthHZ: vtx58ChannelWidth},
	{name: "D3", centerFreqHz: 5436000000, widthHZ: vtx58ChannelWidth},
	{name: "D4", centerFreqHz: 5473000000, widthHZ: vtx58ChannelWidth},
	{name: "D5", centerFreqHz: 5510000000, widthHZ: vtx58ChannelWidth},
	{name: "D6", centerFreqHz: 5547000000, widthHZ: vtx58ChannelWidth},
	{name: "D7", centerFreqHz: 5584000000, widthHZ: vtx58ChannelWidth},
	{name: "D8", centerFreqHz: 5621000000, widthHZ: vtx58ChannelWidth},

	{name: "U1", centerFreqHz: 5325000000, widthHZ: vtx58ChannelWidth},
	{name: "U2", centerFreqHz: 5348000000, widthHZ: vtx58ChannelWidth},
	{name: "U3", centerFreqHz: 5366000000, widthHZ: vtx58ChannelWidth},
	{name: "U4", centerFreqHz: 5384000000, widthHZ: vtx58ChannelWidth},
	{name: "U5", centerFreqHz: 5402000000, widthHZ: vtx58ChannelWidth},
	{name: "U6", centerFreqHz: 5420000000, widthHZ: vtx58ChannelWidth},
	{name: "U7", centerFreqHz: 5438000000, widthHZ: vtx58ChannelWidth},
	{name: "U8", centerFreqHz: 5456000000, widthHZ: vtx58ChannelWidth},

	{name: "O1", centerFreqHz: 5474000000, widthHZ: vtx58ChannelWidth},
	{name: "O2", centerFreqHz: 5492000000, widthHZ: vtx58ChannelWidth},
	{name: "O3", centerFreqHz: 5510000000, widthHZ: vtx58ChannelWidth},
	{name: "O4", centerFreqHz: 5528000000, widthHZ: vtx58ChannelWidth},
	{name: "O5", centerFreqHz: 5546000000, widthHZ: vtx58ChannelWidth},
	{name: "O6", centerFreqHz: 5564000000, widthHZ: vtx58ChannelWidth},
	{name: "O7", centerFreqHz: 5582000000, widthHZ: vtx58ChannelWidth},
	{name: "O8", centerFreqHz: 5600000000, widthHZ: vtx58ChannelWidth},

	// Band L: Low band
	{name: "L1", centerFreqHz: 5333000000, widthHZ: vtx58ChannelWidth},
	{name: "L2", centerFreqHz: 5373000000, widthHZ: vtx58ChannelWidth},
	{name: "L3", centerFreqHz: 5413000000, widthHZ: vtx58ChannelWidth},
	{name: "L4", centerFreqHz: 5453000000, widthHZ: vtx58ChannelWidth},
	{name: "L5", centerFreqHz: 5493000000, widthHZ: vtx58ChannelWidth},
	{name: "L6", centerFreqHz: 5533000000, widthHZ: vtx58ChannelWidth},
	{name: "L7", centerFreqHz: 5573000000, widthHZ: vtx58ChannelWidth},
	{name: "L8", centerFreqHz: 5613000000, widthHZ: vtx58ChannelWidth},

	// Band H: High band
	{name: "H1", centerFreqHz: 5653000000, widthHZ: vtx58ChannelWidth},
	{name: "H2", centerFreqHz: 5693000000, widthHZ: vtx58ChannelWidth},
	{name: "H3", centerFreqHz: 5733000000, widthHZ: vtx58ChannelWidth},
	{name: "H4", centerFreqHz: 5773000000, widthHZ: vtx58ChannelWidth},
	{name: "H5", centerFreqHz: 5813000000, widthHZ: vtx58ChannelWidth},
	{name: "H6", centerFreqHz: 5853000000, widthHZ: vtx58ChannelWidth},
	{name: "H7", centerFreqHz: 5893000000, widthHZ: vtx58ChannelWidth},
	{name: "H8", centerFreqHz: 5933000000, widthHZ: vtx58ChannelWidth},
}

// var zigbeeChannels = []int{
// 	{name: "11", centerFreqHz: 2405000000, widthHZ: 2000000, note:"Overlaps Ch 1 Newer XBee only"},
// 	{name: "12", centerFreqHz: 2410000000, widthHZ: 2000000, note:"Overlaps Ch 1"},
// 	{name: "13", centerFreqHz: 2415000000, widthHZ: 2000000, note:"Overlaps Ch 1"},
// 	{name: "14", centerFreqHz: 2420000000, widthHZ: 2000000, note:"Overlaps Ch 1"},
// 	{name: "15", centerFreqHz: 2425000000, widthHZ: 2000000, note:"Overlaps Ch 6"},
// 	{name: "16", centerFreqHz: 2430000000, widthHZ: 2000000, note:"Overlaps Ch 6"},
// 	{name: "17", centerFreqHz: 2435000000, widthHZ: 2000000, note:"Overlaps Ch 6"},
// 	{name: "18", centerFreqHz: 2440000000, widthHZ: 2000000, note:"Overlaps Ch 6"},
// 	{name: "19", centerFreqHz: 2445000000, widthHZ: 2000000, note:"Overlaps Ch 6"},
// 	{name: "20", centerFreqHz: 2450000000, widthHZ: 2000000, note:"Overlaps Ch 11"},
// 	{name: "21", centerFreqHz: 2455000000, widthHZ: 2000000, note:"Overlaps Ch 11"},
// 	{name: "22", centerFreqHz: 2460000000, widthHZ: 2000000, note:"Overlaps Ch 11"},
// 	{name: "23", centerFreqHz: 2465000000, widthHZ: 2000000, note:"Overlaps Ch 11"},
// 	{name: "24", centerFreqHz: 2470000000, widthHZ: 2000000, note:"Overlaps Ch 11 Newer XBee only"},
// 	{name: "25", centerFreqHz: 2475000000, widthHZ: 2000000, note:"No Conflict Newer XBee only"},
// 	{name: "26", centerFreqHz: 2480000000, widthHZ: 2000000, note:"No Conflict Newer non-PRO XBee only"},
// }

func main() {
	rfe, err := rfx.New("/dev/tty.SLAB_USBtoUART")
	if err != nil {
		log.Fatal(err)
	}
	defer rfe.Close()

	// if err := rfe.SwitchModuleExp(); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetAnalyzerConfig(2475650, 2501300, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 2.4 GHz Zigbee
	// if err := rfe.SetAnalyzerConfig(2404000, 2481000, 0, -120, 400); err != nil {
	// 	log.Fatal(err)
	// }
	// 2.4 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(2401000, 2495000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetSteps(512); err != nil {
	// 	log.Fatal(err)
	// }
	// Interesting signal
	// if err := rfe.SetAnalyzerConfig(2420000, 2450000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// ISM Band (Region 2)
	// if err := rfe.SetAnalyzerConfig(902000, 928000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 6 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(50000, 54000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 2 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(144000, 148000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 1.25 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(222000, 225000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 70 centimeters
	// if err := rfe.SetAnalyzerConfig(420000, 450000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SwitchModuleMain(); err != nil {
	// 	log.Fatal(err)
	// }
	// 5 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(5170000, 5835000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetAnalyzerConfig(5500000, 5700000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SetAnalyzerConfig(433900, 434100, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	if err := rfe.SetScreenDumpEnabled(false); err != nil {
		log.Fatal(err)
	}

	lcdEnabled := false
	// if err := rfe.SetLCDEnabled(lcdEnabled); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetCalculatorMode(rfx.CalculatorModeAvg); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetScreenDumpEnabled(false); err != nil {
	// 	log.Fatal(err)
	// }
	if err := rfe.RequestConfig(); err != nil {
		log.Fatal(err)
	}
	if err := rfe.RequestPresets(); err != nil {
		log.Fatal(err)
	}

	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	termbox.HideCursor()
	// termbox.SetInputMode(termbox.InputEsc)

	wifi24 := uint32(0)
	vtx85ghz := uint32(0)
	dumpingScreen := uint32(0)

	logFile, err := os.Create("log.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Reset(os.Interrupt, syscall.SIGTERM)
	}()
	go func() {
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEsc:
					select {
					case ch <- os.Signal(nil):
					default:
					}
					return
				case 0:
					switch ev.Ch {
					case 'c':
						if err := rfe.RequestConfig(); err != nil {
							log.Fatal(err)
						}
					case 'h':
						if err := rfe.Hold(); err != nil {
							log.Fatal(err)
						}
					case 'l':
						lcdEnabled = !lcdEnabled
						if err := rfe.SetLCDEnabled(lcdEnabled); err != nil {
							log.Fatal(err)
						}
					case 'm':
						if err := rfe.SetMaxHold(); err != nil {
							log.Fatal(err)
						}
					case 'r':
						if err := rfe.Realtime(); err != nil {
							log.Fatal(err)
						}
					case 's':
						isDumping := atomic.LoadUint32(&dumpingScreen) ^ 1
						atomic.StoreUint32(&dumpingScreen, isDumping)
						if err := rfe.SetScreenDumpEnabled(isDumping != 0); err != nil {
							log.Fatal(err)
						}
					case 'v':
						if atomic.LoadUint32(&vtx85ghz) == 0 {
							if err := rfe.SwitchModuleMain(); err != nil {
								log.Fatal(err)
							}
							if err := rfe.SetAnalyzerConfig(5350000, 5950000, 0, -120, 0); err != nil {
								log.Fatal(err)
							}
							atomic.StoreUint32(&vtx85ghz, 1)
						} else {
							atomic.StoreUint32(&vtx85ghz, 0)
						}
					case 'w':
						if atomic.LoadUint32(&wifi24) == 0 {
							if err := rfe.SetAnalyzerConfig(2401000, 2495000, 0, -120, 0); err != nil {
								log.Fatal(err)
							}
							atomic.StoreUint32(&wifi24, 1)
						} else {
							atomic.StoreUint32(&wifi24, 0)
						}
				}
			}
		}
	}()

	config := &rfx.CurrentConfigPacket{
		StartFreqKHZ: 0,
		FreqStepHZ:   1000,
		AmpTopDBM:    0,
		AmpBottomDBM: -120,
	}
	maxAmp := -999.0
	maxAmpFreq := 0
	maxAmpStep := 0
	var maxSamples []float64
	const numAvg = 0 //2
	var sumSamples []float64
	var sumCount int
	for {
		select {
		case pkt := <-rfe.Chan():
			// fmt.Fprintf(logFile, "%#+v\n", pkt)
			switch pkt := pkt.(type) {
			case *rfx.CurrentConfigPacket:
				fmt.Fprintf(logFile, "%#+v\n", pkt)
				// fmt.Printf("%#+v\n", pkt)
				config = pkt
			case *rfx.SweepDataPacket:
				if atomic.LoadUint32(&dumpingScreen) != 0 {
					break
				}
				if len(pkt.Samples) != len(maxSamples) {
					maxSamples = make([]float64, len(pkt.Samples))
					copy(maxSamples, pkt.Samples)
				}
				if len(pkt.Samples) != len(sumSamples) {
					sumSamples = make([]float64, len(pkt.Samples))
				}
				if numAvg > 0 {
					for i, s := range pkt.Samples {
						sumSamples[i] += s
					}
					sumCount++
					if sumCount < numAvg {
						break
					}
					for i, s := range sumSamples {
						pkt.Samples[i] = s / float64(sumCount)
						sumSamples[i] = 0
					}
					sumCount = 0
					maxAmp = -999.0
					maxAmpFreq = 0
				} else {
					maxAmp = -999
					maxAmpFreq = 0
				}

				if err := termbox.Clear(termbox.ColorWhite, termbox.ColorBlack); err != nil {
					log.Fatal(err)
				}
				width, height := termbox.Size()
				top := 1
				bottom := height - 2
				left := 32
				right := left + len(pkt.Samples)

				// Axis
				for x := left; x < right; x++ {
					termbox.SetCell(x, bottom, '-', termbox.ColorWhite, termbox.ColorBlack)
				}
				for y := top; y < bottom; y++ {
					termbox.SetCell(left-1, y, '|', termbox.ColorWhite, termbox.ColorBlack)
				}
				termbox.SetCell(left-1, bottom, '+', termbox.ColorWhite, termbox.ColorBlack)

				ampToY := func(amp float64) int {
					return top + int(float64(bottom-top)*(amp-float64(config.AmpTopDBM))/float64(config.AmpBottomDBM-config.AmpTopDBM)+0.5)
				}
				// freqToX := func(freqHZ int) int {
				// 	return left + (freqHZ-config.StartFreqKHZ*1000+config.FreqStepHZ/2)/config.FreqStepHZ
				// }

				var channels []channel
				if atomic.LoadUint32(&wifi24) != 0 {
					channels = wifi24Channels
				}

				// if atomic.LoadUint32(&wifi24) != 0 {
				// 	for _, cf := range wifi24Channels {
				// 		x := freqToX(cf.centerFreqHz)
				// 		y := top
				// 		putString(x, y, cf.name, termbox.ColorWhite, termbox.ColorBlack)
				// 		for y++; y < height-1; y++ {
				// 			termbox.SetCell(x, y, '|', termbox.ColorWhite, termbox.ColorBlack)
				// 		}
				// 	}
				// }

				// for i, cf := range zigbeeChannels {
				// 	x := freqToX(cf)
				// 	y := top
				// 	putString(x, y, strconv.Itoa(i+1), termbox.ColorWhite, termbox.ColorBlack)
				// 	for y++; y < height-1; y++ {
				// 		termbox.SetCell(x, y, '|', termbox.ColorWhite, termbox.ColorBlack)
				// 	}
				// }

				if len(channels) == 0 {
					for i, s := range pkt.Samples {
						if s > maxAmp {
							maxAmp = s
							maxAmpFreq = config.StartFreqKHZ*1000 + i*config.FreqStepHZ
							maxAmpStep = i
						}
						y := ampToY(s)
						if numAvg == 0 {
							termbox.SetCell(left+i, y, '.', termbox.ColorWhite, termbox.ColorBlack)
						} else {
							termbox.SetCell(left+i, y, '*', termbox.ColorWhite, termbox.ColorBlack)
						}
						for y++; y < bottom; y++ {
							termbox.SetCell(left+i, y, '.', termbox.ColorWhite, termbox.ColorBlack)
						}
						if numAvg == 0 {
							if s > maxSamples[i] {
								maxSamples[i] = s
							}
							y := ampToY(maxSamples[i])
							termbox.SetCell(left+i, y, '#', termbox.ColorWhite, termbox.ColorBlack)
							const r = '⎟'
							const l = '|'
							if i > 0 {
								if maxSamples[i-1] < maxSamples[i] {
									for y++; y < ampToY(maxSamples[i-1]); y++ {
										termbox.SetCell(left+i-1, y, r, termbox.ColorWhite, termbox.ColorBlack)
									}
								} else if maxSamples[i-1] > maxSamples[i] {
									for y--; y > ampToY(maxSamples[i-1]); y-- {
										termbox.SetCell(left+i, y, l, termbox.ColorWhite, termbox.ColorBlack)
									}
								}
							}
						}
					}
					if atomic.LoadUint32(&vtx85ghz) != 0 {
						var chs []string
						for _, c := range vtx58Channels {
							if maxAmpFreq > c.centerFreqHz-c.widthHZ/2 && maxAmpFreq < c.centerFreqHz+c.widthHZ/2 {
								chs = append(chs, c.name)
							}
						}
						putString(0, bottom-1, strings.Join(chs, ", "), termbox.ColorWhite, termbox.ColorBlack)
					}
				} else {
					chanSums := make([]float64, len(channels))
					chanCounts := make([]float64, len(channels))
					for i, s := range pkt.Samples {
						freq := config.StartFreqKHZ*1000 + i*config.FreqStepHZ
						for i, c := range channels {
							diff := freq - c.centerFreqHz + c.widthHZ/2
							if diff >= 0 && diff <= c.widthHZ {
								d := float64(diff) / float64(c.widthHZ)
								scale := 0.42 - 0.5*math.Cos(2*math.Pi*d) + 0.08*math.Cos(4*math.Pi*d)
								chanSums[i] += s * scale
								chanCounts[i] += scale
							}
						}
					}
					barWidth := (width - left) / len(channels)
					for i, c := range channels {
						startX := left + i*barWidth
						if chanCounts[i] != 0 {
							startY := ampToY(chanSums[i] / float64(chanCounts[i]))
							for x := startX; x < startX+barWidth; x++ {
								termbox.SetCell(x, startY, '-', termbox.ColorWhite, termbox.ColorBlack)
							}
							for y := startY; y < bottom; y++ {
								termbox.SetCell(startX, y, '|', termbox.ColorWhite, termbox.ColorBlack)
								termbox.SetCell(startX+barWidth, y, '|', termbox.ColorWhite, termbox.ColorBlack)
							}
							termbox.SetCell(startX, startY, '+', termbox.ColorWhite, termbox.ColorBlack)
							termbox.SetCell(startX+barWidth, startY, '+', termbox.ColorWhite, termbox.ColorBlack)
						}
						putString(startX+(barWidth+len(c.name))/2, bottom-1, c.name, termbox.ColorWhite, termbox.ColorBlack)
					}
				}

				y := ampToY(maxAmp)
				termbox.SetCell(left+maxAmpStep, y-1, 'V', termbox.ColorWhite, termbox.ColorBlack)
				putString(left+maxAmpStep-2, y-3, fmt.Sprintf("%.3f", float64(maxAmpFreq)/1000000.0),
					termbox.ColorWhite, termbox.ColorBlack)
				putString(left+maxAmpStep-2, y-2, fmt.Sprintf("%.1f", maxAmp),
					termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 0, fmt.Sprintf("CalcMode: %s", config.CalculatorMode), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 1, fmt.Sprintf("MaxSpan: %d", config.MaxSpan), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 2, fmt.Sprintf("MinFreq: %.3f", float64(config.MinFreqKHZ)/1000.0), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 3, fmt.Sprintf("MaxFreq: %.3f", float64(config.MaxFreqKHZ)/1000.0), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 4, fmt.Sprintf("SweepSteps: %d", config.SweepSteps), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 5, fmt.Sprintf("RBW: %d khz", config.RBWKHZ), termbox.ColorWhite, termbox.ColorBlack)

				// Amplitude labels
				s := strconv.Itoa(config.AmpTopDBM)
				putString(left-len(s)-1, top, s, termbox.ColorWhite, termbox.ColorBlack)
				s = strconv.Itoa(config.AmpBottomDBM)
				putString(left-len(s)-1, bottom-1, s, termbox.ColorWhite, termbox.ColorBlack)

				// Frequency labels
				putString(left, bottom+1, fmt.Sprintf("%.3f", float64(config.StartFreqKHZ)/1000.0), termbox.ColorWhite, termbox.ColorBlack)
				s = fmt.Sprintf("%.3f", float64(config.StartFreqKHZ*1000+config.FreqStepHZ*len(pkt.Samples))/1000000.0)
				putString(right-len(s), bottom+1, s, termbox.ColorWhite, termbox.ColorBlack)
				s = fmt.Sprintf("%.3f", float64(config.StartFreqKHZ*1000+config.FreqStepHZ*len(pkt.Samples)/2)/1000000.0)
				putString(left+(right-left)/2-len(s)/2, bottom+1, s, termbox.ColorWhite, termbox.ColorBlack)

				if err := termbox.Flush(); err != nil {
					log.Fatal(err)
				}
			case *rfx.ScreenImage:
				const top = '▀'
				const bottom = '▄'
				for y := pkt.Bounds().Min.Y; y < pkt.Bounds().Max.Y; y += 2 {
					for x := pkt.Bounds().Min.X; x < pkt.Bounds().Max.X; x++ {
						// if pkt.AtGray(x, y).Y == 0 {
						// 	termbox.SetCell(x, y, ' ', termbox.ColorBlack, termbox.ColorBlack)
						// } else {
						// 	termbox.SetCell(x, y, ' ', termbox.ColorWhite, termbox.ColorWhite)
						// }
						t := pkt.AtGray(x, y).Y != 0
						b := pkt.AtGray(x, y+1).Y != 0
						if t && b {
							termbox.SetCell(x, y/2, ' ', termbox.ColorWhite, termbox.ColorWhite)
						} else if t {
							termbox.SetCell(x, y/2, bottom, termbox.ColorBlack, termbox.ColorWhite)
						} else if b {
							termbox.SetCell(x, y/2, top, termbox.ColorBlack, termbox.ColorWhite)
						} else {
							termbox.SetCell(x, y/2, ' ', termbox.ColorBlack, termbox.ColorBlack)
						}
					}
				}
				if err := termbox.Flush(); err != nil {
					log.Fatal(err)
				}
			// case *rfx.CalibrationAvailabilityPacket:
			// case *rfx.SerialNumberPacket:
			// case *rfx.CurrentSetupPacket:
			case *rfx.UnhandledPacket:
				fmt.Fprintf(logFile, "%s\n", hex.Dump(pkt.Data))
			default:
				fmt.Fprintf(logFile, "%#+v\n", pkt)
			}
		case sig := <-ch:
			fmt.Printf("Quitting due to signal %s", sig)
			return
		}
	}
}

func putString(x, y int, s string, fg, bg termbox.Attribute) {
	for i, r := range s {
		termbox.SetCell(x+i, y, r, fg, bg)
	}
}
