package main

// http://j3.rf-explorer.com/menu-the-news/164-rf-explorer-support-for-wifi-analyzer-5ghz-band
// https://en.wikipedia.org/wiki/List_of_WLAN_channels#5.C2.A0GHz_.28802.11a.2Fh.2Fj.2Fn.2Fac.29.5B18.5D

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	termbox "github.com/nsf/termbox-go"
	"samuelks.com/rfx"
	"samuelks.com/rfx/bands"
	"samuelks.com/rfx/trace"
)

const (
	// connectTimeout bounds the wait for the device to answer on startup.
	connectTimeout = 5 * time.Second
	// configTimeout bounds the wait for the device to apply a new configuration.
	configTimeout = 10 * time.Second
)

// defaultDevice returns the serial device from RFX_DEVICE, falling back to
// the usual macOS name of the CP210x USB bridge.
func defaultDevice() string {
	if dev := os.Getenv("RFX_DEVICE"); dev != "" {
		return dev
	}
	return "/dev/tty.usbserial-0001"
}

func main() {
	flagDevice := flag.String("dev", defaultDevice(), "Path to serial device (default from RFX_DEVICE)")
	flagBaud := flag.Int("baud", int(rfx.DefaultBaudRate), "Serial baud rate")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	rfe, err := rfx.Open(ctx, *flagDevice, rfx.WithBaudRate(rfx.BaudRate(*flagBaud)))
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer rfe.Close() //nolint:errcheck // exiting, nothing to do with the error
	sub := rfe.Subscribe(rfx.SubscribeOptions{})

	var lcdEnabled bool

	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	termbox.HideCursor()

	wifi24 := uint32(0)
	vtx58Enabled := uint32(0)
	dumpingScreen := uint32(0)

	logFile, err := os.Create("log.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close() //nolint:errcheck // best-effort debug log
	logger := slog.New(slog.NewTextHandler(logFile, nil))

	// command runs a device command, logging rather than exiting on failure
	// since the device may simply not have answered in time.
	command := func(name string, f func(ctx context.Context) error) {
		ctx, cancel := context.WithTimeout(context.Background(), configTimeout)
		defer cancel()
		if err := f(ctx); err != nil {
			logger.Error(name, "error", err)
		}
	}
	setRange := func(ctx context.Context, start, stop rfx.Frequency) error {
		_, err := rfe.SetSweep(ctx, rfx.SweepRequest{Start: start, Stop: stop, AmpTop: 0, AmpBottom: -120})
		return err
	}
	command("disable screen dump", func(ctx context.Context) error { return rfe.SetScreenDump(ctx, false) })

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
						command("resume", func(ctx context.Context) error {
							_, err := rfe.Resume(ctx)
							return err
						})
					case 'h':
						command("hold", rfe.Hold)
					case 'l':
						lcdEnabled = !lcdEnabled
						command("set LCD", func(ctx context.Context) error { return rfe.SetLCD(ctx, lcdEnabled) })
					case 'm':
						command("set max hold", func(ctx context.Context) error { return rfe.SetCalculator(ctx, rfx.CalculatorMaxHold) })
					case 'r':
						command("set realtime", func(ctx context.Context) error { return rfe.SetCalculator(ctx, rfx.CalculatorNormal) })
					case 's':
						isDumping := atomic.LoadUint32(&dumpingScreen) ^ 1
						atomic.StoreUint32(&dumpingScreen, isDumping)
						command("set screen dump", func(ctx context.Context) error { return rfe.SetScreenDump(ctx, isDumping != 0) })
					case 'v':
						if atomic.LoadUint32(&vtx58Enabled) == 0 {
							command("select VTX range", func(ctx context.Context) error {
								if _, err := rfe.SelectModule(ctx, rfx.ModuleMain); err != nil {
									return err
								}
								return setRange(ctx, 5350*rfx.MHz, 5950*rfx.MHz)
							})
							atomic.StoreUint32(&vtx58Enabled, 1)
						} else {
							atomic.StoreUint32(&vtx58Enabled, 0)
						}
					case 'w':
						if atomic.LoadUint32(&wifi24) == 0 {
							command("select WiFi range", func(ctx context.Context) error {
								return setRange(ctx, 2401*rfx.MHz, 2495*rfx.MHz)
							})
							atomic.StoreUint32(&wifi24, 1)
						} else {
							atomic.StoreUint32(&wifi24, 0)
						}
					}
				}
			}
		}
	}()

	wifi24Channels := channelsOf(bands.Builtin(bands.NameWiFi24))
	vtx58Channels := channelsOf(bands.Builtin(bands.NameFPV58))

	config := rfe.Config()
	var maxHold trace.MaxHold
	for {
		select {
		case pkt, ok := <-sub.C():
			if !ok {
				termbox.Close()
				fmt.Fprintln(os.Stderr, "Connection lost:", rfe.Err())
				os.Exit(1) //nolint:gocritic // the deferred cleanup is done above or not needed on exit
			}
			switch pkt := pkt.(type) {
			case *rfx.SweepConfig:
				logger.Info("config", "config", fmt.Sprintf("%+v", *pkt))
				config = *pkt
			case *rfx.Sweep:
				if atomic.LoadUint32(&dumpingScreen) != 0 {
					break
				}
				samples := pkt.Samples
				maxHold.Add(pkt)
				held := maxHold.Trace().Samples
				peakIdx, peakAmp := pkt.Peak()

				if err := termbox.Clear(termbox.ColorWhite, termbox.ColorBlack); err != nil {
					log.Fatal(err)
				}
				width, height := termbox.Size()
				top := 1
				bottom := height - 2
				left := 32
				right := left + len(samples)

				// Axis
				for x := left; x < right; x++ {
					termbox.SetCell(x, bottom, '-', termbox.ColorWhite, termbox.ColorBlack)
				}
				for y := top; y < bottom; y++ {
					termbox.SetCell(left-1, y, '|', termbox.ColorWhite, termbox.ColorBlack)
				}
				termbox.SetCell(left-1, bottom, '+', termbox.ColorWhite, termbox.ColorBlack)

				ampToY := func(amp float32) int {
					return top + int(float64(bottom-top)*(float64(amp)-float64(config.AmpTop))/float64(config.AmpBottom-config.AmpTop)+0.5)
				}

				var channels []namedChannel
				if atomic.LoadUint32(&wifi24) != 0 {
					channels = wifi24Channels
				}

				if len(channels) == 0 {
					for i, s := range samples {
						y := ampToY(s)
						termbox.SetCell(left+i, y, '.', termbox.ColorWhite, termbox.ColorBlack)
						for y++; y < bottom; y++ {
							termbox.SetCell(left+i, y, '.', termbox.ColorWhite, termbox.ColorBlack)
						}
						y = ampToY(held[i])
						termbox.SetCell(left+i, y, '#', termbox.ColorWhite, termbox.ColorBlack)
						const r = '⎟'
						const l = '|'
						if i > 0 {
							if held[i-1] < held[i] {
								for y++; y < ampToY(held[i-1]); y++ {
									termbox.SetCell(left+i-1, y, r, termbox.ColorWhite, termbox.ColorBlack)
								}
							} else if held[i-1] > held[i] {
								for y--; y > ampToY(held[i-1]); y-- {
									termbox.SetCell(left+i, y, l, termbox.ColorWhite, termbox.ColorBlack)
								}
							}
						}
					}
					if atomic.LoadUint32(&vtx58Enabled) != 0 && peakIdx >= 0 {
						var chs []string
						for _, c := range vtx58Channels {
							if c.Contains(pkt.FrequencyAt(peakIdx)) {
								chs = append(chs, c.name)
							}
						}
						putString(0, bottom-1, strings.Join(chs, ", "), termbox.ColorWhite, termbox.ColorBlack)
					}
					if peakIdx >= 0 {
						y := ampToY(peakAmp)
						termbox.SetCell(left+peakIdx, y-1, 'V', termbox.ColorWhite, termbox.ColorBlack)
						putString(left+peakIdx-2, y-3, fmt.Sprintf("%.3f", pkt.FrequencyAt(peakIdx).MHz()),
							termbox.ColorWhite, termbox.ColorBlack)
						putString(left+peakIdx-2, y-2, fmt.Sprintf("%.1f", peakAmp),
							termbox.ColorWhite, termbox.ColorBlack)
					}
				} else {
					barWidth := (width - left) / len(channels)
					for i, c := range channels {
						startX := left + i*barWidth
						if p, ok := trace.ChannelPower(pkt, c.Center, c.Width, trace.WindowBlackman); ok {
							startY := ampToY(float32(p))
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

				putString(0, 0, fmt.Sprintf("CalcMode: %s", config.Calculator), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 1, fmt.Sprintf("MaxSpan: %s", config.MaxSpan), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 2, fmt.Sprintf("MinFreq: %.3f", config.MinFreq.MHz()), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 3, fmt.Sprintf("MaxFreq: %.3f", config.MaxFreq.MHz()), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 4, fmt.Sprintf("SweepSteps: %d", config.Points), termbox.ColorWhite, termbox.ColorBlack)
				putString(0, 5, fmt.Sprintf("RBW: %s", config.RBW), termbox.ColorWhite, termbox.ColorBlack)

				// Amplitude labels
				s := strconv.Itoa(config.AmpTop)
				putString(left-len(s)-1, top, s, termbox.ColorWhite, termbox.ColorBlack)
				s = strconv.Itoa(config.AmpBottom)
				putString(left-len(s)-1, bottom-1, s, termbox.ColorWhite, termbox.ColorBlack)

				// Frequency labels
				putString(left, bottom+1, fmt.Sprintf("%.3f", pkt.Start.MHz()), termbox.ColorWhite, termbox.ColorBlack)
				s = fmt.Sprintf("%.3f", pkt.FrequencyAt(len(samples)).MHz())
				putString(right-len(s), bottom+1, s, termbox.ColorWhite, termbox.ColorBlack)
				s = fmt.Sprintf("%.3f", pkt.FrequencyAt(len(samples)/2).MHz())
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
			case *rfx.Unhandled:
				logger.Info("unhandled frame", "data", hex.EncodeToString(pkt.Data))
			default:
				logger.Info("message", "kind", pkt.Kind().String(), "message", fmt.Sprintf("%+v", pkt))
			}
		case sig := <-ch:
			fmt.Printf("Quitting due to signal %s", sig)
			return
		}
	}
}

// namedChannel is a channel labeled with its band, such as "A1".
type namedChannel struct {
	bands.Channel
	name string
}

// channelsOf flattens a band set into channels named band + channel.
func channelsOf(bs *bands.BandSet) []namedChannel {
	var chs []namedChannel
	for _, b := range bs.Bands {
		for _, c := range b.Channels {
			name := c.Name
			if len(bs.Bands) > 1 {
				name = b.Name + c.Name
			}
			chs = append(chs, namedChannel{Channel: c, name: name})
		}
	}
	return chs
}

func putString(x, y int, s string, fg, bg termbox.Attribute) {
	for i, r := range s {
		termbox.SetCell(x+i, y, r, fg, bg)
	}
}
