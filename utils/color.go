package utils

import "image/color"

// DisableColor takes a color.NRGBA and returns a desaturated,
// semi-transparent version for disabled UI elements.
func DisableColor(c color.NRGBA) color.NRGBA {
	// Calculate Luminance (standard weights for human eye perception)
	// L = 0.299R + 0.587G + 0.114B
	lum := uint8(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B))

	// Reduce Alpha (e.g., set to ~45% of original or a fixed low value)
	// A value of 110-128 is usually good for disabled states.
	newAlpha := uint8(float64(c.A) * 0.45)

	return color.NRGBA{
		R: lum,
		G: lum,
		B: lum,
		A: newAlpha,
	}
}
