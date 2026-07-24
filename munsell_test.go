// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package iro_test

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/iro"
)

// renotationSamples are entries of the Munsell renotation data, whose chromaticities
// are referred to illuminant C.
var renotationSamples = []struct {
	name             string
	hue, value, chro float64
	x, y             float64
}{
	{"10RP 1/2", iro.MunsellRP + 10, 1, 2, 0.3629, 0.2710},
	{"7.5PB 1/38", iro.MunsellPB + 7.5, 1, 38, 0.1680, 0.0140},
	{"7.5G 2/4", iro.MunsellG + 7.5, 2, 4, 0.2540, 0.3705},
	{"5P 3/10", iro.MunsellP + 5, 3, 10, 0.2772, 0.1707},
	{"5R 4/14", iro.MunsellR + 5, 4, 14, 0.5734, 0.3057},
	{"10B 4/8", iro.MunsellB + 10, 4, 8, 0.1893, 0.2160},
	{"10GY 5/10", iro.MunsellGY + 10, 5, 10, 0.3028, 0.5237},
	{"2.5B 6/8", iro.MunsellB + 2.5, 6, 8, 0.2080, 0.2789},
	{"10YR 7/12", iro.MunsellYR + 10, 7, 12, 0.4900, 0.4480},
	{"5Y 8/12", iro.MunsellY + 5, 8, 12, 0.4562, 0.4788},
	{"2.5R 9/2", iro.MunsellR + 2.5, 9, 2, 0.3210, 0.3168},
	{"2.5P 9/4", iro.MunsellP + 2.5, 9, 4, 0.2963, 0.2865},
}

func TestMunsellChromaticityAtRenotationPoints(t *testing.T) {
	for _, tc := range renotationSamples {
		t.Run(tc.name, func(t *testing.T) {
			x, y := iro.MunsellChromaticity(tc.hue, tc.value, tc.chro)
			if diff, ok := check(x, tc.x); !ok {
				t.Errorf("x: got %f, want %f (diff=%g)", x, tc.x, diff)
			}
			if diff, ok := check(y, tc.y); !ok {
				t.Errorf("y: got %f, want %f (diff=%g)", y, tc.y, diff)
			}
		})
	}
}

func TestMunsellLuminance(t *testing.T) {
	// Value 10 is the perfect diffuser by definition.
	if diff, ok := check(iro.MunsellLuminance(10), 1); !ok {
		t.Errorf("value 10: got %f, want 1 (diff=%g)", iro.MunsellLuminance(10), diff)
	}
	if got := iro.MunsellLuminance(0); got != 0 {
		t.Errorf("value 0: got %f, want 0", got)
	}

	// The evaluation is a rewriting of the quintic of ASTM D1535.
	for v := 0.0; v <= 10; v += 0.05 {
		want := (1.1914*v - 0.22533*v*v + 0.23352*v*v*v -
			0.020484*v*v*v*v + 0.00081939*v*v*v*v*v) / 100
		if diff, ok := check(iro.MunsellLuminance(v), want); !ok {
			t.Errorf("value %g: got %f, want %f (diff=%g)", v, iro.MunsellLuminance(v), want, diff)
		}
	}

	// The luminance factor increases with value.
	prev := -1.0
	for v := 0.0; v <= 10; v += 0.01 {
		got := iro.MunsellLuminance(v)
		if got <= prev {
			t.Fatalf("value %g: got %f, want more than %f", v, got, prev)
		}
		prev = got
	}
}

// TestMunsellNeutralIsIlluminant checks that a chroma of 0 is the illuminant whatever
// the hue, which is what lets a neutral be written with any hue at all.
func TestMunsellNeutralIsIlluminant(t *testing.T) {
	for _, value := range []float64{0.5, 1, 5, 9, 10} {
		for _, hue := range []float64{-37.5, 0, 2.5, 33, 50, 77.5, 100, 250} {
			x, y := iro.MunsellChromaticity(hue, value, 0)
			if diff, ok := check(x, iro.IlluminantCx); !ok {
				t.Errorf("hue %g value %g: x: got %f, want %f (diff=%g)",
					hue, value, x, iro.IlluminantCx, diff)
			}
			if diff, ok := check(y, iro.IlluminantCy); !ok {
				t.Errorf("hue %g value %g: y: got %f, want %f (diff=%g)",
					hue, value, y, iro.IlluminantCy, diff)
			}
		}
	}
}

