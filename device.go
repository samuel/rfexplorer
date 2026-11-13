package rfx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"
)

var (
	// ErrClosed is returned by operations on a closed Device.
	ErrClosed = errors.New("rfx: closed")
	// ErrTimeout is returned when the device doesn't answer in time. The
	// error also matches context.DeadlineExceeded.
	ErrTimeout = errors.New("rfx: timed out waiting for the device")
	// ErrUnsupported is returned for operations the connected model doesn't
	// support.
	ErrUnsupported = errors.New("rfx: not supported by this device")
	// ErrNotApplied is returned when the device answers but did not make the
	// requested change. The configuration returned with it is the actual one.
	ErrNotApplied = errors.New("rfx: the device did not apply the change")
)

// configEchoTimeout is how long to wait for the device to report a changed
// configuration before asking for it. Echoes arrive within about 200 ms.
var configEchoTimeout = 1500 * time.Millisecond

// Option configures Open.
type Option func(*options)

type options struct {
	baud BaudRate
}

// WithBaudRate sets the serial baud rate used by Open. The default is
// DefaultBaudRate.
func WithBaudRate(baud BaudRate) Option {
	return func(o *options) { o.baud = baud }
}

// Device is a connection to an RF Explorer. Its methods are safe for
// concurrent use.
type Device struct {
	port       io.ReadWriteCloser
	closeCh    chan struct{} // closed by Close
	readerDone chan struct{} // closed when the reader exits
	waiters    waiters
	wg         sync.WaitGroup
	closeOnce  sync.Once
	closeErr   error

	// reqMu serializes commands that wait for a response.
	reqMu sync.Mutex
	// txMu serializes writes to the port.
	txMu sync.Mutex
	// txReady is signaled when the device may have room for another command.
	txReady chan struct{}

	flowMu sync.Mutex
	flow   txWindow

	subsMu sync.Mutex
	subs   []*Subscription
	closed bool

	mu          sync.RWMutex
	err         error
	info        DeviceInfo
	config      SweepConfig
	calibration []byte
	screenDump  bool
}

// Open connects to the RF Explorer on the serial device at path. See New.
func Open(ctx context.Context, path string, opts ...Option) (*Device, error) {
	o := options{baud: DefaultBaudRate}
	for _, opt := range opts {
		opt(&o)
	}
	port, err := openSerial(path, o.baud)
	if err != nil {
		return nil, err
	}
	return New(ctx, port)
}

// New starts communicating with the RF Explorer connected to port and waits
// for it to report its configuration. ctx bounds that wait. A device that is
// not in an analyzer mode may not respond. New takes ownership of port and
// closes it on failure.
func New(ctx context.Context, port io.ReadWriteCloser) (*Device, error) {
	d := &Device{
		port:       port,
		closeCh:    make(chan struct{}),
		readerDone: make(chan struct{}),
		txReady:    make(chan struct{}, 1),
		info:       DeviceInfo{Main: ModelNone, Expansion: ModelNone},
	}
	d.wg.Add(1)
	go d.readLoop()

	_, err := d.Resume(ctx)
	if err == nil {
		// The reply is tracked by the reader; see Calibration.
		err = d.SendRaw(ctx, []byte("Cq"))
	}
	if err != nil {
		_ = d.Close() // returning the original error
		return nil, fmt.Errorf("rfx: no response from device: %w", err)
	}
	return d, nil
}

// Close stops communication and closes the port. Subscriptions are closed
// once Close returns.
func (d *Device) Close() error {
	d.closeOnce.Do(func() {
		d.subsMu.Lock()
		d.closed = true
		d.subsMu.Unlock()
		close(d.closeCh)
		d.closeErr = d.port.Close()
		d.wg.Wait()
	})
	return d.closeErr
}

// Err returns the error that stopped reading from the port, or nil if reading
// is still active or was stopped by Close.
func (d *Device) Err() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.err
}

// Info returns what is known about the device.
func (d *Device) Info() DeviceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.info
}

// Config returns the configuration last reported by the device.
func (d *Device) Config() SweepConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// ActiveModel returns the model of the selected RF board.
func (d *Device) ActiveModel() Model {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.activeModelLocked()
}

func (d *Device) activeModelLocked() Model {
	if d.config.ExpansionActive {
		return d.info.Expansion
	}
	return d.info.Main
}

// Calibration returns the embedded amplitude calibration offsets in dB for
// the selected RF board, or nil if the device hasn't reported any. The
// device applies its calibration itself; the offsets are informational.
func (d *Device) Calibration() []float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return calibrationOffsetsDB(d.activeModelLocked(), d.calibration)
}

