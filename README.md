RF Explorer Handheld Spectrum Analyzer Library and Tools
========================================================

Go library and tools for RF Explorer handheld spectrum analyzers over serial.

Documentation: https://pkg.go.dev/samuelks.com/rfx

- `samuelks.com/rfx` — protocol library (repository root)
- `bands` — channel plans (WiFi, FPV video, Zigbee, MURS, GMRS) and custom band sets
- `trace` — UI-independent sweep processing: max hold, linear-power averaging
  and channel power, peak search, waterfall images and color gradients
- `cmd/rfxui` — desktop GUI
- `cmd/rfexplorer` — terminal tools: spectrum display, device info, screenshots

Usage
-----

```bash
go build ./cmd/rfexplorer
./rfexplorer -dev /dev/ttyUSB0                 # terminal spectrum display (same as "tui")
./rfexplorer tui -log rfx.log                  # also log device messages and errors
./rfexplorer info                              # model, firmware, serial number, configuration
./rfexplorer screenshot -o screen.png          # save the device screen

go build ./cmd/rfxui && ./rfxui -dev /dev/ttyUSB0
```

The serial device is taken from `-dev`, then the `RFX_DEVICE` environment
variable, then `/dev/tty.usbserial-0001`. The baud rate defaults to 500000 and
can be changed with `-baud`.

Neither tool changes the device's settings on startup. `rfxui` can do so on
request: `-maxhold` switches the calculator to max hold and
`-module main|expansion` selects a board.

In the terminal display, `w` shows WiFi 2.4 GHz channel power, `v` names the
5.8 GHz FPV channels at the peak, `m`/`r` switch between max hold and
realtime, `h`/`c` hold and continue, `l` toggles the LCD, `s` mirrors the
device screen, and `q` or Esc quits.

In `rfxui`, Esc closes the preset editor, or the window when the editor is
not open. Device errors are shown at the bottom of the sidebar.

Library
-------

```go
dev, err := rfx.Open(ctx, "/dev/ttyUSB0") // ctx bounds the wait for the device to answer
if err != nil {
	return err
}
defer dev.Close()

info := dev.Info()
fmt.Println(info.Main, info.Expansion, info.Firmware)

sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep})
defer sub.Close()

cfg, err := dev.SetSweep(ctx, rfx.SweepRequest{
	Start:     2400 * rfx.MHz,
	Stop:      2500 * rfx.MHz,
	AmpTop:    0,
	AmpBottom: -120,
})
if err != nil {
	return err
}
fmt.Println("sweeping", cfg.Start, "to", cfg.Stop())

for msg := range sub.C() {
	sweep := msg.(*rfx.Sweep)
	i, level := sweep.Peak()
	fmt.Printf("peak %.1f dBm at %s\n", level, sweep.FrequencyAt(i))
}
return dev.Err() // nil if the subscription ended because of Close
```

- Commands take a context and return the configuration the device reports.
  A wait that runs out returns an error matching `rfx.ErrTimeout`.
- Each subscription is buffered independently. A slow subscriber loses its
  oldest sweeps, never device state, and never delays the device.
- Frequencies are `rfx.Frequency` values in hertz. Levels are in dBm.
- Package `rfxtest` provides an in-memory fake device for testing code that
  uses the library without hardware.

Custom band sets
----------------

`rfxui` merges the built-in band sets with
`<user config dir>/rfexplorer/bands.json`; a set with the same name as a
built-in replaces it. Frequencies are strings with a unit:

```json
[
  {
    "name": "My radios",
    "bands": [
      {
        "name": "UHF",
        "channels": [{"name": "1", "center": "462.5625MHz", "width": "12.5kHz"}]
      }
    ]
  }
]
```

Every set, band and channel needs a name, and centers and widths must be
positive.

Supported hardware
------------------

Developed against a 6G unit with a WSUB3G expansion board on firmware 1.34.
Other spectrum analyzer models use the same protocol but are untested;
high-resolution sweeps (`$s` and `$z` frames) in particular have not been
verified on hardware. Signal generator and sniffer modes are not supported.