// TestMunsellNeutralAdaptsToWhitePoint checks the adaptation from illuminant C:
// a neutral of value 10 is the illuminant itself, so it must land exactly on the
// D65 white point that the other spaces of this package are referred to.
func TestMunsellNeutralAdaptsToWhitePoint(t *testing.T) {
	wx, wy, wz, _ := iro.ColorFromLinearSRGB(1, 1, 1, 1).XYZ()

	x, y, z, alpha := iro.ColorFromMunsell(0, 10, 0, 1).XYZ()
	if diff, ok := check(x, wx); !ok {
		t.Errorf("x: got %f, want %f (diff=%g)", x, wx, diff)
	}
	if diff, ok := check(y, wy); !ok {
		t.Errorf("y: got %f, want %f (diff=%g)", y, wy, diff)
	}
	if diff, ok := check(z, wz); !ok {
		t.Errorf("z: got %f, want %f (diff=%g)", z, wz, diff)
	}
	if alpha != 1 {
		t.Errorf("alpha: got %f, want 1", alpha)
	}

	// A neutral of any value stays neutral, so it converts to an equal-channel sRGB.
	for _, value := range []float64{1, 2.5, 5, 7.5, 9} {
		r, g, b, _ := iro.ColorFromMunsell(0, value, 0, 1).SRGB()
		if diff, ok := check(r, g); !ok {
			t.Errorf("value %g: r and g differ: %f vs %f (diff=%g)", value, r, g, diff)
		}
		if diff, ok := check(g, b); !ok {
			t.Errorf("value %g: g and b differ: %f vs %f (diff=%g)", value, g, b, diff)
		}
	}
}

func TestMunsellBlack(t *testing.T) {
	x, y, z, alpha := iro.ColorFromMunsell(0, 0, 0, 1).XYZ()
	if x != 0 || y != 0 || z != 0 {
		t.Errorf("got (%f, %f, %f), want (0, 0, 0)", x, y, z)
	}
	if alpha != 1 {
		t.Errorf("alpha: got %f, want 1", alpha)
	}
}

func TestMunsellHueWrapsAroundTheCircle(t *testing.T) {
	for _, tc := range renotationSamples {
		t.Run(tc.name, func(t *testing.T) {
			x, y := iro.MunsellChromaticity(tc.hue, tc.value, tc.chro)
			for _, turn := range []float64{-200, -100, 100, 200} {
				x2, y2 := iro.MunsellChromaticity(tc.hue+turn, tc.value, tc.chro)
				if diff, ok := check(x2, x); !ok {
					t.Errorf("hue%+g: x: got %f, want %f (diff=%g)", turn, x2, x, diff)
				}
				if diff, ok := check(y2, y); !ok {
					t.Errorf("hue%+g: y: got %f, want %f (diff=%g)", turn, y2, y, diff)
				}
			}
		})
	}
}

// TestMunsellChromaIsClamped checks that a chroma beyond the renotation data settles
// on the largest chroma it covers rather than extrapolating.
func TestMunsellChromaIsClamped(t *testing.T) {
	for _, hue := range []float64{2.5, 17.5, 42.5, 77.5, 100} {
		for _, value := range []float64{1, 5, 9} {
			x, y := iro.MunsellChromaticity(hue, value, 1000)
			x2, y2 := iro.MunsellChromaticity(hue, value, 5000)
			if diff, ok := check(x2, x); !ok {
				t.Errorf("hue %g value %g: x: got %f, want %f (diff=%g)", hue, value, x2, x, diff)
			}
			if diff, ok := check(y2, y); !ok {
				t.Errorf("hue %g value %g: y: got %f, want %f (diff=%g)", hue, value, y2, y, diff)
			}
		}
	}
}

