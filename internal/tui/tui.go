// Package tui is a terminal spectrum display for an RF Explorer.
package tui

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"samuelks.com/bandplan"
	"samuelks.com/rfx"
	"samuelks.com/rfx/trace"
)

// commandTimeout bounds a device command. Switching modules alone can take
// the device a few seconds.
const commandTimeout = 10 * time.Second

// leftWidth is the width of the information column left of the plot.
const leftWidth = 32

var (
	styleNormal  = tcell.StyleDefault.Foreground(color.White).Background(color.Black)
	styleInverse = tcell.StyleDefault.Foreground(color.Black).Background(color.White)
	styleError   = tcell.StyleDefault.Foreground(color.Red).Background(color.Black)
)

var keyHelp = []string{
	"Keys:",
	" w  WiFi 2.4 GHz channels",
	" v  5.8 GHz FPV channels",
	" m  max hold   r  realtime",
	" h  hold       c  continue",
	" l  LCD on/off",
	" s  screen dump on/off",
	" q  quit",
}

// Options configures Run.
type Options struct {
	// Logger receives device messages and command failures. Nil discards
	// them.
	Logger *slog.Logger
}

// namedChannel is a channel labeled with its band, such as "A1".
type namedChannel struct {
	bandplan.Channel
	name string
}

// channelsOf flattens a band set into channels named band + channel.
func channelsOf(bs *bandplan.BandSet) []namedChannel {
	var chs []namedChannel
	for _, b := range bs.Bands {
		for _, c := range b.Channels {
			name := c.Name
			if len(bs.Bands) > 1 {
				name = b.Name + c.Name
			}
			chs = append(chs, namedChannel{Channel: c, name: name})
		}
	}
	return chs
}

type tui struct {
	dev    *rfx.Device
	screen tcell.Screen
	logger *slog.Logger

	// results carries the outcome of commands run off the event loop.
	results chan error

	config     rfx.SweepConfig
	boards     []rfx.Board
	sweep      *rfx.Sweep
	lcd        *rfx.ScreenImage
	maxHold    trace.MaxHold
	status     string
	statusErr  bool
	lcdEnabled bool
	dumping    bool
	wifi       bool
	vtx        bool

	wifiChannels []namedChannel
	vtxChannels  []namedChannel

	// Scratch buffers for resampling to the plot width.
	cols, heldCols []float32
}

// Run shows the spectrum display until ctx is done, the user quits or the
// device connection fails. The terminal is restored before it returns.
func Run(ctx context.Context, dev *rfx.Device, opts Options) (err error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("opening terminal: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("initializing terminal: %w", err)
	}
	defer func() {
		// Restore the terminal before a panic is reported.
		p := recover()
		screen.Fini()
		if p != nil {
			panic(p)
		}
	}()
	screen.SetStyle(styleNormal)
	screen.HideCursor()

	t := &tui{
		dev:          dev,
		screen:       screen,
		logger:       logger,
		results:      make(chan error, 8),
		config:       dev.Config(),
		boards:       dev.Boards(),
		lcdEnabled:   true,
		wifiChannels: channelsOf(bandplan.Builtin(bandplan.NameWiFi24)),
		vtxChannels:  channelsOf(bandplan.Builtin(bandplan.NameFPV58)),
	}
	return t.loop(ctx)
}

func (t *tui) loop(ctx context.Context) error {
	sub := t.dev.Subscribe(rfx.SubscribeOptions{})
	defer sub.Close()
	events := t.screen.EventQ()
	t.draw()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-sub.C():
			if !ok {
				if err := t.dev.Err(); err != nil {
					return fmt.Errorf("connection lost: %w", err)
				}
				return errors.New("connection closed")
			}
			t.handleMessage(msg)
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				t.screen.Sync()
			case *tcell.EventKey:
				if t.handleKey(ev) {
					return nil
				}
			}
		case err := <-t.results:
			if err != nil {
				t.status, t.statusErr = err.Error(), true
			} else {
				t.status, t.statusErr = "", false
			}
		}
		t.draw()
	}
}

func (t *tui) handleMessage(msg rfx.Message) {
	switch msg := msg.(type) {
	case *rfx.SweepConfig:
		t.logger.Info("config", "config", fmt.Sprintf("%+v", *msg))
		t.config = *msg
		t.boards = t.dev.Boards()
		t.maxHold.Reset()
	case *rfx.Sweep:
		t.sweep = msg
		t.maxHold.Add(msg)
	case *rfx.ScreenImage:
		t.lcd = msg
	case *rfx.Unhandled:
		t.logger.Info("unhandled frame", "data", hex.EncodeToString(msg.Data))
	default:
		t.logger.Info("message", "kind", msg.Kind().String(), "message", fmt.Sprintf("%+v", msg))
	}
}

