package rfx

import (
	"errors"
	"fmt"
	"strings"
)

// ErrOutOfRange is matched by a RangeError, returned when no installed board
// can sweep a requested range.
var ErrOutOfRange = errors.New("rfx: frequency range not supported by the device")

type boardSpec struct {
	min, max, maxSpan Frequency
}

// boardSpecs are the published limits of each analyzer model without
// optional licenses. Max span is the standard resolution limit.
var boardSpecs = map[Model]boardSpec{
	ModelWSUB1G:     {240 * MHz, 960 * MHz, 300 * MHz},
	Model24G:        {2350 * MHz, 2550 * MHz, 85 * MHz},
	ModelWSUB3G:     {15 * MHz, 2700 * MHz, 600 * MHz},
	Model6G:         {4850 * MHz, 6100 * MHz, 600 * MHz},
	ModelWSUB1GPlus: {50 * KHz, 960 * MHz, 960 * MHz},
	ModelAudioPro:   {15 * MHz, 2700 * MHz, 960 * MHz},
	Model2400Plus:   {2350 * MHz, 2550 * MHz, 200 * MHz},
	Model4GPlus:     {240 * MHz, 4000 * MHz, 960 * MHz},
	Model6GPlus:     {240 * MHz, 6100 * MHz, 960 * MHz},
}

// Range returns the published frequency limits and widest span of an
// analyzer board model, without optional licenses. ok is false for models
// with no known limits.
func (m Model) Range() (minFreq, maxFreq, maxSpan Frequency, ok bool) {
	s, ok := boardSpecs[m]
	return s.min, s.max, s.maxSpan, ok
}

// Board is an installed RF board and its limits.
type Board struct {
	Module Module
	Model  Model
	// Min and Max are the frequency limits and MaxSpan the widest sweep.
	// They are zero if unknown.
	Min     Frequency
	Max     Frequency
	MaxSpan Frequency
	// Reported is true once the limits came from the device, which happens
	// when the board is selected. Until then they are the model's published
	// limits, which don't include optional licenses.
	Reported bool
}

// Known reports whether the board's limits are known.
func (b Board) Known() bool {
	return b.Max > 0
}

// Contains reports whether the board can tune to f.
func (b Board) Contains(f Frequency) bool {
	return b.Known() && f >= b.Min && f <= b.Max
}

// Covers reports whether the board can sweep from start to stop.
func (b Board) Covers(start, stop Frequency) bool {
	return b.Contains(start) && b.Contains(stop) && stop-start <= b.MaxSpan
}

func (b Board) String() string {
	if !b.Known() {
		return fmt.Sprintf("%s (%s, range unknown)", b.Model, b.Module)
	}
	return fmt.Sprintf("%s (%s) %s to %s, max span %s", b.Model, b.Module, b.Min, b.Max, b.MaxSpan)
}

// setModel records the board's model. The published limits replace the
// current ones unless the device reported them before the model was known.
func (b *Board) setModel(m Model) {
	if b.Model == m {
		return
	}
	keep := b.Reported && b.Model == ModelNone
	b.Model = m
	if !keep {
		b.Min, b.Max, b.MaxSpan, _ = m.Range()
		b.Reported = false
	}
}

// installed reports whether the board is present.
func (b Board) installed() bool {
	return b.Reported || (b.Model != ModelNone && b.Model != ModelInvalid)
}

// RangeError reports a frequency range that no installed board can sweep.
// Start equals Stop for a single frequency.
type RangeError struct {
	Start  Frequency
	Stop   Frequency
	Boards []Board
}

func (e *RangeError) Error() string {
	var sb strings.Builder
	switch {
	case e.Start == e.Stop:
		fmt.Fprintf(&sb, "rfx: no board can tune to %s", e.Start)
	case e.spanBoard() != nil:
		b := e.spanBoard()
		fmt.Fprintf(&sb, "rfx: span %s exceeds the %s board's max span %s", e.Stop-e.Start, b.Model, b.MaxSpan)
		return sb.String()
	default:
		fmt.Fprintf(&sb, "rfx: no board can sweep %s to %s", e.Start, e.Stop)
	}
	for i, b := range e.Boards {
		if i == 0 {
			sb.WriteString("; boards: ")
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(b.String())
	}
	return sb.String()
}

// spanBoard returns a board containing both ends of the range, which means
// only the span is too wide.
func (e *RangeError) spanBoard() *Board {
	for i, b := range e.Boards {
		if b.Contains(e.Start) && b.Contains(e.Stop) {
			return &e.Boards[i]
		}
	}
	return nil
}

func (e *RangeError) Unwrap() error {
	return ErrOutOfRange
}

// pickBoard returns the board accepted by ok, preferring the active one so
// that no module switch is needed. If none is accepted but the active
// board's limits are unknown, the active board is returned and the device
// decides.
func pickBoard(boards []Board, active Module, ok func(Board) bool) (Board, bool) {
	for _, b := range boards {
		if b.Module == active && ok(b) {
			return b, true
		}
	}
	for _, b := range boards {
		if ok(b) {
			return b, true
		}
	}
	for _, b := range boards {
		if b.Module == active && !b.Known() {
			return b, true
		}
	}
	if len(boards) == 0 {
		return Board{Module: active, Model: ModelNone}, true
	}
	return Board{}, false
}