// SupportsHighRes reports whether the number of sweep points can be changed
// in the current mode.
func (d *Device) SupportsHighRes() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.config.Mode == ModeWifiAnalyzer {
		return false
	}
	switch d.activeModelLocked() {
	case ModelWSUB1GPlus, Model4GPlus, Model6GPlus, ModelAudioPro, ModelW5G3G, ModelW5G4G, ModelW5G5G:
		return true
	}
	return d.info.isMWSUB3G()
}

// SupportsInputStage reports whether the selected RF board has a selectable
// input stage.
func (d *Device) SupportsInputStage() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	switch d.activeModelLocked() {
	case ModelWSUB1GPlus, Model2400Plus, Model4GPlus, Model6GPlus, ModelAudioPro:
		return true
	}
	return d.info.isMWSUB3G()
}

// SweepRequest is a frequency and amplitude range to configure.
type SweepRequest struct {
	Start Frequency
	Stop  Frequency
	// AmpTop and AmpBottom are the displayed amplitude range in dBm. They
	// don't affect measurements.
	AmpTop    int
	AmpBottom int
	// RBW is the resolution bandwidth. Zero leaves it to the device.
	RBW Frequency
}

// SetSweep changes the frequency range and displayed amplitude range and
// returns the configuration the device reports afterwards, which may differ
// from the request.
func (d *Device) SetSweep(ctx context.Context, req SweepRequest) (SweepConfig, error) {
	// #<Size>C2-F:<Start_Freq>,<End_Freq>,<Amp_Top>,<Amp_Bottom>[,<RBW_KHZ>]
	// <Start_Freq>, <End_Freq> = 7 ascii digits, decimal kHz
	// <Amp_Top>, <Amp_Bottom> = 4 ascii digits, decimal
	// <RBW_KHZ> = 5 ascii digits, decimal
	if err := validateRange(req.Start, req.Stop); err != nil {
		return SweepConfig{}, err
	}
	if err := validateAmplitude(req.AmpTop, req.AmpBottom); err != nil {
		return SweepConfig{}, err
	}
	startKHz, stopKHz := int(req.Start/KHz), int(req.Stop/KHz)

	var rbw string
	if rbwKHz := int(req.RBW / KHz); rbwKHz >= 3 && rbwKHz <= 670 {
		points := min(max((stopKHz-startKHz+rbwKHz/2)/rbwKHz, 112), MaxSweepPoints)
		rbwKHz = (stopKHz - startKHz + points/2) / points
		if rbwKHz >= 3 && rbwKHz < 620 {
			rbw = fmt.Sprintf(",%05d", rbwKHz)
		}
	}

	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	cfg := d.Config()
	// The device reports a step size rather than the stop frequency.
	stop := int(cfg.Stop() / KHz)
	changed := rbw != "" || int(cfg.Start/KHz) != startKHz || stop < stopKHz-1 || stop > stopKHz+1 ||
		cfg.AmpTop != req.AmpTop || cfg.AmpBottom != req.AmpBottom
	cmd := fmt.Sprintf("C2-F:%07d,%07d,%04d,%04d%s", startKHz, stopKHz, req.AmpTop, req.AmpBottom, rbw)
	return d.configCommand(ctx, []byte(cmd), changed, false, func(c *SweepConfig) bool {
		return int(c.Start/KHz) == startKHz
	})
}

// SelectModule switches between the main and expansion RF boards and returns
// the configuration of the selected board.
func (d *Device) SelectModule(ctx context.Context, m Module) (SweepConfig, error) {
	if m > ModuleExpansion {
		return SweepConfig{}, fmt.Errorf("rfx: invalid module %d", m)
	}
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	cfg := d.Config()
	return d.configCommand(ctx, []byte{'C', 'M', byte(m)}, cfg.Module() != m, true, func(c *SweepConfig) bool {
		return c.Module() == m
	})
}

// SetWifiMode enables or disables the WiFi analyzer. While it is enabled the
// device ignores SetSweep and SetCalculator. Leaving WiFi mode does not
// restore the previous frequency range or calculator mode.
func (d *Device) SetWifiMode(ctx context.Context, mode WifiMode) (SweepConfig, error) {
	if mode > Wifi5GHz {
		return SweepConfig{}, fmt.Errorf("rfx: invalid WiFi mode %d", mode)
	}
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	enabled := d.Config().Mode == ModeWifiAnalyzer
	// The mode is an ASCII digit. Firmware 1.34 ignores a zero byte, so the
	// raw value can't disable the analyzer.
	return d.configCommand(ctx, []byte{'C', 'W', '0' + byte(mode)}, enabled != (mode != WifiDisabled), true, func(c *SweepConfig) bool {
		return (c.Mode == ModeWifiAnalyzer) == (mode != WifiDisabled)
	})
}

