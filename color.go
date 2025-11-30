// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Hajime Hoshi

// Package iro provides color conversion utilities between various color spaces.
package iro

import (
	"image/color"
	"math"
)

// The matrices are referenced from: https://www.w3.org/TR/css-color-4/#color-conversion-code

// Color holds an XYZ D65 color and alpha.
// Unlike [color.Color], Color has one color space (XYZ D65) internally,
// and can convert to/from various color spaces to minimize loss within floating-point limits.
type Color struct {
	x     float64
	y     float64
	z     float64
	alpha float64
}

// Alpha returns the alpha value.
func (c Color) Alpha() float64 {
	return c.alpha
}

// WithAlpha returns a new Color with the alpha value.
func (c Color) WithAlpha(alpha float64) Color {
	return Color{
		x:     c.x,
		y:     c.y,
		z:     c.z,
		alpha: alpha,
	}
}

// ColorFromXYZ builds a Color from XYZ D65 coordinates and alpha in [0,1].
func ColorFromXYZ(x, y, z, alpha float64) Color {
	return Color{
		x:     x,
		y:     y,
		z:     z,
		alpha: alpha,
	}
}

// ColorFromSRGB builds a Color from nonlinear sRGB channels in [0,1] and alpha.
func ColorFromSRGB(r, g, b, alpha float64) Color {
	r = degamma(r)
	g = degamma(g)
	b = degamma(b)

	return ColorFromLinearSRGB(r, g, b, alpha)
}

// ColorFromSRGBColor converts an sRGB [color.Color] to Color.
// As special cases, [color.NRGBA] and [color.NRGBA64] are handled directly without RGBA method calls.
// For other [color.Color] values, the RGBA method is used, assuming the values are alpha-premultiplied after applied gamma.
func ColorFromSRGBColor(c color.Color) Color {
	switch v := c.(type) {
	case color.NRGBA:
		// Use non-premultiplied alpha directly.
		// This is not only for performance but also for semantics:
		// RGB values are no longer precise after premultiplying alpha.
		return ColorFromSRGB(
			float64(v.R)/0xff,
			float64(v.G)/0xff,
			float64(v.B)/0xff,
			float64(v.A)/0xff,
		)
	case color.NRGBA64:
		// Use non-premultiplied alpha directly in the same way as color.NRGBA.
		return ColorFromSRGB(
			float64(v.R)/0xffff,
			float64(v.G)/0xffff,
			float64(v.B)/0xffff,
			float64(v.A)/0xffff,
		)
	case color.Alpha:
		// This is just a performance optimization.
		a := float64(v.A) / 0xff
		return ColorFromSRGB(
			1,
			1,
			1,
			a,
		)
	case color.Alpha16:
		// This is just a performance optimization.
		a := float64(v.A) / 0xffff
		return ColorFromSRGB(
			1,
			1,
			1,
			a,
		)
	case color.Gray:
		// This is just a performance optimization.
		y := float64(v.Y) / 0xff
		return ColorFromSRGB(
			y,
			y,
			y,
			1,
		)
	case color.Gray16:
		// This is just a performance optimization.
		y := float64(v.Y) / 0xffff
		return ColorFromSRGB(
			y,
			y,
			y,
			1,
		)
	default:
		// Applying degamma the RGB values after dividing by alpha might be incorrect,
		// but color.Color interface does not provide enough information to do better.
		r, g, b, a := c.RGBA()
		if a == 0 {
			return Color{}
		}
		return ColorFromSRGB(
			float64(r)/float64(a),
			float64(g)/float64(a),
			float64(b)/float64(a),
			float64(a)/0xffff,
		)
	}
}

// ColorFromLinearSRGB builds a Color from linear sRGB channels in [0,1] and alpha.
func ColorFromLinearSRGB(r, g, b, alpha float64) Color {
	return Color{
		x:     r*506752/1228815 + g*87881/245763 + b*12673/70218,
		y:     r*87098/409605 + g*175762/245763 + b*12673/175545,
		z:     r*7918/409605 + g*87881/737289 + b*1001167/1053270,
		alpha: alpha,
	}
}

