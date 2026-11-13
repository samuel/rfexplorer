package trace

import (
	"image"
)

// Waterfall renders successive sweeps as rows of an image, newest first.
// Rows are stored in a ring buffer, so adding a sweep writes one row instead
// of scrolling the whole image.
type Waterfall struct {
	gradient    Gradient
	bottom, top float64
	img         *image.RGBA
	head        int // row holding the newest line
	line        []float32
}

// NewWaterfall returns a waterfall of the given size that maps levels from
// bottom to top dBm onto the default gradient.
func NewWaterfall(width, height int, bottom, top float64) *Waterfall {
	w := &Waterfall{gradient: DefaultGradient(), bottom: bottom, top: top}
	w.alloc(max(width, 1), max(height, 1))
	return w
}

func (w *Waterfall) alloc(width, height int) {
	w.img = image.NewRGBA(image.Rect(0, 0, width, height))
	w.line = make([]float32, width)
	w.head = 0
}

// Size returns the width and height of the waterfall in pixels.
func (w *Waterfall) Size() (width, height int) {
	return w.img.Rect.Dx(), w.img.Rect.Dy()
}

// Resize changes the image size, clearing it if the size changed.
func (w *Waterfall) Resize(width, height int) {
	width, height = max(width, 1), max(height, 1)
	if cw, ch := w.Size(); cw != width || ch != height {
		w.alloc(width, height)
	}
}

// SetRange changes the level range mapped onto the gradient, clearing the
// image if the range changed.
func (w *Waterfall) SetRange(bottom, top float64) {
	if bottom != w.bottom || top != w.top {
		w.bottom, w.top = bottom, top
		w.Clear()
	}
}

// Clear erases all rows to transparent.
func (w *Waterfall) Clear() {
	clear(w.img.Pix)
	w.head = 0
}

// Add resamples samples to the image width and writes them as the newest row.
func (w *Waterfall) Add(samples []float32) {
	_, height := w.Size()
	w.head = (w.head + height - 1) % height
	Resample(w.line, samples)
	span := w.top - w.bottom
	row := w.img.Pix[w.head*w.img.Stride:]
	for x, v := range w.line {
		var a float64
		if span != 0 {
			a = (float64(v) - w.bottom) / span
		}
		c := w.gradient.At(a)
		p := row[x*4 : x*4+4 : x*4+4]
		p[0], p[1], p[2], p[3] = c.R, c.G, c.B, c.A
	}
}

// Segments returns the waterfall as two views of its ring buffer in display
// order: draw first at the top and second directly below it. first starts
// with the newest row; second may be empty.
func (w *Waterfall) Segments() (first, second *image.RGBA) {
	width, height := w.Size()
	first = w.img.SubImage(image.Rect(0, w.head, width, height)).(*image.RGBA)
	second = w.img.SubImage(image.Rect(0, 0, width, w.head)).(*image.RGBA)
	return first, second
}
