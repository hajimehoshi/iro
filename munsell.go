// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package iro

import "math"

// The Munsell renotation data is published by the Rochester Institute of Technology.
// See the Notice section of README.md for its sources.

//go:generate go run gen.go

// Munsell hue family positions on the 100-step Munsell hue circle.
// A hue is the family position plus its prefix, so 7.5PB is MunsellPB+7.5.
const (
	MunsellR  = 0.0
	MunsellYR = 10.0
	MunsellY  = 20.0
	MunsellGY = 30.0
	MunsellG  = 40.0
	MunsellBG = 50.0
	MunsellB  = 60.0
	MunsellPB = 70.0
	MunsellP  = 80.0
	MunsellRP = 90.0
)

const munsellHueCount = 40

// munsellValues lists the Munsell values covered by the renotation data.
var munsellValues = [...]float64{0.2, 0.4, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// The renotation chromaticities are given under CIE illuminant C with the 1931
// 2 degree observer.
const (
	illuminantCx = 0.31006
	illuminantCy = 0.31616
)

// ColorFromMunsell builds a Color from a Munsell hue, value and chroma, and alpha.
//
// The hue is a position on the 100-step Munsell hue circle, where each family
// spans ten steps from [MunsellR] through [MunsellRP]; it wraps freely.
// The value is in [0,10] and the chroma is non-negative.
// Values outside the renotation data are clamped to it.
//
// A chroma of 0 is a neutral, written N in Munsell notation, and the hue is then
// ignored: N5/ is ColorFromMunsell(0, 5, 0, 1).
//
// The Munsell renotation is defined under CIE illuminant C with the 1931 2 degree
// observer. The result is adapted to D65 with the Bradford transform, so that it
// shares the white point of the other color spaces of this package.
func ColorFromMunsell(hue, value, chroma, alpha float64) Color {
	value = min(max(value, 0), 10)
	y := munsellLuminance(value)

	x, yy := munsellChromaticity(hue, value, chroma)

	// Convert xyY to XYZ, then adapt from illuminant C to D65.
	var cx, cy, cz float64
	if yy != 0 {
		cx = x * y / yy
		cy = y
		cz = (1 - x - yy) * y / yy
	}

	// Adapt from illuminant C to D65 with the Bradford transform, whose matrix is
	// M^-1 . diag(D65 cone response / C cone response) . M, for the cone response
	// matrix M of Lam, "Metamerism and Colour Constancy", PhD thesis, University of
	// Bradford (1985):
	//
	//	 0.8951  0.2664 -0.1614
	//	-0.7502  1.7135  0.0367
	//	 0.0389 -0.0685  1.0296
	//
	// The D65 it adapts to is the white point of the other spaces of this package,
	// not a rounded tabulation of it. TestMunsellAdaptsFromIlluminantC derives the
	// product again from the above and checks these values against it.
	return Color{
		x:     cx*0.9904204325304461 + cy*-0.0071783714344136 + cz*-0.0115685684119785,
		y:     cx*-0.0123813610481101 + cy*1.0155865733199367 + cz*-0.0029131740977424,
		z:     cx*-0.0035535917172210 + cy*0.0067525960192809 + cz*0.9184103600262958,
		alpha: alpha,
	}
}

// munsellLuminance returns the luminance factor in [0,1] for a Munsell value in [0,10].
func munsellLuminance(v float64) float64 {
	// The quintic of ASTM D1535, Standard Practice for Specifying Color by the Munsell
	// System, section 8.7. Its coefficients are those of the 1943 renotation scaled by
	// 0.975, which refers the result to the perfect reflecting diffuser rather than to
	// magnesium oxide, and they give exactly 100 at value 10.
	// TestMunsellLuminanceMatchesRenotation checks them against the renotation data.
	return v * (1.1914 + v*(-0.22533+v*(0.23352+v*(-0.020484+v*0.00081939)))) / 100
}

// munsellChromaticity returns the CIE xy chromaticity under illuminant C
// for a Munsell hue, value and chroma.
func munsellChromaticity(hue, value, chroma float64) (x, y float64) {
	if chroma <= 0 {
		return illuminantCx, illuminantCy
	}

	lo, hi, t := munsellValueBracket(value)

	x0, y0 := munsellPlaneChromaticity(hue, lo, chroma)
	if lo == hi {
		return x0, y0
	}
	x1, y1 := munsellPlaneChromaticity(hue, hi, chroma)

	return x0 + (x1-x0)*t, y0 + (y1-y0)*t
}

// munsellValueBracket returns the indices of the value planes surrounding value,
// and the interpolation factor between them measured in luminance.
func munsellValueBracket(value float64) (lo, hi int, t float64) {
	// The samples stop at value 0.2, below which the luminance factor is small
	// enough that holding the chromaticity constant is not visible.
	if value <= munsellValues[0] {
		return 0, 0, 0
	}
	if value >= munsellValues[len(munsellValues)-1] {
		return len(munsellValues) - 1, len(munsellValues) - 1, 0
	}

	hi = 1
	for munsellValues[hi] < value {
		hi++
	}
	if munsellValues[hi] == value {
		return hi, hi, 0
	}
	lo = hi - 1

	// Interpolating in luminance rather than value keeps the result consistent
	// with the luminance assigned to the result.
	ylo := munsellLuminance(munsellValues[lo])
	yhi := munsellLuminance(munsellValues[hi])
	return lo, hi, (munsellLuminance(value) - ylo) / (yhi - ylo)
}

// munsellPlaneChromaticity returns the chromaticity for a hue and chroma
// within a single value plane.
func munsellPlaneChromaticity(hue float64, valueIdx int, chroma float64) (x, y float64) {
	// A hue and chroma are only interpolatable where every surrounding sample
	// exists, so the chroma is clamped to what this plane supports.
	chroma = min(chroma, munsellMaxChroma(hue, valueIdx))
	if chroma <= 0 {
		return illuminantCx, illuminantCy
	}

	// Chroma samples are spaced by 2, with the illuminant standing in for chroma 0.
	i := int(math.Floor(chroma / 2))
	frac := chroma/2 - float64(i)

	x1, y1 := illuminantCx, illuminantCy
	if i > 0 {
		x1, y1 = munsellHueChromaticity(hue, valueIdx, i-1)
	}
	if frac == 0 {
		return x1, y1
	}
	x2, y2 := munsellHueChromaticity(hue, valueIdx, i)

	return x1 + (x2-x1)*frac, y1 + (y2-y1)*frac
}

// munsellHueChromaticity returns the chromaticity for a hue at a fixed value plane
// and chroma sample, interpolating between the surrounding renotation hues.
func munsellHueChromaticity(hue float64, valueIdx, chromaIdx int) (x, y float64) {
	// Renotation hues are the 40 multiples of 2.5, indexed so that index 0 is 2.5R.
	pos := hue/2.5 - 1
	i := int(math.Floor(pos))
	t := pos - float64(i)

	x1, y1, _ := munsellSample(i, valueIdx, chromaIdx)
	if t == 0 {
		return x1, y1
	}
	x2, y2, _ := munsellSample(i+1, valueIdx, chromaIdx)

	// The samples at a fixed value and chroma trace an ovoid around the illuminant,
	// so they are interpolated as radius and angle about it. A cubic through the two
	// neighbouring samples follows the ovoid's curvature where they are available.
	x0, y0, ok0 := munsellSample(i-1, valueIdx, chromaIdx)
	x3, y3, ok3 := munsellSample(i+2, valueIdx, chromaIdx)
	if !ok0 || !ok3 {
		r1, a1 := munsellPolar(x1, y1)
		r2, a2 := munsellPolar(x2, y2)
		a2 = unwrapAngle(a1, a2)
		return munsellCartesian(r1+(r2-r1)*t, a1+(a2-a1)*t)
	}

	r0, a0 := munsellPolar(x0, y0)
	r1, a1 := munsellPolar(x1, y1)
	r2, a2 := munsellPolar(x2, y2)
	r3, a3 := munsellPolar(x3, y3)
	a0 = unwrapAngle(a1, a0)
	a2 = unwrapAngle(a1, a2)
	a3 = unwrapAngle(a2, a3)

	return munsellCartesian(catmullRom(r0, r1, r2, r3, t), catmullRom(a0, a1, a2, a3, t))
}

// munsellSample returns the renotation chromaticity at a hue, value and chroma index,
// reporting whether the data covers it. The hue index wraps around the hue circle.
func munsellSample(hueIdx, valueIdx, chromaIdx int) (x, y float64, ok bool) {
	hueIdx = ((hueIdx % munsellHueCount) + munsellHueCount) % munsellHueCount

	g := hueIdx*len(munsellValues) + valueIdx
	start := int(munsellOffsets[g])
	if chromaIdx < 0 || chromaIdx >= int(munsellOffsets[g+1])-start {
		return 0, 0, false
	}

	i := 2 * (start + chromaIdx)
	return munsellXY[i], munsellXY[i+1], true
}

// munsellMaxChroma returns the largest chroma the renotation data covers for a hue
// in a value plane, being the largest chroma common to the surrounding hues.
func munsellMaxChroma(hue float64, valueIdx int) float64 {
	pos := hue/2.5 - 1
	i := int(math.Floor(pos))

	c := munsellHueMaxChroma(i, valueIdx)
	if pos != math.Floor(pos) {
		c = min(c, munsellHueMaxChroma(i+1, valueIdx))
	}
	return c
}

func munsellHueMaxChroma(hueIdx, valueIdx int) float64 {
	hueIdx = ((hueIdx % munsellHueCount) + munsellHueCount) % munsellHueCount

	g := hueIdx*len(munsellValues) + valueIdx
	return 2 * float64(int(munsellOffsets[g+1])-int(munsellOffsets[g]))
}

func munsellPolar(x, y float64) (r, a float64) {
	x -= illuminantCx
	y -= illuminantCy
	return math.Hypot(x, y), math.Atan2(y, x)
}

func munsellCartesian(r, a float64) (x, y float64) {
	return illuminantCx + r*math.Cos(a), illuminantCy + r*math.Sin(a)
}

// unwrapAngle returns b shifted by whole turns to lie within half a turn of a.
func unwrapAngle(a, b float64) float64 {
	return a + math.Remainder(b-a, 2*math.Pi)
}

// catmullRom evaluates a Catmull-Rom spline through four uniformly spaced values.
func catmullRom(y0, y1, y2, y3, t float64) float64 {
	return y1 + 0.5*t*(y2-y0+t*(2*y0-5*y1+4*y2-y3+t*(3*(y1-y2)+y3-y0)))
}
