package trace

import (
	"image/color"
	"math"
)

// Gradient maps values in [0, 1] to colors by interpolating linearly between
// evenly spaced stops.
type Gradient []color.RGBA

// At returns the color for v. Values outside [0, 1] are clamped.
func (g Gradient) At(v float64) color.RGBA {
	switch {
	case len(g) == 0:
		return color.RGBA{}
	case len(g) == 1 || math.IsNaN(v) || v <= 0:
		return g[0]
	case v >= 1:
		return g[len(g)-1]
	}
	pos := v * float64(len(g)-1)
	i := int(pos)
	t := pos - float64(i)
	a, b := g[i], g[i+1]
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), lerp(a.A, b.A)}
}

// DefaultGradient returns the black → blue → yellow → red → dark red
// gradient used for waterfalls and heat maps.
func DefaultGradient() Gradient {
	return Gradient{
		{0x00, 0x00, 0x00, 0xff},
		{0x00, 0x00, 0x20, 0xff},
		{0x00, 0x00, 0x30, 0xff},
		{0x00, 0x00, 0x50, 0xff},
		{0x00, 0x00, 0x91, 0xff},
		{0x1e, 0x90, 0xff, 0xff},
		{0xff, 0xff, 0x00, 0xff},
		{0xfe, 0x6d, 0x16, 0xff},
		{0xff, 0x00, 0x00, 0xff},
		{0xc6, 0x00, 0x00, 0xff},
		{0x9f, 0x00, 0x00, 0xff},
		{0x75, 0x00, 0x00, 0xff},
		{0x4a, 0x00, 0x00, 0xff},
	}
}