// SetSweepPoints changes the number of points in a sweep, from 112 to
// MaxSweepPoints. It returns ErrUnsupported on models with a fixed
// resolution.
func (d *Device) SetSweepPoints(ctx context.Context, points int) (SweepConfig, error) {
	if points < 112 || points > MaxSweepPoints {
		return SweepConfig{}, fmt.Errorf("rfx: sweep points %d out of range [112,%d]", points, MaxSweepPoints)
	}
	if !d.SupportsHighRes() {
		return SweepConfig{}, ErrUnsupported
	}
	// CJ takes a count of 16-point blocks up to 4096 points; Cj takes any
	// count.
	cmd := []byte{'C', 'j', byte(points >> 8), byte(points)}
	if points <= 4096 && points%16 == 0 {
		cmd = []byte{'C', 'J', byte((points - 16) / 16)}
	}
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	cfg := d.Config()
	return d.configCommand(ctx, cmd, cfg.Points != points, true, func(c *SweepConfig) bool {
		return c.Points == points
	})
}

// Resume asks the device for its configuration, which also restarts sweeps
// after Hold.
func (d *Device) Resume(ctx context.Context) (SweepConfig, error) {
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	return d.requestConfig(ctx)
}

// Hold stops the device from sending sweeps until Resume.
func (d *Device) Hold(ctx context.Context) error {
	return d.SendRaw(ctx, []byte("CH"))
}

// SetCalculator changes how the device combines consecutive sweeps.
func (d *Device) SetCalculator(ctx context.Context, mode CalculatorMode) error {
	if !mode.valid() {
		return fmt.Errorf("rfx: invalid calculator mode %d", int(mode))
	}
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	_, err := d.request(ctx, []byte{'C', '+', byte(mode)}, func(msg Message) bool {
		c, ok := msg.(*CalculatorChange)
		return ok && c.Mode == mode
	})
	return err
}

