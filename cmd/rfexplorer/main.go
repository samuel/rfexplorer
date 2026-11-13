// Command rfexplorer talks to an RF Explorer spectrum analyzer from the
// terminal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/internal/tui"
)

const (
	// connectTimeout bounds the wait for the device to answer on startup.
	connectTimeout = 5 * time.Second
	// commandTimeout bounds a one-shot command such as a screenshot.
	commandTimeout = 10 * time.Second
)

const usage = `Usage: rfexplorer [command] [flags]

Commands:
  tui         live spectrum display (default)
  info        print the device model, firmware, serial number and configuration
  screenshot  save an image of the device screen as PNG

Run "rfexplorer <command> -h" for the flags of a command.
`

// defaultDevice returns the serial device from RFX_DEVICE, falling back to
// the usual macOS name of the CP210x USB bridge.
func defaultDevice() string {
	if dev := os.Getenv("RFX_DEVICE"); dev != "" {
		return dev
	}
	return "/dev/tty.usbserial-0001"
}

// connection holds the flags shared by all commands.
type connection struct {
	dev  string
	baud int
}

func (c *connection) register(fs *flag.FlagSet) {
	fs.StringVar(&c.dev, "dev", defaultDevice(), "path to serial device (default from RFX_DEVICE)")
	fs.IntVar(&c.baud, "baud", int(rfx.DefaultBaudRate), "serial baud rate")
}

func (c *connection) open(ctx context.Context) (*rfx.Device, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	dev, err := rfx.Open(ctx, c.dev, rfx.WithBaudRate(rfx.BaudRate(c.baud)))
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", c.dev, err)
	}
	return dev, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfexplorer:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := "tui"
	if len(args) > 0 && (args[0] == "" || args[0][0] != '-') {
		cmd, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("rfexplorer "+cmd, flag.ContinueOnError)
	var conn connection
	conn.register(fs)
	switch cmd {
	case "tui":
		logPath := fs.String("log", "", "write device messages and errors to this file")
		if err := fs.Parse(args); err != nil {
			return err
		}
		return runTUI(ctx, &conn, *logPath)
	case "info":
		if err := fs.Parse(args); err != nil {
			return err
		}
		return runInfo(ctx, &conn, os.Stdout)
	case "screenshot":
		out := fs.String("o", "screen.png", "output PNG file")
		if err := fs.Parse(args); err != nil {
			return err
		}
		return runScreenshot(ctx, &conn, *out)
	case "help":
		fmt.Print(usage)
		return nil
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func runTUI(ctx context.Context, conn *connection, logPath string) error {
	var logger *slog.Logger
	if logPath != "" {
		f, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("creating log file: %w", err)
		}
		defer f.Close() //nolint:errcheck // best-effort debug log
		logger = slog.New(slog.NewTextHandler(f, nil))
	}
	dev, err := conn.open(ctx)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error
	return tui.Run(ctx, dev, tui.Options{Logger: logger})
}

func runInfo(ctx context.Context, conn *connection, w io.Writer) error {
	dev, err := conn.open(ctx)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error

	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	serial, err := dev.SerialNumber(ctx)
	if err != nil {
		return fmt.Errorf("reading serial number: %w", err)
	}
	info := dev.Info()
	c := dev.Config()
	var b strings.Builder
	fmt.Fprintf(&b, "Main board:      %s\n", info.Main)
	fmt.Fprintf(&b, "Expansion board: %s\n", info.Expansion)
	fmt.Fprintf(&b, "Firmware:        %s\n", info.Firmware)
	fmt.Fprintf(&b, "Serial number:   %s\n", rfx.FormatSerialNumber(serial))
	fmt.Fprintf(&b, "Active module:   %s (%s)\n", c.Module(), dev.ActiveModel())
	fmt.Fprintf(&b, "Mode:            %s\n", c.Mode)
	fmt.Fprintf(&b, "Sweep:           %s to %s, %d points, step %s\n", c.Start, c.Stop(), c.Points, c.Step)
	fmt.Fprintf(&b, "RBW:             %s\n", c.RBW)
	fmt.Fprintf(&b, "Amplitude:       %d to %d dBm\n", c.AmpBottom, c.AmpTop)
	fmt.Fprintf(&b, "Calculator:      %s\n", c.Calculator)
	fmt.Fprintf(&b, "Board range:     %s to %s, max span %s\n", c.MinFreq, c.MaxFreq, c.MaxSpan)
	_, err = io.WriteString(w, b.String())
	return err
}

func runScreenshot(ctx context.Context, conn *connection, path string) error {
	dev, err := conn.open(ctx)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error

	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	img, err := dev.Screenshot(ctx)
	if err != nil {
		return fmt.Errorf("taking screenshot: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close() // the encode error is the one to report
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return f.Close()
}