// command runs f off the event loop and reports its outcome on the status
// line.
func (t *tui) command(name string, f func(ctx context.Context) error) {
	t.status, t.statusErr = name+"...", false
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		err := f(ctx)
		if err != nil {
			t.logger.Error(name, "error", err)
			err = fmt.Errorf("%s: %w", name, err)
		}
		t.results <- err
	}()
}

func (t *tui) setRange(start, stop rfx.Frequency) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		_, err := t.dev.SetSweep(ctx, rfx.SweepRequest{Start: start, Stop: stop, AmpTop: 0, AmpBottom: -120})
		return err
	}
}

// handleKey acts on a key press and reports whether to quit.
func (t *tui) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return true
	case tcell.KeyRune:
	default:
		return false
	}
	switch ev.Str() {
	case "q":
		return true
	case "c":
		t.command("continue", func(ctx context.Context) error {
			_, err := t.dev.Resume(ctx)
			return err
		})
	case "h":
		t.command("hold", t.dev.Hold)
	case "l":
		t.lcdEnabled = !t.lcdEnabled
		on := t.lcdEnabled
		t.command("set LCD", func(ctx context.Context) error { return t.dev.SetLCD(ctx, on) })
	case "m":
		t.command("set max hold", func(ctx context.Context) error { return t.dev.SetCalculator(ctx, rfx.CalculatorMaxHold) })
	case "r":
		t.command("set realtime", func(ctx context.Context) error { return t.dev.SetCalculator(ctx, rfx.CalculatorNormal) })
	case "s":
		t.dumping = !t.dumping
		on := t.dumping
		t.lcd = nil
		t.command("set screen dump", func(ctx context.Context) error { return t.dev.SetScreenDump(ctx, on) })
	case "v":
		t.vtx = !t.vtx
		if t.vtx {
			t.wifi = false
			t.command("select FPV range", t.setRange(5350*rfx.MHz, 5950*rfx.MHz))
		}
	case "w":
		t.wifi = !t.wifi
		if t.wifi {
			t.vtx = false
			t.command("select WiFi range", t.setRange(2401*rfx.MHz, 2495*rfx.MHz))
		}
	}
	return false
}

func (t *tui) set(x, y int, r rune) {
	t.screen.SetContent(x, y, r, nil, styleNormal)
}

func (t *tui) put(x, y int, s string) {
	t.screen.PutStrStyled(x, y, s, styleNormal)
}

func (t *tui) draw() {
	t.screen.Clear()
	defer t.screen.Show()

	if t.dumping {
		if t.lcd != nil {
			t.drawLCD()
		} else {
			t.put(0, 0, "Waiting for screen image...")
		}
		return
	}

	width, height := t.screen.Size()
	t.drawInfo(height)
	if t.status != "" {
		style := styleNormal
		if t.statusErr {
			style = styleError
		}
		t.screen.PutStrStyled(leftWidth, 0, t.status, style)
	}
	if t.sweep == nil || len(t.sweep.Samples) == 0 {
		t.put(leftWidth, 1, "Waiting for sweeps...")
		return
	}

	top, bottom := 1, height-2
	left, plotW := leftWidth, width-leftWidth-1
	if plotW < 2 || bottom-top < 4 {
		return
	}

	// Axes
	for x := left; x < left+plotW; x++ {
		t.set(x, bottom, '-')
	}
	for y := top; y < bottom; y++ {
		t.set(left-1, y, '|')
	}
	t.set(left-1, bottom, '+')

	ampTop, ampBottom := float64(t.config.AmpTop), float64(t.config.AmpBottom)
	ampToY := func(amp float32) int {
		if ampTop == ampBottom {
			return bottom
		}
		y := top + int(float64(bottom-top)*(float64(amp)-ampTop)/(ampBottom-ampTop)+0.5)
		return min(max(y, top), bottom)
	}

	s := t.sweep
	if t.wifi {
		t.drawChannels(s, left, plotW, bottom, ampToY)
	} else {
		t.drawTrace(s, left, plotW, bottom, ampToY)
	}

	// Amplitude labels
	label := strconv.Itoa(t.config.AmpTop)
	t.put(left-len(label)-1, top, label)
	label = strconv.Itoa(t.config.AmpBottom)
	t.put(left-len(label)-1, bottom-1, label)

	// Frequency labels
	n := len(s.Samples)
	t.put(left, bottom+1, fmt.Sprintf("%.3f", s.Start.MHz()))
	label = fmt.Sprintf("%.3f", s.FrequencyAt(n/2).MHz())
	t.put(left+plotW/2-len(label)/2, bottom+1, label)
	label = fmt.Sprintf("%.3f", s.FrequencyAt(n-1).MHz())
	t.put(left+plotW-len(label), bottom+1, label)
}

