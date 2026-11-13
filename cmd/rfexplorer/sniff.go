package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"samuelks.com/rfx"
	"samuelks.com/rfx/internal/cli"
)

func runSniff(ctx context.Context, fs *flag.FlagSet, conn *cli.Connection, args []string, stdout io.Writer) error {
	freq := fs.String("freq", "", "frequency to listen on, such as 433.92MHz (required)")
	rbw := fs.String("rbw", "225kHz", "receiver bandwidth")
	threshold := fs.Float64("threshold", -50, "OOK threshold in dBm, from -127.5 to 0")
	delay := fs.Int("delay", 266, "sniffer sample rate setting (the device's Delay); captures of the reference client used 266")
	format := fs.String("format", "pulses", `output per capture: "pulses" (run lengths, + for high and - for low), "bits" or "hex"`)
	count := fs.Int("n", 0, "stop after this many captures (0 runs until interrupted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *freq == "" {
		fs.Usage()
		return errors.New("-freq is required")
	}
	req := rfx.SnifferRequest{ThresholdDBm: *threshold, Delay: *delay}
	var err error
	if req.Frequency, err = rfx.ParseFrequency(*freq); err != nil {
		return err
	}
	if req.RBW, err = rfx.ParseFrequency(*rbw); err != nil {
		return err
	}
	formatCapture, ok := captureFormats[*format]
	if !ok {
		return fmt.Errorf("unknown -format %q", *format)
	}

	dev, err := conn.Open(ctx, connectTimeout)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error

	sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindRawData, Buffer: 256})
	defer sub.Close()
	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	cfg, err := dev.SetSniffer(cmdCtx, req)
	cancel()
	if errors.Is(err, rfx.ErrNotApplied) {
		return errors.New("the device stayed in analyzer mode; select a sub-GHz board such as the WSUB3G expansion first")
	}
	if err != nil {
		return fmt.Errorf("starting sniffer: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Sniffing at %s, RBW %s, threshold %.1f dBm, delay %d, %s\n",
		cfg.Start, cfg.RBW, cfg.ThresholdDBm, cfg.Delay, cfg.Modulation)
	// The device ignores analyzer commands in sniffer mode; see
	// rfx.Device.SetSniffer.
	fmt.Fprintln(os.Stderr, "The device stays in sniffer mode after exit; return to the analyzer from its menu.")

	return printCaptures(ctx, sub, *count, formatCapture, stdout)
}

func printCaptures(ctx context.Context, sub *rfx.Subscription, count int, format func(*rfx.RawData) string, w io.Writer) error {
	for n := 0; count == 0 || n < count; n++ {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-sub.C():
			if !ok {
				return errors.New("device disconnected")
			}
			raw := msg.(*rfx.RawData)
			if _, err := fmt.Fprintf(w, "%s %d %s\n", raw.Time.Format("15:04:05.000"), raw.Len(), format(raw)); err != nil {
				return err
			}
		}
	}
	return nil
}

var captureFormats = map[string]func(*rfx.RawData) string{
	"pulses": func(r *rfx.RawData) string {
		var b strings.Builder
		for i, p := range r.Pulses() {
			if i > 0 {
				b.WriteByte(' ')
			}
			if p.High {
				b.WriteByte('+')
			} else {
				b.WriteByte('-')
			}
			fmt.Fprint(&b, p.Samples)
		}
		return b.String()
	},
	"bits": func(r *rfx.RawData) string {
		b := make([]byte, r.Len())
		for i := range b {
			b[i] = '0'
			if r.Bit(i) {
				b[i] = '1'
			}
		}
		return string(b)
	},
	"hex": func(r *rfx.RawData) string { return hex.EncodeToString(r.Data) },
}
