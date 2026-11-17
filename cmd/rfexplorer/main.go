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
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/internal/cli"
	"samuelks.com/rfx/internal/tui"
	"samuelks.com/rfx/internal/web"
)

const (
	// connectTimeout bounds finding and connecting to the device.
	connectTimeout = 10 * time.Second
	// commandTimeout bounds a one-shot command such as a screenshot.
	commandTimeout = 10 * time.Second
)

const usage = `Usage: rfexplorer [command] [flags]

Commands:
  tui         live spectrum display (default)
  info        print the device model, firmware, serial number and configuration
  screenshot  save an image of the device screen as PNG
  sniff       capture OOK/PSK signals in sniffer mode
  presets     list, export, import, set or delete the device's presets
  serve       serve a spectrum display to web browsers

Every command takes -dev, -baud, -record and -replay. Run
"rfexplorer <command> -h" for the flags of a command.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfexplorer:", err)
		os.Exit(1)
	}
}

// splitCommand returns the leading non-flag argument, or def if there is
// none.
func splitCommand(args []string, def string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return def, args
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	cmd, args := splitCommand(args, "tui")
	fs := flag.NewFlagSet("rfexplorer "+cmd, flag.ContinueOnError)
	var conn cli.Connection
	conn.Register(fs)
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
		return runInfo(ctx, &conn, stdout)
	case "screenshot":
		out := fs.String("o", "screen.png", "output PNG file")
		if err := fs.Parse(args); err != nil {
			return err
		}
		return runScreenshot(ctx, &conn, *out)
	case "sniff":
		return runSniff(ctx, fs, &conn, args, stdout)
	case "presets":
		return runPresets(ctx, fs, &conn, args, stdout)
	case "serve":
		addr := fs.String("addr", "localhost:8080", "address to listen on; the page can change the device's settings, so keep it local unless the network is trusted")
		if err := fs.Parse(args); err != nil {
			return err
		}
		return runServe(ctx, &conn, *addr)
	case "help":
		_, err := fmt.Fprint(stdout, usage)
		return err
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func runTUI(ctx context.Context, conn *cli.Connection, logPath string) error {
	var logger *slog.Logger
	if logPath != "" {
		f, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("creating log file: %w", err)
		}
		defer f.Close() //nolint:errcheck // best-effort debug log
		logger = slog.New(slog.NewTextHandler(f, nil))
	}
	dev, err := conn.Open(ctx, connectTimeout)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error
	return tui.Run(ctx, dev, tui.Options{Logger: logger})
}

func runInfo(ctx context.Context, conn *cli.Connection, w io.Writer) error {
	dev, err := conn.Open(ctx, connectTimeout)
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
	var b strings.Builder
	for _, board := range dev.Boards() {
		source := "published"
		if board.Reported {
			source = "reported"
		}
		label := board.Module.String() + " board:"
		if board.Known() {
			fmt.Fprintf(&b, "%-17s%s, %s to %s, max span %s (%s)\n", label, board.Model, board.Min, board.Max, board.MaxSpan, source)
		} else {
			fmt.Fprintf(&b, "%-17s%s, range unknown\n", label, board.Model)
		}
	}
	fmt.Fprintf(&b, "Firmware:        %s\n", info.Firmware)
	fmt.Fprintf(&b, "Serial number:   %s\n", rfx.FormatSerialNumber(serial))
	if s, ok := dev.Sniffer(); ok {
		fmt.Fprintf(&b, "Mode:            sniffer at %s, RBW %s, threshold %.1f dBm, delay %d\n", s.Start, s.RBW, s.ThresholdDBm, s.Delay)
	} else {
		c := dev.Config()
		fmt.Fprintf(&b, "Active module:   %s (%s)\n", c.Module(), dev.ActiveModel())
		fmt.Fprintf(&b, "Mode:            %s\n", c.Mode)
		fmt.Fprintf(&b, "Sweep:           %s to %s, %d points, step %s\n", c.Start, c.Stop(), c.Points, c.Step)
		fmt.Fprintf(&b, "RBW:             %s\n", c.RBW)
		fmt.Fprintf(&b, "Amplitude:       %d to %d dBm, offset %d dB\n", c.AmpBottom, c.AmpTop, c.AmpOffset)
		fmt.Fprintf(&b, "Calculator:      %s\n", c.Calculator)
	}
	if dev.SupportsInputStage() {
		fmt.Fprintf(&b, "Input stage:     %s\n", dev.InputStage())
	}
	_, err = io.WriteString(w, b.String())
	return err
}

func runScreenshot(ctx context.Context, conn *cli.Connection, path string) error {
	dev, err := conn.Open(ctx, connectTimeout)
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

func runServe(ctx context.Context, conn *cli.Connection, addr string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dev, err := conn.Open(ctx, connectTimeout)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           web.New(dev, logger),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	logger.Info("serving", "url", "http://"+ln.Addr().String())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