// TestMunsellIsFinite sweeps the whole domain, including beyond it, to check that no
// combination produces a non-finite result.
func TestMunsellIsFinite(t *testing.T) {
	for hue := -50.0; hue <= 150; hue += 0.5 {
		for value := 0.0; value <= 10; value += 0.25 {
			for chroma := 0.0; chroma <= 60; chroma += 1.5 {
				x, y, z, _ := iro.ColorFromMunsell(hue, value, chroma, 1).XYZ()
				if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) ||
					math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
					t.Fatalf("hue %g value %g chroma %g: got (%f, %f, %f)", hue, value, chroma, x, y, z)
				}
			}
		}
	}
}

// TestMunsellIsContinuous checks that the interpolation has no steps in it, at a
// chroma the renotation data covers for every hue at these values.
func TestMunsellIsContinuous(t *testing.T) {
	for _, value := range []float64{2, 5, 8} {
		var px, py float64
		for hue := 0.0; hue <= 100; hue += 0.05 {
			x, y := iro.MunsellChromaticity(hue, value, 2)
			if hue > 0 {
				if d := math.Hypot(x-px, y-py); d > 1e-3 {
					t.Fatalf("value %g: step of %g at hue %g", value, d, hue)
				}
			}
			px, py = x, y
		}
	}

	for _, hue := range []float64{5, 27.5, 63.75, 88} {
		var px, py float64
		for value := 1.0; value <= 9; value += 0.01 {
			x, y := iro.MunsellChromaticity(hue, value, 2)
			if value > 1 {
				if d := math.Hypot(x-px, y-py); d > 1e-3 {
					t.Fatalf("hue %g: step of %g at value %g", hue, d, value)
				}
			}
			px, py = x, y
		}
	}
}

// TestMunsellStaysNearItsSamples bounds how far the interpolation departs from the
// samples it runs between. The renotation samples at a fixed value and chroma lie on
// an ovoid that bulges between them, so some departure is expected, but a large one
// would mean the interpolation had begun to oscillate.
func TestMunsellStaysNearItsSamples(t *testing.T) {
	radius := func(x, y float64) float64 {
		return math.Hypot(x-iro.IlluminantCx, y-iro.IlluminantCy)
	}
	// A chroma beyond what the data covers settles on the largest one it does, which
	// shows up as the result no longer changing with chroma.
	clamped := func(hue, value, chroma float64) bool {
		x1, y1 := iro.MunsellChromaticity(hue, value, chroma)
		x2, y2 := iro.MunsellChromaticity(hue, value, chroma+0.25)
		return x1 == x2 && y1 == y2
	}

	for _, value := range []float64{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		for chroma := 2.0; chroma <= 20; chroma += 2 {
			for i := 0; i < 40; i++ {
				h1 := float64(i)*2.5 + 2.5
				h2 := h1 + 2.5
				// Restrict to where the chroma is covered right across the span used,
				// so that clamping is not mistaken for the interpolation departing.
				if clamped(h1-2.5, value, chroma) || clamped(h1, value, chroma) ||
					clamped(h2, value, chroma) || clamped(h2+2.5, value, chroma) {
					continue
				}

				x1, y1 := iro.MunsellChromaticity(h1, value, chroma)
				x2, y2 := iro.MunsellChromaticity(h2, value, chroma)
				lo := math.Min(radius(x1, y1), radius(x2, y2))
				hi := math.Max(radius(x1, y1), radius(x2, y2))

				for f := 0.05; f < 1; f += 0.05 {
					hue := h1 + 2.5*f
					if clamped(hue, value, chroma) {
						continue
					}
					x, y := iro.MunsellChromaticity(hue, value, chroma)
					r := radius(x, y)
					if r > hi*1.03 || r < lo*0.97 {
						t.Fatalf("value %g chroma %g hue %g: radius %f departs from [%f, %f]",
							value, chroma, hue, r, lo, hi)
					}
				}
			}
		}
	}
}

