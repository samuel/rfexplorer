package main

import (
	"image/color"
	"math"
)

var gradient = [13]color.RGBA{
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

// gradientColorForValue returns a gradient color when given a value in the range [0.0,1.0]
func gradientColorForValue(value float64) color.RGBA {
	if value < 0.0 {
		value = 0.0
	} else if value >= 1.0 {
		value = 1.0
	}

	colorF := value * float64(len(gradient)+1)
	alpha := colorF - float64(math.Floor(colorF))

	color1I := int(colorF)
	if color1I >= len(gradient) {
		return gradient[len(gradient)-1]
	}
	color2I := color1I + 1
	if color2I >= len(gradient) {
		color2I = len(gradient) - 1
	}
	color1 := gradient[color1I]
	color2 := gradient[color2I]
	return color.RGBA{
		uint8(int(color1.R) + int(float64(int(color2.R)-int(color1.R))*alpha)),
		uint8(int(color1.G) + int(float64(int(color2.G)-int(color1.G))*alpha)),
		uint8(int(color1.B) + int(float64(int(color2.B)-int(color1.B))*alpha)),
		255,
	}
}
