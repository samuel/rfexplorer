package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/internal/cli"
)

const presetsUsage = `Usage: rfexplorer presets <action> [flags] [args]

Actions:
  list                  list the used preset slots
  export [-o file]      write the used presets as JSON (to stdout by default)
  import file           write the presets in a JSON file to the device
  set -slot N ...       write one preset from flags
  delete SLOT...        clear preset slots

Slots are numbered from 1 as on the device. In JSON files "index" is the
slot minus 1, frequencies are strings such as "433.92MHz", and "calculator"
and "marker" are mode names such as "MaxHold" and "Peak".
`

func runPresets(ctx context.Context, fs *flag.FlagSet, conn *cli.Connection, args []string, stdout io.Writer) error {
	action, args := splitCommand(args, "")
	fs.Usage = func() {
		_, _ = fmt.Fprint(fs.Output(), presetsUsage) // usage output is best effort
		fs.PrintDefaults()
	}
	switch action {
	case "list":
		if err := fs.Parse(args); err != nil {
			return err
		}
		return withPresets(ctx, conn, func(ctx context.Context, dev *rfx.Device) error {
			presets, err := usedPresets(ctx, dev)
			if err != nil {
				return err
			}
			return listPresets(stdout, presets)
		})
	case "export":
		out := fs.String("o", "", "output file (default stdout)")
		if err := fs.Parse(args); err != nil {
			return err
		}
		return withPresets(ctx, conn, func(ctx context.Context, dev *rfx.Device) error {
			presets, err := usedPresets(ctx, dev)
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(presets, "", "  ")
			if err != nil {
				return err
			}
			data = append(data, '\n')
			if *out == "" {
				_, err = stdout.Write(data)
				return err
			}
			return os.WriteFile(*out, data, 0o644)
		})
	case "import":
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("import takes one JSON file")
		}
		presets, err := readPresets(fs.Arg(0))
		if err != nil {
			return err
		}
		return withPresets(ctx, conn, func(ctx context.Context, dev *rfx.Device) error {
			return writePresets(ctx, dev, presets, stdout)
		})
	case "set":
		p, err := presetFromFlags(fs, args)
		if err != nil {
			return err
		}
		return withPresets(ctx, conn, func(ctx context.Context, dev *rfx.Device) error {
			return writePresets(ctx, dev, []rfx.Preset{p}, stdout)
		})
	case "delete":
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() == 0 {
			return errors.New("delete takes one or more slot numbers")
		}
		var slots []int
		for _, a := range fs.Args() {
			slot, err := strconv.Atoi(a)
			if err != nil || slot < 1 {
				return fmt.Errorf("invalid slot %q", a)
			}
			slots = append(slots, slot)
		}
		return withPresets(ctx, conn, func(ctx context.Context, dev *rfx.Device) error {
			for _, slot := range slots {
				if err := dev.DeletePreset(ctx, slot-1); err != nil {
					return fmt.Errorf("deleting slot %d: %w", slot, err)
				}
				if _, err := fmt.Fprintf(stdout, "Deleted slot %d\n", slot); err != nil {
					return err
				}
			}
			return nil
		})
	}
	fs.Usage()
	if action == "" {
		return errors.New("missing action")
	}
	return fmt.Errorf("unknown action %q", action)
}

// withPresets connects and runs f with a timeout long enough for every slot.
func withPresets(ctx context.Context, conn *cli.Connection, f func(context.Context, *rfx.Device) error) error {
	dev, err := conn.Open(ctx, connectTimeout)
	if err != nil {
		return err
	}
	defer dev.Close() //nolint:errcheck // exiting, nothing to do with the error
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return f(ctx, dev)
}

func usedPresets(ctx context.Context, dev *rfx.Device) ([]rfx.Preset, error) {
	all, err := dev.Presets(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading presets: %w", err)
	}
	used := []rfx.Preset{}
	for _, p := range all {
		if p.Name != "" {
			used = append(used, p)
		}
	}
	return used, nil
}