// TestMunsellToSRGB checks a few colours land in the part of sRGB their hue names.
func TestMunsellToSRGB(t *testing.T) {
	testCases := []struct {
		name             string
		hue, value, chro float64
		want             string
	}{
		{"5R 4/14", iro.MunsellR + 5, 4, 14, "red"},
		{"10GY 5/10", iro.MunsellGY + 10, 5, 10, "green"},
		{"7.5PB 4/12", iro.MunsellPB + 7.5, 4, 12, "blue"},
		{"5Y 8/12", iro.MunsellY + 5, 8, 12, "yellow"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, _ := iro.ColorFromMunsell(tc.hue, tc.value, tc.chro, 1).SRGB()
			var ok bool
			switch tc.want {
			case "red":
				ok = r > g && r > b
			case "green":
				ok = g > r && g > b
			case "blue":
				ok = b > r && b > g
			case "yellow":
				ok = r > b && g > b
			}
			if !ok {
				t.Errorf("got (%f, %f, %f), want a %s", r, g, b, tc.want)
			}
		})
	}
}

func ExampleColorFromMunsell() {
	// Munsell 5R 4/14, a vivid red.
	c := iro.ColorFromMunsell(iro.MunsellR+5, 4, 14, 1)
	r, g, b, _ := c.SRGB()

	fmt.Printf("R=%.4f G=%.4f B=%.4f\n", r, g, b)
	// Output:
	// R=0.7357 G=0.0998 B=0.2006
}

// hueFamilies maps the hue family names of the renotation data to the constants of
// this package.
var hueFamilies = map[string]float64{
	"R": iro.MunsellR, "YR": iro.MunsellYR, "Y": iro.MunsellY, "GY": iro.MunsellGY,
	"G": iro.MunsellG, "BG": iro.MunsellBG, "B": iro.MunsellB, "PB": iro.MunsellPB,
	"P": iro.MunsellP, "RP": iro.MunsellRP,
}

type renotationEntry struct {
	hue, value, chroma float64
	x, y, bigY         float64
}

