package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"samuelks.com/rfx"
	"samuelks.com/rfx/internal/cli"
)

// connectTimeout bounds connecting to and configuring the device on startup.
const connectTimeout = 10 * time.Second

func main() {
	var conn cli.Connection
	conn.Register(flag.CommandLine)
	flagMaxHold := flag.Bool("maxhold", false, "switch the device's calculator to max hold on startup")
	flagModule := flag.String("module", "", `select the "main" or "expansion" board on startup`)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	go func() {
		code := 0
		if err := run(logger, &conn, *flagMaxHold, *flagModule); err != nil {
			fmt.Fprintln(os.Stderr, "rfxui:", err)
			code = 1
		}
		os.Exit(code)
	}()
	app.Main()
}

func run(logger *slog.Logger, conn *cli.Connection, maxHold bool, module string) error {
	var startModule *rfx.Module
	switch module {
	case "":
	case "main":
		m := rfx.ModuleMain
		startModule = &m
	case "expansion":
		m := rfx.ModuleExpansion
		startModule = &m
	default:
		return fmt.Errorf("invalid -module %q: want main or expansion", module)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	dev, err := conn.Open(ctx, connectTimeout)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error

	// The device is only changed when asked to.
	if maxHold {
		if err := dev.SetCalculator(ctx, rfx.CalculatorMaxHold); err != nil {
			return fmt.Errorf("setting max hold: %w", err)
		}
	}
	if startModule != nil {
		if _, err := dev.SelectModule(ctx, *startModule); err != nil {
			return fmt.Errorf("selecting %s board: %w", module, err)
		}
	}

	ui := NewUI(dev, logger)
	defer ui.Close()
	w := new(app.Window)
	w.Option(app.Title("RF-Explorer"))
	w.Option(app.Size(unit.Dp(1200), unit.Dp(1024)))
	return ui.Run(w)
}
