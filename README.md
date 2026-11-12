RF Explorer Handheld Spectrum Analyzer Library and Tools
========================================================

Go library and tools for RF Explorer handheld spectrum analyzers over serial.

Documentation: https://pkg.go.dev/github.com/samuel/rfexplorer/rfx

- `rfx` — protocol library
- `rfxbands` — channel plans (WiFi, FPV video, MURS, GMRS) and custom band sets
- `rfxui` — desktop GUI
- `.` — terminal spectrum display

Usage
-----

```bash
go build -o rfexplorer . && ./rfexplorer -dev /dev/ttyUSB0
go build -o rfxui/rfxui ./rfxui && ./rfxui/rfxui -dev /dev/ttyUSB0
```

The serial device is taken from `-dev`, then the `RFX_DEVICE` environment
variable, then `/dev/tty.usbserial-0001`.

Development
-----------

```bash
go test ./...
golangci-lint run ./...
```
