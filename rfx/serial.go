//go:build linux || darwin

package rfx

import (
	"io"

	"github.com/jacobsa/go-serial/serial"
)

func OpenSerial(device string) (io.ReadWriteCloser, error) {
	// TODO: currently a baud rate of 500,000 is assumed.
	options := serial.OpenOptions{
		PortName:        device,
		BaudRate:        500000,
		DataBits:        8,
		ParityMode:      serial.PARITY_NONE,
		StopBits:        1,
		MinimumReadSize: 1,
	}
	return serial.Open(options)
}
