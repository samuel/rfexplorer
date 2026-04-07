package main

import (
	"image"
	"sync"
)

type Waterfall struct {
	width, height int
	min, max      float64
	top           bool
	scroll        bool

	mu  sync.Mutex
	img *image.RGBA
	row int
}

func NewWaterfall(width, height int, min, max float64, top, scroll bool) *Waterfall {
	return &Waterfall{
		width:  width,
		height: height,
		min:    min,
		max:    max,
		img:    image.NewRGBA(image.Rect(0, 0, width, height)),
		top:    top,
		scroll: scroll,
	}
}

func (w *Waterfall) Update(values []float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// TODO: support interpolating values if it doesn't match width
	if len(values) > w.width {
		values = values[:w.width]
	}
	row := w.row
	if w.scroll {
		if w.top {
			row = 0
			copy(w.img.Pix[w.img.Stride:], w.img.Pix)
		} else {
			row = w.height - 1
			copy(w.img.Pix, w.img.Pix[w.img.Stride:])
		}
	}
	for i, v := range values {
		c := gradientColorForValue((v - w.min) / (w.max - w.min))
		w.img.Pix[row*w.img.Stride+i*4] = c.R
		w.img.Pix[row*w.img.Stride+i*4+1] = c.G
		w.img.Pix[row*w.img.Stride+i*4+2] = c.B
		w.img.Pix[row*w.img.Stride+i*4+3] = c.A
	}
	w.row = (w.row + 1) % w.height
}

func (w *Waterfall) Row() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.row
}

func (w *Waterfall) Image(m *image.RGBA) *image.RGBA {
	w.mu.Lock()
	defer w.mu.Unlock()

	if m == nil || m.Bounds() != w.img.Bounds() {
		m = image.NewRGBA(w.img.Bounds())
	}
	copy(m.Pix, w.img.Pix)
	return m
}
