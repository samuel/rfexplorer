RF Explorer Handheld Spectrum Analyzer Library and Tools
========================================================

Go library and tools for RF Explorer handheld spectrum analyzers over serial.

Documentation: https://pkg.go.dev/samuelks.com/rfx

- `samuelks.com/rfx` — protocol library (repository root)
- `bands` — channel plans (WiFi, FPV video, MURS, GMRS) and custom band sets
- `cmd/rfxui` — desktop GUI
- `cmd/rfexplorer` — terminal spectrum display

Usage
-----

```bash
go build ./cmd/rfexplorer && ./rfexplorer -dev /dev/ttyUSB0
go build ./cmd/rfxui && ./rfxui -dev /dev/ttyUSB0
```

The serial device is taken from `-dev`, then the `RFX_DEVICE` environment
variable, then `/dev/tty.usbserial-0001`.

Development
-----------

```bash
go test ./...
golangci-lint run ./...
```
