package utils

import (
	"fmt"
	"math"
	"strconv"
)

// AI generated
func HsvToHex(h, s, v float64) string {
	h = h - 360*float64(int(h/360))
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - AbsMod(h/60))
	m := v - c

	var r1, g1, b1 float64
	switch {
	case h < 60:
		r1, g1, b1 = c, x, 0
	case h < 120:
		r1, g1, b1 = x, c, 0
	case h < 180:
		r1, g1, b1 = 0, c, x
	case h < 240:
		r1, g1, b1 = 0, x, c
	case h < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	r := uint8((r1 + m) * 255)
	g := uint8((g1 + m) * 255)
	b := uint8((b1 + m) * 255)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// absMod returns |(h/60 mod 2) - 1|
func AbsMod(x float64) float64 {
	m := x - 2*float64(int(x/2))
	if m < 0 {
		m += 2
	}
	if m > 1 {
		return m - 1
	}
	return 1 - m
}

// AI generated
// HSLtoHEX converts HSL values to a hex color string "#RRGGBB".
func HslToHex(h, s, l float64) string {
	// Wrap hue to [0,360), then normalize to [0,1)
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	h /= 360

	s = clamp01(s)
	l = clamp01(l)

	var r, g, b float64
	if s == 0 {
		// achromatic (grey)
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hue2rgb(p, q, h+1.0/3.0)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3.0)
	}

	// convert to 0–255 and format as hex
	ri := int(math.Round(r * 255))
	gi := int(math.Round(g * 255))
	bi := int(math.Round(b * 255))

	return fmt.Sprintf("#%02X%02X%02X", ri, gi, bi)
}

// hue2rgb is a helper for HSL → RGB conversion
func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

// clamp01 ensures v is within [0,1]
func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// AI generated
func HexToHSL(hex string) (float64, float64, float64) {
	if len(hex) == 7 && hex[0] == '#' {
		hex = hex[1:]
	}

	rComp, _ := strconv.ParseUint(hex[0:2], 16, 8)
	gComp, _ := strconv.ParseUint(hex[2:4], 16, 8)
	bComp, _ := strconv.ParseUint(hex[4:6], 16, 8)

	// Normalize to [0,1]
	r := float64(rComp) / 255.0
	g := float64(gComp) / 255.0
	b := float64(bComp) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	// Compute lightness
	l := (max + min) / 2.0

	var h, s float64

	if delta == 0 {
		// achromatic
		h = 0
		s = 0
	} else {
		// Saturation
		s = delta / (1 - math.Abs(2*l-1))

		// Hue
		switch max {
		case r:
			h = (g - b) / delta
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/delta + 2
		case b:
			h = (r-g)/delta + 4
		}
		h *= 60 // convert to degrees
	}

	return h, s * 100, l * 100
}
