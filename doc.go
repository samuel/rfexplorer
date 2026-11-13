// Package rfx communicates with RF Explorer handheld spectrum analyzers over
// their serial (UART) interface.
//
// Open a device with [Open], or wrap an existing connection with [New]:
//
//	dev, err := rfx.Open(ctx, "/dev/ttyUSB0")
//	if err != nil {
//		return err
//	}
//	defer dev.Close()
//
// # State
//
// The Device tracks what the analyzer reports. [Device.Info] describes the
// hardware, [Device.Config] is the current frequency and amplitude
// configuration, and [Device.Calibration] is the embedded calibration table.
//
// # Commands
//
// Commands take a context that bounds how long to wait for the device. Those
// that change the configuration, such as [Device.SetSweep] and
// [Device.SelectModule], return the configuration the device reports
// afterwards. A wait that runs out returns an error matching [ErrTimeout].
//
// # Messages
//
// Everything the device sends is delivered to subscribers as a [Message].
// [Device.Subscribe] returns a [Subscription] with its own buffer, so
// subscribers are independent and a slow one never delays the device. When a
// subscriber falls behind, its oldest sweeps are discarded while messages
// carrying device state are kept.
//
// A [Sweep] carries the frequency of its first sample and the step between
// samples, so it can be interpreted without tracking the configuration.
//
// # Units
//
// Frequencies are [Frequency] values in hertz; use the [KHz], [MHz] and [GHz]
// constants, as in 2400*rfx.MHz. Levels are in dBm.
//
// # Hardware notes
//
// The device buffers 64 bytes of commands and handles about one command per
// sweep. Commands are paced to respect this, so sending many in a row takes
// about one sweep period each.
//
// The library has been exercised against a 6G unit with a WSUB3G expansion
// board on firmware 1.34. Signal generator and sniffer modes are decoded but
// not otherwise supported.
package rfx