func listPresets(w io.Writer, presets []rfx.Preset) error {
	var b strings.Builder
	b.WriteString("SLOT\tNAME\tSTART\tSTOP\tAMPLITUDE\tCALCULATOR\tITER\tBOARD\tMARKER\n")
	for _, p := range presets {
		board := "expansion"
		if p.Mainboard {
			board = "main"
		}
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%d to %d dBm\t%s\t%d\t%s\t%s\n",
			p.Index+1, p.Name, p.Start, p.Stop, p.AmpBottom, p.AmpTop, p.Calculator, p.Iterations, board, p.Marker)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := io.WriteString(tw, b.String()); err != nil {
		return err
	}
	return tw.Flush()
}

// readPresets reads and validates a JSON file of presets.
func readPresets(path string) ([]rfx.Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var presets []rfx.Preset
	if err := json.Unmarshal(data, &presets); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	seen := map[int]bool{}
	for _, p := range presets {
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("preset %q: %w", p.Name, err)
		}
		if seen[p.Index] {
			return nil, fmt.Errorf("slot %d appears twice", p.Index+1)
		}
		seen[p.Index] = true
	}
	return presets, nil
}

// writePresets checks every preset fits the device before writing any.
func writePresets(ctx context.Context, dev *rfx.Device, presets []rfx.Preset, w io.Writer) error {
	for _, p := range presets {
		if err := dev.CheckPreset(p); err != nil {
			return fmt.Errorf("preset %q in slot %d: %w", p.Name, p.Index+1, err)
		}
	}
	for _, p := range presets {
		if err := dev.WritePreset(ctx, p); err != nil {
			return fmt.Errorf("writing slot %d: %w", p.Index+1, err)
		}
		if _, err := fmt.Fprintf(w, "Wrote slot %d: %s\n", p.Index+1, p.Name); err != nil {
			return err
		}
	}
	return nil
}

func presetFromFlags(fs *flag.FlagSet, args []string) (rfx.Preset, error) {
	slot := fs.Int("slot", 0, "slot to write, from 1 (required)")
	name := fs.String("name", "", "name, up to 12 ASCII characters (required)")
	start := fs.String("start", "", "start frequency, such as 433MHz (required)")
	stop := fs.String("stop", "", "stop frequency, such as 435MHz (required)")
	ampTop := fs.Int("amp-top", 0, "top of the amplitude range in dBm")
	ampBottom := fs.Int("amp-bottom", -100, "bottom of the amplitude range in dBm")
	calc := fs.String("calculator", "Normal", "calculator mode: Normal, Max, Average, Overwrite, MaxHold or MaxHistorical")
	iterations := fs.Int("iterations", 4, "calculator iterations, from 1 to 16")
	board := fs.String("board", "main", `RF board: "main" or "expansion"`)
	marker := fs.String("marker", "Peak", "marker mode: Peak, None or Manual")
	if err := fs.Parse(args); err != nil {
		return rfx.Preset{}, err
	}
	if *slot < 1 || *name == "" || *start == "" || *stop == "" {
		fs.Usage()
		return rfx.Preset{}, errors.New("-slot, -name, -start and -stop are required")
	}
	p := rfx.Preset{
		Index:      *slot - 1,
		Name:       *name,
		AmpTop:     *ampTop,
		AmpBottom:  *ampBottom,
		Iterations: *iterations,
	}
	var err error
	if p.Start, err = rfx.ParseFrequency(*start); err != nil {
		return p, err
	}
	if p.Stop, err = rfx.ParseFrequency(*stop); err != nil {
		return p, err
	}
	if err := p.Calculator.UnmarshalText([]byte(*calc)); err != nil {
		return p, err
	}
	if err := p.Marker.UnmarshalText([]byte(*marker)); err != nil {
		return p, err
	}
	switch *board {
	case "main":
		p.Mainboard = true
	case "expansion":
	default:
		return p, fmt.Errorf("invalid -board %q", *board)
	}
	return p, p.Validate()
}
