package rfx

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// The RF Explorer connects through a Silicon Labs CP210x USB to UART bridge.
const (
	cp210xVID = "10C4"
	cp210xPID = "EA60"
)

// ErrNoDevice is returned by FindPorts and Open when no RF Explorer is
// connected.
var ErrNoDevice = errors.New("rfx: no RF Explorer found")

// FindPorts returns the serial ports of connected USB devices that use the
// RF Explorer's CP210x USB bridge. Other devices using the same bridge are
// included too. It returns ErrNoDevice if there are none.
func FindPorts() ([]string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("rfx: listing serial ports: %w", err)
	}
	var names []string
	for _, p := range ports {
		if p.IsUSB && strings.EqualFold(p.VID, cp210xVID) && strings.EqualFold(p.PID, cp210xPID) {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil, ErrNoDevice
	}
	return names, nil
}

// openPort opens a serial port. Tests replace it.
var openPort = openSerial

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