// ColorFromLinearSRGBColor converts a linear sRGB [color.Color] to Color.
func ColorFromLinearSRGBColor(c color.Color) Color {
	switch v := c.(type) {
	case color.RGBA:
		if v.A == 0 {
			return Color{}
		}
		return ColorFromLinearSRGB(
			float64(v.R)/float64(v.A),
			float64(v.G)/float64(v.A),
			float64(v.B)/float64(v.A),
			float64(v.A)/0xff,
		)
	case color.RGBA64:
		if v.A == 0 {
			return Color{}
		}
		return ColorFromLinearSRGB(
			float64(v.R)/float64(v.A),
			float64(v.G)/float64(v.A),
			float64(v.B)/float64(v.A),
			float64(v.A)/0xffff,
		)
	case color.NRGBA:
		return ColorFromLinearSRGB(
			float64(v.R)/0xff,
			float64(v.G)/0xff,
			float64(v.B)/0xff,
			float64(v.A)/0xff,
		)
	case color.NRGBA64:
		return ColorFromLinearSRGB(
			float64(v.R)/0xffff,
			float64(v.G)/0xffff,
			float64(v.B)/0xffff,
			float64(v.A)/0xffff,
		)
	case color.Alpha:
		a := float64(v.A) / 0xff
		return ColorFromLinearSRGB(
			1,
			1,
			1,
			a,
		)
	case color.Alpha16:
		a := float64(v.A) / 0xffff
		return ColorFromLinearSRGB(
			1,
			1,
			1,
			a,
		)
	case color.Gray:
		y := float64(v.Y) / 0xff
		return ColorFromLinearSRGB(
			y,
			y,
			y,
			1,
		)
	case color.Gray16:
		y := float64(v.Y) / 0xffff
		return ColorFromLinearSRGB(
			y,
			y,
			y,
			1,
		)
	default:
		r, g, b, a := c.RGBA()
		if a == 0 {
			return Color{}
		}
		return ColorFromLinearSRGB(
			float64(r)/float64(a),
			float64(g)/float64(a),
			float64(b)/float64(a),
			float64(a)/0xffff,
		)
	}
}

// ColorFromDisplayP3 builds a Color from nonlinear Display P3 channels in [0,1] and alpha.
func ColorFromDisplayP3(r, g, b, alpha float64) Color {
	r = degamma(r)
	g = degamma(g)
	b = degamma(b)

	return ColorFromLinearDisplayP3(r, g, b, alpha)
}

// ColorFromLinearDisplayP3 builds a Color from linear Display P3 channels in [0,1] and alpha.
func ColorFromLinearDisplayP3(r, g, b, alpha float64) Color {
	return Color{
		x:     r*608311/1250200 + g*189793/714400 + b*198249/1000160,
		y:     r*35783/156275 + g*247089/357200 + b*198249/2500400,
		z:     g*32229/714400 + b*5220557/5000800,
		alpha: alpha,
	}
}

// ColorFromOKLab builds a Color from OKLab components and alpha.
func ColorFromOKLab(l, a, b, alpha float64) Color {
	l_ := l + 0.3963377773761749*a + 0.2158037573099136*b
	m_ := l - 0.1055613458156586*a + -0.0638541728258133*b
	s_ := l - 0.0894841775298119*a - 1.2914855480194092*b

	l_ = l_ * l_ * l_
	m_ = m_ * m_ * m_
	s_ = s_ * s_ * s_

	return Color{
		x:     1.2268798758459243*l_ - 0.5578149944602171*m_ + 0.2813910456659647*s_,
		y:     -0.0405757452148008*l_ + 1.1122868032803170*m_ - 0.0717110580655164*s_,
		z:     -0.0763729366746601*l_ - 0.4214933324022432*m_ + 1.5869240198367816*s_,
		alpha: alpha,
	}
}

// ColorFromOKLch builds a Color from OKLCh components (h in radians) and alpha.
func ColorFromOKLch(l, c, h, alpha float64) Color {
	a := math.Cos(h) * c
	b := math.Sin(h) * c
	return ColorFromOKLab(l, a, b, alpha)
}

// XYZ returns the XYZ D65 coordinates and alpha.
func (c Color) XYZ() (x, y, z, a float64) {
	return c.x, c.y, c.z, c.alpha
}