// Presets returns the presets stored on the device, including unused slots,
// which have an empty name.
func (d *Device) Presets(ctx context.Context) ([]Preset, error) {
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	var presets []Preset
	_, err := d.request(ctx, []byte("CP\x00"), func(msg Message) bool {
		switch msg := msg.(type) {
		case *Preset:
			presets = append(presets, *msg)
		case *PresetsEnd:
			return true
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	return presets, nil
}

// WritePreset stores a preset on the device and waits for it to be
// acknowledged.
func (d *Device) WritePreset(ctx context.Context, p Preset) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := d.checkPresetIndex(p.Index); err != nil {
		return err
	}
	return d.writePreset(ctx, &p)
}

// DeletePreset clears the preset slot at index and waits for it to be
// acknowledged.
func (d *Device) DeletePreset(ctx context.Context, index int) error {
	if err := d.checkPresetIndex(index); err != nil {
		return err
	}
	// A slot with an empty name is unused.
	return d.writePreset(ctx, &Preset{Index: index})
}

func (d *Device) checkPresetIndex(index int) error {
	info := d.Info()
	if n := info.MaxPresets(); index < 0 || index >= n {
		return fmt.Errorf("rfx: preset index %d out of range [0,%d]", index, n-1)
	}
	return nil
}

func (d *Device) writePreset(ctx context.Context, p *Preset) error {
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	w := d.waiters.add(func(msg Message) bool { return msg.Kind() == KindPresetsEnd })
	if err := d.write(ctx, encodePreset(p)); err != nil {
		d.waiters.remove(w)
		return err
	}
	_, err := d.await(ctx, w)
	return err
}

// Screenshot returns an image of the device's LCD.
func (d *Device) Screenshot(ctx context.Context) (*ScreenImage, error) {
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	d.mu.RLock()
	dumping := d.screenDump
	d.mu.RUnlock()
	isScreen := func(msg Message) bool { return msg.Kind() == KindScreen }
	if dumping {
		w := d.waiters.add(isScreen)
		msg, err := d.await(ctx, w)
		if err != nil {
			return nil, err
		}
		return msg.(*ScreenImage), nil
	}
	msg, err := d.request(ctx, []byte("D1"), isScreen)
	// Stop the dump even if no screen arrived, without the expired ctx.
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commandTimeout)
	defer cancel()
	if stopErr := d.SendRaw(stopCtx, []byte("D0")); err == nil {
		err = stopErr
	}
	if err != nil {
		return nil, err
	}
	return msg.(*ScreenImage), nil
}

// SetScreenDump makes the device send an image of its LCD after every sweep,
// delivered as ScreenImage messages.
func (d *Device) SetScreenDump(ctx context.Context, enabled bool) error {
	cmd := "D0"
	if enabled {
		cmd = "D1"
	}
	if err := d.SendRaw(ctx, []byte(cmd)); err != nil {
		return err
	}
	d.mu.Lock()
	d.screenDump = enabled
	d.mu.Unlock()
	return nil
}

// SetLCD turns the device's LCD on or off. Sweeps are faster with it off.
func (d *Device) SetLCD(ctx context.Context, enabled bool) error {
	cmd := "L0"
	if enabled {
		cmd = "L1"
	}
	return d.SendRaw(ctx, []byte(cmd))
}

// SerialNumber returns the device serial number.
func (d *Device) SerialNumber(ctx context.Context) (string, error) {
	if sn := d.Info().Serial; sn != "" {
		return sn, nil
	}
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	msg, err := d.request(ctx, []byte("Cn"), func(msg Message) bool { return msg.Kind() == KindSerialNumber })
	if err != nil {
		return "", err
	}
	return msg.(*SerialNumber).Serial, nil
}

// SetTrackingStep requests the device to jump to a tracking step frequency
// and measure it immediately.
func (d *Device) SetTrackingStep(ctx context.Context, step uint16) error {
	return d.SendRaw(ctx, []byte{'k', byte(step >> 8), byte(step)})
}

// SetBaudRate changes the baud rate of the device. The port must be reopened
// at the new rate afterwards.
func (d *Device) SetBaudRate(ctx context.Context, baud BaudRate) error {
	i := slices.Index([]BaudRate{
		BaudRate500000, BaudRate1200, BaudRate2400, BaudRate4800, BaudRate9600,
		BaudRate19200, BaudRate38400, BaudRate57600, BaudRate115200,
	}, baud)
	if i < 0 {
		return fmt.Errorf("rfx: unsupported baud rate %d", baud)
	}
	return d.SendRaw(ctx, []byte{'c', byte('0' + i)})
}

// Shutdown powers the device off. Per the UART specification (firmware
// 1.11+); not exercised against hardware.
func (d *Device) Shutdown(ctx context.Context) error {
	return d.SendRaw(ctx, []byte("S"))
}

// SendRaw sends a command without interpreting it. cmd is the command
// without the "#" and length prefix, such as "C0". The device buffers 64
// bytes of commands and handles about one per sweep, so SendRaw blocks when
// more than that is pending. Responses arrive through subscriptions.
func (d *Device) SendRaw(ctx context.Context, cmd []byte) error {
	if len(cmd) > 253 {
		return fmt.Errorf("rfx: command may not exceed a length of 253, got %d", len(cmd))
	}
	buf := make([]byte, 0, 2+len(cmd))
	buf = append(buf, '#', byte(2+len(cmd)))
	buf = append(buf, cmd...)
	return d.write(ctx, buf)
}

// configCommand sends a command that changes the configuration and returns
// the resulting configuration. The device only reports its configuration
// when a command changed it, so if changed is false, or no configuration
// accepted by match arrives, the configuration is requested. If strict, a
// resulting configuration not accepted by match is an error. reqMu must be
// held.
func (d *Device) configCommand(ctx context.Context, cmd []byte, changed, strict bool, match func(*SweepConfig) bool) (SweepConfig, error) {
	if changed {
		w := d.waiters.add(func(msg Message) bool {
			c, ok := msg.(*SweepConfig)
			return ok && match(c)
		})
		if err := d.SendRaw(ctx, cmd); err != nil {
			d.waiters.remove(w)
			return SweepConfig{}, err
		}
		echoCtx, cancel := context.WithTimeout(ctx, configEchoTimeout)
		msg, err := d.await(echoCtx, w)
		cancel()
		if err == nil {
			return *msg.(*SweepConfig), nil
		}
		if ctx.Err() != nil || !errors.Is(err, ErrTimeout) {
			return SweepConfig{}, err
		}
		// No echo: the device adjusted the request or considered it unchanged.
	}
	cfg, err := d.requestConfig(ctx)
	if err == nil && strict && !match(&cfg) {
		err = ErrNotApplied
	}
	return cfg, err
}

// requestConfig asks the device for its configuration. reqMu must be held.
func (d *Device) requestConfig(ctx context.Context) (SweepConfig, error) {
	msg, err := d.request(ctx, []byte("C0"), func(msg Message) bool { return msg.Kind() == KindConfig })
	if err != nil {
		return SweepConfig{}, err
	}
	return *msg.(*SweepConfig), nil
}

// request sends cmd and waits for the message accepted by match. reqMu must
// be held.
func (d *Device) request(ctx context.Context, cmd []byte, match func(Message) bool) (Message, error) {
	w := d.waiters.add(match)
	if err := d.SendRaw(ctx, cmd); err != nil {
		d.waiters.remove(w)
		return nil, err
	}
	return d.await(ctx, w)
}

// await waits for the message w was registered for.
func (d *Device) await(ctx context.Context, w *waiter) (Message, error) {
	defer d.waiters.remove(w)
	select {
	case msg := <-w.ch:
		return msg, nil
	case <-ctx.Done():
		return nil, ctxError(ctx)
	case <-d.readerDone:
		return nil, d.closedError()
	}
}

// ctxError returns the error for a done ctx, marking an expired deadline as
// ErrTimeout.
func ctxError(ctx context.Context) error {
	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrTimeout, context.DeadlineExceeded)
}