// readRenotation reads the copy of the renotation data that the table is generated
// from. Its chromaticities are referred to illuminant C and its Y is referred to
// magnesium oxide, on a scale of 100.
func readRenotation(t *testing.T, name string) []renotationEntry {
	t.Helper()

	f, err := os.Open(filepath.Join("munselldata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var entries []renotationEntry
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 6 || fields[0] == "h" || fields[0] == "H" {
			continue
		}

		i := strings.IndexFunc(fields[0], func(r rune) bool {
			return r != '.' && (r < '0' || r > '9')
		})
		if i <= 0 {
			t.Fatalf("invalid hue %q", fields[0])
		}
		prefix, err := strconv.ParseFloat(fields[0][:i], 64)
		if err != nil {
			t.Fatal(err)
		}
		base, ok := hueFamilies[fields[0][i:]]
		if !ok {
			t.Fatalf("unknown hue family in %q", fields[0])
		}

		var e renotationEntry
		e.hue = base + prefix
		for _, p := range []struct {
			dst *float64
			s   string
		}{
			{&e.value, fields[1]}, {&e.chroma, fields[2]},
			{&e.x, fields[3]}, {&e.y, fields[4]}, {&e.bigY, fields[5]},
		} {
			v, err := strconv.ParseFloat(p.s, 64)
			if err != nil {
				t.Fatal(err)
			}
			*p.dst = v
		}
		entries = append(entries, e)
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatalf("no entries read from %s", name)
	}
	return entries
}

// TestMunsellLuminanceMatchesRenotation checks the coefficients of the quintic against
// the luminance factors the renotation data itself lists.
func TestMunsellLuminanceMatchesRenotation(t *testing.T) {
	entries := readRenotation(t, "real.dat")

	// The Y of the data is referred to magnesium oxide, whose reflectance is 0.975 of
	// the perfect reflecting diffuser that the quintic is referred to. Both are given
	// to a few digits only, so they agree to about a part in ten thousand.
	var worst float64
	for _, e := range entries {
		want := e.bigY * 0.975 / 100
		if d := math.Abs(iro.MunsellLuminance(e.value) - want); d > worst {
			worst = d
		}
	}
	t.Logf("checked %d entries, largest difference %.3g", len(entries), worst)

	if worst > 5e-5 {
		t.Errorf("largest difference %.3g, want at most 5e-5", worst)
	}
}

// TestMunsellAdaptsFromIlluminantC derives the adaptation from the Bradford cone
// response matrix again and checks that every entry of the renotation data converts
// to the same colour as the package produces.
func TestMunsellAdaptsFromIlluminantC(t *testing.T) {
	bradford := [3][3]float64{
		{0.8951, 0.2664, -0.1614},
		{-0.7502, 1.7135, 0.0367},
		{0.0389, -0.0685, 1.0296},
	}
	det := bradford[0][0]*(bradford[1][1]*bradford[2][2]-bradford[1][2]*bradford[2][1]) -
		bradford[0][1]*(bradford[1][0]*bradford[2][2]-bradford[1][2]*bradford[2][0]) +
		bradford[0][2]*(bradford[1][0]*bradford[2][1]-bradford[1][1]*bradford[2][0])
	inverse := [3][3]float64{
		{
			(bradford[1][1]*bradford[2][2] - bradford[1][2]*bradford[2][1]) / det,
			(bradford[0][2]*bradford[2][1] - bradford[0][1]*bradford[2][2]) / det,
			(bradford[0][1]*bradford[1][2] - bradford[0][2]*bradford[1][1]) / det,
		},
		{
			(bradford[1][2]*bradford[2][0] - bradford[1][0]*bradford[2][2]) / det,
			(bradford[0][0]*bradford[2][2] - bradford[0][2]*bradford[2][0]) / det,
			(bradford[0][2]*bradford[1][0] - bradford[0][0]*bradford[1][2]) / det,
		},
		{
			(bradford[1][0]*bradford[2][1] - bradford[1][1]*bradford[2][0]) / det,
			(bradford[0][1]*bradford[2][0] - bradford[0][0]*bradford[2][1]) / det,
			(bradford[0][0]*bradford[1][1] - bradford[0][1]*bradford[1][0]) / det,
		},
	}
	apply := func(m [3][3]float64, x, y, z float64) (float64, float64, float64) {
		return m[0][0]*x + m[0][1]*y + m[0][2]*z,
			m[1][0]*x + m[1][1]*y + m[1][2]*z,
			m[2][0]*x + m[2][1]*y + m[2][2]*z
	}

	// The destination is the white point of the other spaces of this package.
	dx, dy, dz, _ := iro.ColorFromLinearSRGB(1, 1, 1, 1).XYZ()
	sr, sg, sb := apply(bradford,
		iro.IlluminantCx/iro.IlluminantCy, 1,
		(1-iro.IlluminantCx-iro.IlluminantCy)/iro.IlluminantCy)
	dr, dg, db := apply(bradford, dx, dy, dz)

	adapt := func(x, y, z float64) (float64, float64, float64) {
		r, g, b := apply(bradford, x, y, z)
		return apply(inverse, r*dr/sr, g*dg/sg, b*db/sb)
	}

	entries := readRenotation(t, "real.dat")
	var worst float64
	var worstAt renotationEntry
	for _, e := range entries {
		bigY := iro.MunsellLuminance(e.value)
		wx, wy, wz := adapt(e.x*bigY/e.y, bigY, (1-e.x-e.y)*bigY/e.y)

		x, y, z, _ := iro.ColorFromMunsell(e.hue, e.value, e.chroma, 1).XYZ()
		if d := math.Sqrt((x-wx)*(x-wx) + (y-wy)*(y-wy) + (z-wz)*(z-wz)); d > worst {
			worst, worstAt = d, e
		}
	}
	t.Logf("checked %d entries, largest difference %.3g", len(entries), worst)

	if worst > 1e-12 {
		t.Errorf("largest difference %.3g at hue %g value %g chroma %g, want at most 1e-12",
			worst, worstAt.hue, worstAt.value, worstAt.chroma)
	}
}
