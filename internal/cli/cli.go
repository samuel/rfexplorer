// Package cli holds the connection flags shared by the commands.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/record"
)

// Connection describes how to reach the device.
type Connection struct {
	// Dev is the serial device; empty finds it by its USB bridge.
	Dev string
	// Baud is the baud rate; zero tries each of rfx.DetectBaudRates.
	Baud int
	// Record, if set, is a file to record the serial traffic to.
	Record string
	// Replay, if set, is a recording to play back instead of a device.
	Replay string
}

// Register adds the connection flags to fs.
func (c *Connection) Register(fs *flag.FlagSet) {
	fs.StringVar(&c.Dev, "dev", os.Getenv("RFX_DEVICE"), "serial device (default $RFX_DEVICE, else found by its USB bridge)")
	fs.IntVar(&c.Baud, "baud", int(rfx.DefaultBaudRate), "serial baud rate; 0 tries 500000, then 115200")
	fs.StringVar(&c.Record, "record", "", "record the serial traffic to this file")
	fs.StringVar(&c.Replay, "replay", "", "play back a recording made with -record instead of using a device; it loops in real time")
}

// Open connects to the device or recording. timeout bounds connecting.
func (c *Connection) Open(ctx context.Context, timeout time.Duration) (*rfx.Device, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if c.Replay != "" {
		if c.Record != "" {
			return nil, errors.New("-record and -replay can't be combined")
		}
		player, err := record.Open(c.Replay, record.PlayerOptions{Speed: 1, Loop: true})
		if err != nil {
			return nil, err
		}
		return rfx.New(ctx, player)
	}
	opts := []rfx.Option{rfx.WithBaudRate(rfx.BaudRate(c.Baud))}
	if c.Record != "" {
		opts = append(opts, rfx.WithPortWrapper(func(port io.ReadWriteCloser) (io.ReadWriteCloser, error) {
			// Each baud rate attempt starts the file over.
			return record.Create(c.Record, port)
		}))
	}
	dev, err := rfx.Open(ctx, c.Dev, opts...)
	if err != nil {
		name := c.Dev
		if name == "" {
			name = "RF Explorer"
		}
		return nil, fmt.Errorf("connecting to %s: %w", name, err)
	}
	return dev, nil
}
