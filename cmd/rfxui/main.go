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

// connectTimeout bounds connecting to and configuring the device on startup.
const connectTimeout = 10 * time.Second

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
	defer cancel()
	rfe, err := rfx.Open(ctx, *flagDevice, rfx.WithBaudRate(rfx.BaudRate(*flagBaud)))
	if err != nil {
		log.Fatal(err)
	}
	defer rfe.Close() //nolint:errcheck // exiting, nothing to do with the error

	if err := rfe.SetLCD(ctx, true); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SetScreenDump(ctx, false); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SetCalculator(ctx, rfx.CalculatorMaxHold); err != nil {
		log.Fatal(err)
	}
	if _, err := rfe.SelectModule(ctx, rfx.ModuleExpansion); err != nil {
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