func (t *tui) drawInfo(height int) {
	c := t.config
	lines := []string{
		fmt.Sprintf("Calculator: %s", c.Calculator),
		fmt.Sprintf("Points: %d", c.Points),
		fmt.Sprintf("RBW: %s", c.RBW),
	}
	for _, b := range t.boards {
		mark := " "
		if b.Module == c.Module() {
			mark = "*"
		}
		lines = append(lines, fmt.Sprintf("%s%s", mark, b.Model))
		if b.Known() {
			lines = append(lines, fmt.Sprintf("  %.3f-%.3f", b.Min.MHz(), b.Max.MHz()), fmt.Sprintf("  max span %s", b.MaxSpan))
		}
	}
	lines = append(append(lines, ""), keyHelp...)
	for i, l := range lines {
		if i >= height-3 {
			break
		}
		t.put(0, i, l)
	}
}

// columnOf returns the plot column that trace.Resample maps sample i of n
// into across w columns.
func columnOf(i, n, w int) int {
	x := i * w / n
	for x+1 < w && (x+1)*n/w <= i {
		x++
	}
	return x
}

func (t *tui) drawTrace(s *rfx.Sweep, left, plotW, bottom int, ampToY func(float32) int) {
	if len(t.cols) != plotW {
		t.cols = make([]float32, plotW)
		t.heldCols = make([]float32, plotW)
	}
	trace.Resample(t.cols, s.Samples)
	trace.Resample(t.heldCols, t.maxHold.Trace().Samples)
	held := t.heldCols

	for x, v := range t.cols {
		for y := ampToY(v); y < bottom; y++ {
			t.set(left+x, y, '.')
		}
		y := ampToY(held[x])
		t.set(left+x, y, '#')
		if x > 0 {
			prev := ampToY(held[x-1])
			for yy := y + 1; yy < prev; yy++ {
				t.set(left+x-1, yy, '⎟')
			}
			for yy := y - 1; yy > prev; yy-- {
				t.set(left+x, yy, '|')
			}
		}
	}

	peakIdx, peakAmp := s.Peak()
	peakFreq := s.FrequencyAt(peakIdx)
	x := left + columnOf(peakIdx, len(s.Samples), plotW)
	t.drawMarker(x, ampToY(peakAmp), fmt.Sprintf("%.3f", peakFreq.MHz()), fmt.Sprintf("%.1f", peakAmp))

	if t.vtx {
		var names []string
		for _, c := range t.vtxChannels {
			if c.Contains(bandplan.Frequency(peakFreq)) {
				names = append(names, c.name)
			}
		}
		t.put(0, bottom-1, "Peak: "+strings.Join(names, ", "))
	}
}

func (t *tui) drawChannels(s *rfx.Sweep, left, plotW, bottom int, ampToY func(float32) int) {
	channels := t.wifiChannels
	barWidth := plotW / len(channels)
	if barWidth < 2 {
		return
	}
	best, bestPower := -1, 0.0
	for i, c := range channels {
		startX := left + i*barWidth
		if p, ok := trace.ChannelPower(s, rfx.Frequency(c.Center), rfx.Frequency(c.Width), trace.WindowBlackman); ok {
			if best < 0 || p > bestPower {
				best, bestPower = i, p
			}
			startY := ampToY(float32(p))
			for x := startX; x < startX+barWidth; x++ {
				t.set(x, startY, '-')
			}
			for y := startY; y < bottom; y++ {
				t.set(startX, y, '|')
				t.set(startX+barWidth, y, '|')
			}
			t.set(startX, startY, '+')
			t.set(startX+barWidth, startY, '+')
		}
		t.put(startX+(barWidth-len(c.name))/2, bottom-1, c.name)
	}
	if best >= 0 {
		x := left + best*barWidth + barWidth/2
		t.drawMarker(x, ampToY(float32(bestPower)), "ch "+channels[best].name, fmt.Sprintf("%.1f", bestPower))
	}
}

// drawMarker draws a 'V' above (x, y) with two label lines above it.
func (t *tui) drawMarker(x, y int, line1, line2 string) {
	t.set(x, y-1, 'V')
	t.put(x-2, y-3, line1)
	t.put(x-2, y-2, line2)
}

// drawLCD draws the device screen using half-block characters, two pixel
// rows per terminal row.
func (t *tui) drawLCD() {
	b := t.lcd.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x++ {
			top := t.lcd.AtGray(x, y).Y != 0
			bot := t.lcd.AtGray(x, y+1).Y != 0
			r, style := ' ', styleNormal
			switch {
			case top && bot:
				style = styleInverse
			case top:
				r, style = '▄', styleInverse
			case bot:
				r, style = '▀', styleInverse
			}
			t.screen.SetContent(x, y/2, r, nil, style)
		}
	}
}
