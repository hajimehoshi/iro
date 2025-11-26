// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Hajime Hoshi

package iro

import (
	"math"
	"testing"
)

const tol = 1e-6

func check(got, want float64) (float64, bool) {
	diff := math.Abs(got - want)
	return diff, diff <= tol
}

func TestXYZRoundTrip(t *testing.T) {
	x0, y0, z0, a0 := 0.3, 0.4, 0.5, 0.6
	c := ColorFromXYZ(x0, y0, z0, a0)
	x1, y1, z1, a1 := c.XYZ()

	if diff, ok := check(x1, x0); !ok {
		t.Errorf("x: got %f, want %f (diff=%g)", x1, x0, diff)
	}
	if diff, ok := check(y1, y0); !ok {
		t.Errorf("y: got %f, want %f (diff=%g)", y1, y0, diff)
	}
	if diff, ok := check(z1, z0); !ok {
		t.Errorf("z: got %f, want %f (diff=%g)", z1, z0, diff)
	}
	if diff, ok := check(a1, a0); !ok {
		t.Errorf("a: got %f, want %f (diff=%g)", a1, a0, diff)
	}
}

func TestSRGBRoundTrip(t *testing.T) {
	r0, g0, b0, a0 := 0.2, 0.4, 0.6, 0.8
	c := ColorFromSRGB(r0, g0, b0, a0)
	r1, g1, b1, a1 := c.SRGB()

	if diff, ok := check(r1, r0); !ok {
		t.Errorf("r: got %f, want %f (diff=%g)", r1, r0, diff)
	}
	if diff, ok := check(g1, g0); !ok {
		t.Errorf("g: got %f, want %f (diff=%g)", g1, g0, diff)
	}
	if diff, ok := check(b1, b0); !ok {
		t.Errorf("b: got %f, want %f (diff=%g)", b1, b0, diff)
	}
	if diff, ok := check(a1, a0); !ok {
		t.Errorf("a: got %f, want %f (diff=%g)", a1, a0, diff)
	}
}

func TestDisplayP3RoundTrip(t *testing.T) {
	r0, g0, b0, a0 := 0.1, 0.7, 0.3, 0.5
	c := ColorFromDisplayP3(r0, g0, b0, a0)
	r1, g1, b1, a1 := c.DisplayP3()

	if diff, ok := check(r1, r0); !ok {
		t.Errorf("r: got %f, want %f (diff=%g)", r1, r0, diff)
	}
	if diff, ok := check(g1, g0); !ok {
		t.Errorf("g: got %f, want %f (diff=%g)", g1, g0, diff)
	}
	if diff, ok := check(b1, b0); !ok {
		t.Errorf("b: got %f, want %f (diff=%g)", b1, b0, diff)
	}
	if diff, ok := check(a1, a0); !ok {
		t.Errorf("a: got %f, want %f (diff=%g)", a1, a0, diff)
	}
}

func TestOKLabRoundTrip(t *testing.T) {
	l0, a0, b0, alpha0 := 0.5, 0.1, -0.2, 0.9
	c := ColorFromOKLab(l0, a0, b0, alpha0)
	l1, a1, b1, alpha1 := c.OKLab()

	if diff, ok := check(l1, l0); !ok {
		t.Errorf("l: got %f, want %f (diff=%g)", l1, l0, diff)
	}
	if diff, ok := check(a1, a0); !ok {
		t.Errorf("a: got %f, want %f (diff=%g)", a1, a0, diff)
	}
	if diff, ok := check(b1, b0); !ok {
		t.Errorf("b: got %f, want %f (diff=%g)", b1, b0, diff)
	}
	if diff, ok := check(alpha1, alpha0); !ok {
		t.Errorf("alpha: got %f, want %f (diff=%g)", alpha1, alpha0, diff)
	}
}

func TestChainedConversions(t *testing.T) {
	r0, g0, b0, a0 := 0.15, 0.35, 0.55, 0.75

	cSRGB := ColorFromSRGB(r0, g0, b0, a0)
	p3r, p3g, p3b, p3a := cSRGB.DisplayP3()

	cP3 := ColorFromDisplayP3(p3r, p3g, p3b, p3a)
	l, aComp, bComp, alpha := cP3.OKLab()

	cLab := ColorFromOKLab(l, aComp, bComp, alpha)
	r1, g1, b1, a1 := cLab.SRGB()

	if diff, ok := check(r1, r0); !ok {
		t.Errorf("r: got %f, want %f (diff=%g)", r1, r0, diff)
	}
	if diff, ok := check(g1, g0); !ok {
		t.Errorf("g: got %f, want %f (diff=%g)", g1, g0, diff)
	}
	if diff, ok := check(b1, b0); !ok {
		t.Errorf("b: got %f, want %f (diff=%g)", b1, b0, diff)
	}
	if diff, ok := check(a1, a0); !ok {
		t.Errorf("a: got %f, want %f (diff=%g)", a1, a0, diff)
	}
}