// SRGB converts Color to nonlinear sRGB channels and alpha.
func (c Color) SRGB() (r, g, b, a float64) {
	r, g, b, a = c.LinearSRGB()
	r = gamma(r)
	g = gamma(g)
	b = gamma(b)
	return
}

// SRGBColor converts Color to a nonlinear sRGB [color.Color].
func (c Color) SRGBColor() color.Color {
	r, g, b, a := c.SRGB()
	return color.NRGBA64{
		R: toUint16(r),
		G: toUint16(g),
		B: toUint16(b),
		A: toUint16(a),
	}
}

// LinearSRGB converts Color to linear sRGB channels and alpha.
func (c Color) LinearSRGB() (r, g, b, a float64) {
	r = c.x*12831/3959 + c.y*-329/214 + c.z*-1974/3959
	g = c.x*-851781/878810 + c.y*1648619/878810 + c.z*36519/878810
	b = c.x*705/12673 + c.y*-2585/12673 + c.z*705/667
	a = c.alpha
	return
}

// LinearSRGBColor converts Color to a linear sRGB [color.Color].
func (c Color) LinearSRGBColor() color.Color {
	r, g, b, a := c.LinearSRGB()
	return color.NRGBA64{
		R: toUint16(r),
		G: toUint16(g),
		B: toUint16(b),
		A: toUint16(a),
	}
}

// DisplayP3 converts Color to nonlinear Display P3 channels and alpha.
func (c Color) DisplayP3() (r, g, b, a float64) {
	r, g, b, a = c.LinearDisplayP3()
	r = gamma(r)
	g = gamma(g)
	b = gamma(b)
	return
}

// LinearDisplayP3 converts Color to linear Display P3 channels and alpha.
func (c Color) LinearDisplayP3() (r, g, b, a float64) {
	r = c.x*446124/178915 + c.y*-333277/357830 + c.z*-72051/178915
	g = c.x*-14852/17905 + c.y*63121/35810 + c.z*423/17905
	b = c.x*11844/330415 + c.y*-50337/660830 + c.z*316169/330415
	a = c.alpha
	return
}

// OKLab converts Color to OKLab components and alpha.
func (c Color) OKLab() (l, a, b, alpha float64) {
	l_ := 0.8190224379967030*c.x + 0.3619062600528904*c.y + -0.1288737815209879*c.z
	m_ := 0.0329836539323885*c.x + 0.9292868615863434*c.y + 0.0361446663506424*c.z
	s_ := 0.0481771893596242*c.x + 0.2642395317527308*c.y + 0.6335478284694309*c.z

	l_ = math.Cbrt(l_)
	m_ = math.Cbrt(m_)
	s_ = math.Cbrt(s_)

	l = 0.2104542683093140*l_ + 0.7936177747023054*m_ + -0.0040720430116193*s_
	a = 1.9779985324311684*l_ + -2.4285922420485799*m_ + 0.4505937096174110*s_
	b = 0.0259040424655478*l_ + 0.7827717124575296*m_ + -0.8086757549230774*s_
	alpha = c.alpha
	return
}

// OKLch converts Color to OKLCh components (h in radians) and alpha.
func (c Color) OKLch() (l, ch, h, alpha float64) {
	l, a, b, alpha := c.OKLab()
	ch = math.Hypot(a, b)
	h = math.Atan2(b, a)
	return
}

func degamma(x float64) float64 {
	// https://www.w3.org/TR/css-color-4/#color-conversion-code
	sign := math.Copysign(1, x)
	abs := math.Abs(x)
	if abs <= 0.04045 {
		return x / 12.92
	}
	return sign * math.Pow((abs+0.055)/1.055, 2.4)
}

func gamma(x float64) float64 {
	// https://www.w3.org/TR/css-color-4/#color-conversion-code
	sign := math.Copysign(1, x)
	abs := math.Abs(x)
	if abs <= 0.0031308 {
		return 12.92 * x
	}
	return sign*1.055*math.Pow(abs, 1/2.4) - 0.055
}

func toUint16(v float64) uint16 {
	return uint16(min(max(math.Round(v*0xffff), 0), 0xffff))
}
