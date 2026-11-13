package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"samuelks.com/rfx"
)

// connectTimeout bounds the wait for the device to answer on startup.
const connectTimeout = 5 * time.Second

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

	port, err := rfx.OpenSerial(*flagDevice, rfx.BaudRate(*flagBaud))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	rfe, err := rfx.New(ctx, port)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer rfe.Close() //nolint:errcheck // exiting, nothing to do with the error

	if err := rfe.SetLCDEnabled(true); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SetScreenDumpEnabled(false); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SetMaxHold(); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SwitchModuleExp(); err != nil {
		log.Fatal(err)
	}
	if err := rfe.RequestConfig(); err != nil {
		log.Fatal(err)
	}
	if err := rfe.RequestPresets(); err != nil {
		log.Fatal(err)
	}

	ui := NewUI(rfe)

	go func() {
		w := new(app.Window)
		w.Option(app.Title("RF-Explorer"))
		w.Option(app.Size(unit.Dp(1200), unit.Dp(1024)))
		if err := ui.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	app.Main()
}
