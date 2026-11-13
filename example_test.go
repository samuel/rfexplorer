package rfx_test

import (
	"context"
	"fmt"
	"time"

	"samuelks.com/rfx"
	"samuelks.com/rfx/rfxtest"
)

// exampleDevice stands in for an RF Explorer on a serial port.
func exampleDevice() *rfxtest.Device {
	config := "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
	return rfxtest.NewDevice(func(cmd []byte) []byte {
		switch {
		case string(cmd) == "C0":
			return []byte("#C2-M:006,005,01.34\r\n" + config)
		case len(cmd) > 5 && string(cmd[:5]) == "C2-F:":
			config = "#C2-F:0433000,0018018,-010,-110,0112,1,000,0015000,2700000,0600000,00032,0000,000\r\n"
			// One sweep with a carrier at -42.5 dBm on the 51st point.
			sweep := make([]byte, 112)
			for i := range sweep {
				sweep[i] = 200
			}
			sweep[50] = 85
			return append([]byte(config+"$S\x70"), append(sweep, "\r\n"...)...)
		}
		return nil
	})
}

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// With hardware, use rfx.Open(ctx, "/dev/ttyUSB0") instead.
	dev, err := rfx.New(ctx, exampleDevice())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dev.Close() //nolint:errcheck // nothing to do with the error in an example

	info := dev.Info()
	fmt.Printf("%s with %s expansion, firmware %s\n", info.Main, info.Expansion, info.Firmware)

	sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep})
	defer sub.Close()

	cfg, err := dev.SetSweep(ctx, rfx.SweepRequest{
		Start:     433 * rfx.MHz,
		Stop:      435 * rfx.MHz,
		AmpTop:    -10,
		AmpBottom: -110,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("sweeping %s to %s in %d points\n", cfg.Start, cfg.Stop(), cfg.Points)

	msg := <-sub.C()
	sweep := msg.(*rfx.Sweep)
	i, level := sweep.Peak()
	fmt.Printf("peak %.1f dBm at %s\n", level, sweep.FrequencyAt(i))

	// Output:
	// 6G with WSUB3G expansion, firmware 1.34
	// sweeping 433MHz to 434.999998MHz in 112 points
	// peak -42.5 dBm at 433.9009MHz
}
