package rfx

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// maxPresetSlots is the number of preset slots on Plus units, the most
	// any device has.
	maxPresetSlots = 100

	maxPresetNameLen = 12
	maxFreq          = 9999999 * KHz
)

// Validate reports whether the preset can be stored on a device. The index
// is checked against the largest number of slots any device has; WritePreset
// checks it against the connected device.
func (p *Preset) Validate() error {
	if p.Index < 0 || p.Index >= maxPresetSlots {
		return fmt.Errorf("rfx: preset index %d out of range [0,%d]", p.Index, maxPresetSlots-1)
	}
	if p.Name == "" || len(p.Name) > maxPresetNameLen {
		return fmt.Errorf("rfx: preset name must be 1 to %d characters", maxPresetNameLen)
	}
	for i := range len(p.Name) {
		if c := p.Name[i]; c < 0x20 || c > 0x7e {
			return errors.New("rfx: preset name must be printable 7-bit ASCII")
		}
	}
	if err := validateRange(p.Start, p.Stop); err != nil {
		return err
	}
	if err := validateAmplitude(p.AmpTop, p.AmpBottom); err != nil {
		return err
	}
	if p.Iterations < 1 || p.Iterations > 16 {
		return fmt.Errorf("rfx: preset calculator iterations %d out of range [1,16]", p.Iterations)
	}
	if !p.Calculator.valid() {
		return fmt.Errorf("rfx: preset calculator mode %d is invalid", int(p.Calculator))
	}
	if p.Marker > MarkerManual {
		return fmt.Errorf("rfx: preset marker mode %d is invalid", p.Marker)
	}
	return nil
}

// validateRange checks a frequency range. The device works in whole kHz.
func validateRange(start, stop Frequency) error {
	if start < 0 || stop > maxFreq || start/KHz >= stop/KHz {
		return fmt.Errorf("rfx: frequency range %s to %s is invalid", start, stop)
	}
	return nil
}

// validateAmplitude checks a displayed amplitude range in dBm.
func validateAmplitude(top, bottom int) error {
	if top < -110 || top > 35 {
		return fmt.Errorf("rfx: amplitude top %d dBm out of range [-110,35]", top)
	}
	if bottom < -120 || bottom > 25 {
		return fmt.Errorf("rfx: amplitude bottom %d dBm out of range [-120,25]", bottom)
	}
	if top-bottom < 10 {
		return fmt.Errorf("rfx: amplitude top %d dBm must be at least 10 dB above bottom %d dBm", top, bottom)
	}
	return nil
}

// encodePreset returns the command that stores p:
// "#" len:byte "CP" \x01 index:byte name:byte*12 \x00 \x00 minfreqkhz:uint32
// maxfreqkhz:uint32 calcmode:byte amptop:int8 ampbottom:int8 calciter:byte
// mainboard:bool markermode:byte \x42 \x00
func encodePreset(p *Preset) []byte {
	buf := make([]byte, 36)
	buf[0] = '#'
	buf[1] = byte(len(buf))
	buf[2] = 'C'
	buf[3] = 'P'
	buf[4] = 0x01
	buf[5] = byte(p.Index)
	copy(buf[6:6+maxPresetNameLen], p.Name)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(p.Start/KHz))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(p.Stop/KHz))
	buf[28] = byte(p.Calculator)
	buf[29] = byte(int8(p.AmpTop))
	buf[30] = byte(int8(p.AmpBottom))
	buf[31] = byte(p.Iterations)
	if p.Mainboard {
		buf[32] = 1
	}
	buf[33] = byte(p.Marker)
	buf[34] = 0x42
	return buf
}