func (d *Device) closedError() error {
	if err := d.Err(); err != nil {
		return err
	}
	return ErrClosed
}

func (d *Device) write(ctx context.Context, b []byte) error {
	d.txMu.Lock()
	defer d.txMu.Unlock()
	// Wait for room in the device's receive buffer.
	for {
		d.flowMu.Lock()
		wait, ok := d.flow.reserve(len(b), time.Now())
		d.flowMu.Unlock()
		if ok {
			break
		}
		timer := time.NewTimer(wait)
		select {
		case <-d.txReady:
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctxError(ctx)
		case <-d.closeCh:
			timer.Stop()
			return ErrClosed
		}
		timer.Stop()
	}
	select {
	case <-d.readerDone:
		return d.closedError()
	default:
	}
	if n, err := d.port.Write(b); err != nil {
		return fmt.Errorf("rfx: failed to write to port: %w", err)
	} else if n != len(b) {
		return fmt.Errorf("rfx: expected to write %d bytes but wrote %d", len(b), n)
	}
	return nil
}

// readLoop reads from the port until it fails or is closed. It never blocks
// on subscribers.
func (d *Device) readLoop() {
	defer d.wg.Done()
	defer close(d.readerDone)
	var f framer
	buf := make([]byte, 4096)
	for {
		n, err := d.port.Read(buf)
		f.Write(buf[:n])
		for {
			frame, ok := f.Next()
			if !ok {
				break
			}
			msg, err := decode(frame)
			if err != nil {
				msg = &ProtocolError{Frame: bytes.Clone(frame), Err: err}
			}
			d.updateState(msg)
			d.waiters.deliver(msg)
			d.subsMu.Lock()
			for _, s := range d.subs {
				s.push(msg)
			}
			d.subsMu.Unlock()
		}
		if err != nil {
			select {
			case <-d.closeCh:
			default:
				d.mu.Lock()
				d.err = fmt.Errorf("rfx: failed to read from port: %w", err)
				d.mu.Unlock()
			}
			return
		}
	}
}

func (d *Device) unsubscribe(s *Subscription) {
	d.subsMu.Lock()
	if i := slices.Index(d.subs, s); i >= 0 {
		d.subs = slices.Delete(d.subs, i, i+1)
	}
	d.subsMu.Unlock()
}

// updateState tracks the device state reported by msg and completes messages
// that depend on it.
func (d *Device) updateState(msg Message) {
	switch msg := msg.(type) {
	case *Sweep:
		now := time.Now()
		d.flowMu.Lock()
		d.flow.sweep(now)
		d.flowMu.Unlock()
		select {
		case d.txReady <- struct{}{}:
		default:
		}
		d.mu.RLock()
		msg.Time = now
		msg.Start = d.config.Start
		msg.Step = d.config.Step
		d.mu.RUnlock()
	case *SweepConfig:
		d.mu.Lock()
		d.config = *msg
		d.mu.Unlock()
	case *CalculatorChange:
		d.mu.Lock()
		d.config.Calculator = msg.Mode
		d.mu.Unlock()
	case *DeviceInfo:
		d.mu.Lock()
		msg.Serial = d.info.Serial
		d.info = *msg
		d.mu.Unlock()
	case *SerialNumber:
		d.mu.Lock()
		d.info.Serial = msg.Serial
		d.mu.Unlock()
	case *Calibration:
		d.mu.Lock()
		d.calibration = msg.Data
		d.mu.Unlock()
	}
}
