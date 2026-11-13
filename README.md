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
variable, then `/dev/tty.usbserial-0001`. The baud rate defaults to 500000 and
can be changed with `-baud`.

Library
-------

```go
port, err := rfx.OpenSerial("/dev/ttyUSB0", rfx.DefaultBaudRate)
if err != nil {
	return err
}
rfe, err := rfx.New(ctx, port) // ctx bounds the wait for the device to answer
if err != nil {
	return err
}
defer rfe.Close()

cfg, err := rfe.SetAnalyzerConfig(ctx, 2400000, 2500000, 0, -120, 0) // kHz, dBm
if err != nil {
	return err
}
for pkt := range rfe.Chan() {
	if sweep, ok := pkt.(*rfx.SweepDataPacket); ok {
		// sweep.Samples[i] is the level in dBm at cfg.StartFreqKHZ*1000 + i*cfg.FreqStepHZ
	}
}
return rfe.Err() // nil if the channel closed because of Close
```

Package `rfxtest` provides an in-memory fake device for testing code that uses
the library without hardware.

Development
-----------

```bash
go test ./...
golangci-lint run ./...
```
