package rfx

import (
	"fmt"
	"io"

	"go.bug.st/serial"
)

// openSerial opens the serial device an RF Explorer is connected to.
func openSerial(device string, baud BaudRate) (io.ReadWriteCloser, error) {
	port, err := serial.Open(device, &serial.Mode{
		BaudRate: int(baud),
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		return nil, fmt.Errorf("rfx: failed to open %s: %w", device, err)
	}
	return port, nil
}
