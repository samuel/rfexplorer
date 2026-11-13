// Package trace processes sweeps independently of any user interface: max
// hold, averaging, peak search, channel power, waterfall images and color
// gradients. Power is combined in linear units (mW), never by averaging dBm.
package trace

import (
	"math"
	"time"

	"samuelks.com/rfx"
)

// axis identifies a sweep's frequency axis. Traces restart when it changes.
type axis struct {
	start, step rfx.Frequency
	n           int
}

func axisOf(s *rfx.Sweep) axis {
	return axis{start: s.Start, step: s.Step, n: len(s.Samples)}
}

// MaxHold keeps the highest level seen at each point. The zero value is
// ready to use.
type MaxHold struct {
	valid  bool
	axis   axis
	time   time.Time
	values []float32
}

// Add folds s into the hold. A sweep with a different frequency axis
// restarts it.
func (m *MaxHold) Add(s *rfx.Sweep) {
	m.time = s.Time
	if a := axisOf(s); !m.valid || a != m.axis {
		m.valid = true
		m.axis = a
		m.values = append(m.values[:0], s.Samples...)
		return
	}
	for i, v := range s.Samples {
		m.values[i] = max(m.values[i], v)
	}
}

// Reset discards the held levels.
func (m *MaxHold) Reset() {
	m.valid = false
	m.values = m.values[:0]
}

// Trace returns the held levels as a sweep. Its samples are owned by m: they
// are valid until the next Add or Reset and must not be modified.
func (m *MaxHold) Trace() rfx.Sweep {
	return rfx.Sweep{Time: m.time, Start: m.axis.start, Step: m.axis.step, Samples: m.values}
}

// Average is the mean power of the last N sweeps at each point.
type Average struct {
	n     int
	valid bool
	axis  axis
	time  time.Time
	ring  [][]float64 // linear power (mW) per sweep
	next  int
	count int
	out   []float32
}

// NewAverage returns an Average over the last n sweeps (at least 1).
func NewAverage(n int) *Average {
	return &Average{n: max(n, 1)}
}

// Add includes s in the average, replacing the oldest sweep once N are held.
// A sweep with a different frequency axis restarts the average.
func (a *Average) Add(s *rfx.Sweep) {
	a.time = s.Time
	if ax := axisOf(s); !a.valid || ax != a.axis {
		a.valid = true
		a.axis = ax
		a.next = 0
		a.count = 0
		if len(a.ring) != a.n {
			a.ring = make([][]float64, a.n)
		}
		a.out = make([]float32, ax.n)
	}
	slot := a.ring[a.next][:0]
	for _, v := range s.Samples {
		slot = append(slot, rfx.DBmToMilliwatts(float64(v)))
	}
	a.ring[a.next] = slot
	a.next = (a.next + 1) % a.n
	a.count = min(a.count+1, a.n)

	// Summing from scratch avoids the cancellation error a running sum
	// accumulates across a wide dynamic range.
	for i := range a.out {
		var sum float64
		for _, sw := range a.ring[:a.count] {
			sum += sw[i]
		}
		a.out[i] = float32(rfx.MilliwattsToDBm(sum / float64(a.count)))
	}
}

// Count returns the number of sweeps in the average.
func (a *Average) Count() int {
	return a.count
}

// Reset discards all sweeps.
func (a *Average) Reset() {
	a.valid = false
	a.count = 0
	a.next = 0
	a.out = a.out[:0]
}

// Trace returns the averaged levels as a sweep. Its samples are owned by a:
// they are valid until the next Add or Reset and must not be modified.
func (a *Average) Trace() rfx.Sweep {
	return rfx.Sweep{Time: a.time, Start: a.axis.start, Step: a.axis.step, Samples: a.out}
}

// indexRange returns the indexes of the first and last samples of s with a
// frequency in [lo, hi]. When none falls inside but the middle of the range
// is within the sweep, the sample nearest the middle is used, so a channel
// narrower than the sweep step still maps to a sample.
func indexRange(s *rfx.Sweep, lo, hi rfx.Frequency) (first, last int, ok bool) {
	n := len(s.Samples)
	if n == 0 || s.Step <= 0 || hi < lo {
		return 0, 0, false
	}
	first = int(max(ceilDiv(lo-s.Start, s.Step), 0))
	last = int(min(floorDiv(hi-s.Start, s.Step), rfx.Frequency(n-1)))
	if first <= last {
		return first, last, true
	}
	i, ok := s.IndexOf(lo + (hi-lo)/2)
	return i, i, ok
}

func floorDiv(a, b rfx.Frequency) rfx.Frequency {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func ceilDiv(a, b rfx.Frequency) rfx.Frequency {
	return -floorDiv(-a, b)
}

// Peak returns the index and level of the strongest sample of s with a
// frequency in [lo, hi]. A range narrower than the sweep step uses the sample
// nearest its middle. ok is false if the range is outside the sweep.
func Peak(s *rfx.Sweep, lo, hi rfx.Frequency) (i int, dBm float32, ok bool) {
	first, last, ok := indexRange(s, lo, hi)
	if !ok {
		return 0, 0, false
	}
	i, dBm = first, s.Samples[first]
	for j := first + 1; j <= last; j++ {
		if v := s.Samples[j]; v > dBm {
			i, dBm = j, v
		}
	}
	return i, dBm, true
}

// Window weights samples by their position across a channel.
type Window int

const (
	// WindowRect weights every sample in the channel equally.
	WindowRect Window = iota
	// WindowBlackman tapers to zero at the channel edges, de-emphasizing
	// energy that spills over from neighboring channels.
	WindowBlackman
)

// weight returns the window's weight at position d in [0, 1] across the
// channel.
func (w Window) weight(d float64) float64 {
	switch w {
	case WindowBlackman:
		return max(0, 0.42-0.5*math.Cos(2*math.Pi*d)+0.08*math.Cos(4*math.Pi*d))
	default:
		return 1
	}
}

// ChannelPower returns the window-weighted mean power, in dBm, of the samples
// of s within the channel of the given center and width. The mean is taken
// in linear power. A channel narrower than the sweep step uses the sample
// nearest its center. ok is false if the channel is outside the sweep.
func ChannelPower(s *rfx.Sweep, center, width rfx.Frequency, window Window) (dBm float64, ok bool) {
	lo := center - width/2
	first, last, ok := indexRange(s, lo, lo+width)
	if !ok {
		return 0, false
	}
	var sum, weights float64
	if width > 0 {
		for i := first; i <= last; i++ {
			w := window.weight(float64(s.FrequencyAt(i)-lo) / float64(width))
			sum += w * rfx.DBmToMilliwatts(float64(s.Samples[i]))
			weights += w
		}
	}
	if weights == 0 {
		// Only edge samples (or a fallback sample) with zero weight.
		sum = 0
		for i := first; i <= last; i++ {
			sum += rfx.DBmToMilliwatts(float64(s.Samples[i]))
		}
		weights = float64(last - first + 1)
	}
	return rfx.MilliwattsToDBm(sum / weights), true
}

// Resample fills dst from src. When src is longer each output point is the
// maximum of the source points it covers, so narrow peaks survive; when it
// is shorter source points are repeated.
func Resample(dst, src []float32) {
	n, w := len(src), len(dst)
	if n == 0 {
		clear(dst)
		return
	}
	for x := range w {
		lo := x * n / w
		hi := max((x+1)*n/w, lo+1)
		v := src[lo]
		for _, s := range src[lo+1 : hi] {
			v = max(v, s)
		}
		dst[x] = v
	}
}
