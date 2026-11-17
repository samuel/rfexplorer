//go:build hardware

// Smoke test against a connected RF Explorer:
//
//	go test -tags hardware -run Hardware -v .
//
// The device is taken from RFX_DEVICE, or found by its USB bridge. The test
// changes the frequency range and calculator mode and restores them at the
// end. Run it per model and firmware version when changing the protocol.
package rfx_test

import (
	"context"
	"os"
	"testing"
	"time"

	"samuelks.com/rfx"
)

func TestHardware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	openCtx, openCancel := context.WithTimeout(ctx, 10*time.Second)
	dev, err := rfx.Open(openCtx, os.Getenv("RFX_DEVICE"))
	openCancel()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close() //nolint:errcheck // test cleanup

	info := dev.Info()
	serial, err := dev.SerialNumber(ctx)
	if err != nil {
		t.Fatalf("serial number: %s", err)
	}
	orig := dev.Config()
	t.Logf("main %s, expansion %s, firmware %s, serial %s", info.Main, info.Expansion, info.Firmware, serial)
	t.Logf("config %+v", orig)
	if info.Firmware == "" || orig.Points == 0 || orig.Step == 0 {
		t.Fatal("incomplete device state after connecting")
	}

	sub := dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep})
	defer sub.Close()
	waitSweeps := func(n int, cfg rfx.SweepConfig) {
		t.Helper()
		timeout := time.After(15 * time.Second)
		for got := 0; got < n; {
			select {
			case msg := <-sub.C():
				s := msg.(*rfx.Sweep)
				if s.Start != cfg.Start || s.Step != cfg.Step {
					continue // sent before the last change
				}
				if len(s.Samples) != cfg.Points {
					t.Fatalf("sweep has %d points, config says %d", len(s.Samples), cfg.Points)
				}
				got++
			case <-timeout:
				t.Fatalf("timed out waiting for sweeps of %+v", cfg)
			}
		}
	}
	waitSweeps(3, orig)

	defer func() {
		_, err := dev.SetSweep(ctx, rfx.SweepRequest{Start: orig.Start, Stop: orig.Stop(), AmpTop: orig.AmpTop, AmpBottom: orig.AmpBottom})
		if err != nil {
			t.Errorf("restoring range: %s", err)
		}
		if err := dev.SetCalculator(ctx, orig.Calculator); err != nil {
			t.Errorf("restoring calculator: %s", err)
		}
	}()

	// Narrow the range to the middle of the current one.
	span := orig.Span() / 2
	start := orig.Center() - span/2
	cfg, err := dev.SetSweep(ctx, rfx.SweepRequest{Start: start, Stop: start + span, AmpTop: orig.AmpTop, AmpBottom: orig.AmpBottom})
	if err != nil {
		t.Fatalf("set sweep: %s", err)
	}
	if cfg.Start/rfx.KHz != start/rfx.KHz {
		t.Errorf("set sweep start %s, got %s", start, cfg.Start)
	}
	waitSweeps(2, cfg)

	// Sweeping the middle of each board selects it.
	for _, b := range dev.Boards() {
		t.Logf("board %s", b)
		if !b.Known() {
			continue
		}
		span := min(b.MaxSpan, b.Max-b.Min) / 2
		start := (b.Min+b.Max)/2 - span/2
		cfg, err := dev.SetSweep(ctx, rfx.SweepRequest{Start: start, Stop: start + span, AmpTop: orig.AmpTop, AmpBottom: orig.AmpBottom})
		if err != nil {
			t.Errorf("sweep on %s: %s", b.Model, err)
			continue
		}
		if cfg.Module() != b.Module {
			t.Errorf("sweep on %s: %s board active", b.Model, cfg.Module())
		}
		waitSweeps(2, cfg)
		if minFreq, maxFreq, maxSpan, ok := b.Model.Range(); ok && (cfg.MinFreq != minFreq || cfg.MaxFreq != maxFreq || cfg.MaxSpan != maxSpan) {
			t.Logf("%s reports %s to %s, max span %s; published %s to %s, max span %s",
				b.Model, cfg.MinFreq, cfg.MaxFreq, cfg.MaxSpan, minFreq, maxFreq, maxSpan)
		}
	}

	if err := dev.SetCalculator(ctx, rfx.CalculatorMaxHold); err != nil {
		t.Errorf("set calculator: %s", err)
	}

	presets, err := dev.Presets(ctx)
	if err != nil {
		t.Errorf("presets: %s", err)
	}
	t.Logf("%d preset slots", len(presets))

	img, err := dev.Screenshot(ctx)
	if err != nil {
		t.Errorf("screenshot: %s", err)
	} else if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 64 {
		t.Errorf("screenshot is %dx%d", b.Dx(), b.Dy())
	}
}
