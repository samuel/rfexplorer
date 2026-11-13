RF Explorer Handheld Spectrum Analyzer Library and Tools
========================================================

Go library and tools for RF Explorer handheld spectrum analyzers over serial.

Documentation: https://pkg.go.dev/samuelks.com/rfx

- `samuelks.com/rfx` — protocol library (repository root)
- `bands` — channel plans (WiFi, FPV video, Zigbee, MURS, GMRS) and custom band sets
- `trace` — UI-independent sweep processing: max hold, linear-power averaging
  and channel power, peak search, waterfall images and color gradients
- `record` — record serial traffic to a file and replay it as a stand-in device
- `cmd/rfxui` — desktop GUI
- `cmd/rfexplorer` — terminal tools: spectrum display, device info,
  screenshots, sniffer captures, preset management and a web display

Usage
-----

```bash
go build ./cmd/rfexplorer
./rfexplorer -dev /dev/ttyUSB0                 # terminal spectrum display (same as "tui")
./rfexplorer tui -log rfx.log                  # also log device messages and errors
./rfexplorer info                              # model, firmware, serial number, configuration
./rfexplorer screenshot -o screen.png          # save the device screen
./rfexplorer sniff -freq 433.92MHz             # print sniffer captures as pulse lengths
./rfexplorer presets list                      # also export, import, set and delete
./rfexplorer serve                             # browser display at http://localhost:8080

go build ./cmd/rfxui && ./rfxui -dev /dev/ttyUSB0
```

The serial device is taken from `-dev`, then the `RFX_DEVICE` environment
variable; without either, the first USB serial port using the RF Explorer's
CP210x bridge is used. The baud rate defaults to 500000; `-baud 0` tries
500000 and then 115200, but commands sent at the wrong rate can confuse the
device, so prefer giving the rate.

`-record file` saves the serial traffic of a session and `-replay file`
plays it back in a loop instead of using a device, which is handy for
working on the frontends without hardware. A replayed device doesn't answer
commands.

Neither tool changes the device's settings on startup. `rfxui` can do so on
request: `-maxhold` switches the calculator to max hold and
`-module main|expansion` selects a board.

In the terminal display, `w` shows WiFi 2.4 GHz channel power, `v` names the
5.8 GHz FPV channels at the peak, `m`/`r` switch between max hold and
realtime, `h`/`c` hold and continue, `l` toggles the LCD, `s` mirrors the
device screen, and `q` or Esc quits.

In `rfxui`, Esc closes the preset editor, or the window when the editor is
not open. Device errors are shown at the bottom of the sidebar.

`serve` listens on `localhost:8080` by default (`-addr` changes it). The page
shows the spectrum, max hold and waterfall, and can change the frequency
range and calculator mode and apply presets, so only listen on other
addresses on a trusted network.

`presets` numbers slots from 1 as the device does. Exported JSON uses the
library's `rfx.Preset` form, where `index` is the slot minus 1:

```json
[{"index": 0, "name": "433", "start": "433MHz", "stop": "435MHz", "amp_top": -20,
  "amp_bottom": -100, "calculator": "MaxHold", "iterations": 4, "mainboard": false,
  "marker": "Peak"}]
```

`sniff` switches the device to sniffer mode and prints each capture as runs
of high (`+`) and low (`-`) samples. On a 6G + WSUB3G it only works with the
WSUB3G selected, and the device then ignores analyzer commands: return to the
analyzer from the device's menu. In testing the device stopped answering
after its port was reopened while sniffing, until it was reset from its
menu.

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
  uses the library without hardware, and package `record` replays captured
  sessions.
- `rfx.Open(ctx, "")` finds the device by its USB bridge.

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
verified on hardware. Sniffer mode is experimental (see above). Signal
generators are detected and rejected with `rfx.ErrUnsupported`.

`SetInputStage` and `SetDSP` follow the reference client but have not been
verified on hardware; `SetAmplitudeOffset` has.

To check a device after protocol changes, run the smoke test, which changes
the frequency range and calculator and restores them:

```bash
go test -tags hardware -run Hardware -v .
```
