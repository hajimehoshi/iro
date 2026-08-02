# Iro (色)

Package iro provides color conversion utilities between various color spaces.

Unlike the standard library’s `color.Color`, it keeps values in the XYZ D65 space to minimize loss when moving between spaces (within floating-point limits).

## Usage

```go
package main

import (
	"fmt"

	"github.com/hajimehoshi/iro"
)

func main() {
	// Convert nonlinear sRGB to Oklch.
	c := iro.ColorFromSRGB(0.2, 0.4, 0.6, 1)
	l, ch, h, alpha := c.Oklch()

	fmt.Printf("L=%.6f C=%.6f h=%.2f° alpha=%.2f\n", l, ch, h, alpha)
}
```

## Munsell

`ColorFromMunsell` converts a Munsell hue, value and chroma. The hue is a position on
the 100-step Munsell hue circle, so `5R 4/14` is `MunsellR+5` with value 4 and chroma 14.

```go
c := iro.ColorFromMunsell(iro.MunsellR+5, 4, 14, 1)
r, g, b, _ := c.SRGB()
```

The conversion interpolates the Munsell renotation data, which is referred to CIE
illuminant C. The result is adapted to D65 with the Bradford transform, so it composes
with the other spaces here. A chroma beyond what the data covers is clamped to it.

The conversion is one-way for now: there is no XYZ to Munsell direction, because
inverting the renotation is iterative and does not converge for every input.

## Notice

Iro is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

`munselldata/all.dat` and `munselldata/real.dat` are verbatim copies of the Munsell
renotation data published by the [Munsell Color Science Laboratory, Rochester Institute
of Technology](https://www.rit.edu/science/munsell-color-science-lab-educational-resources#munsell-renotation-data),
from which `munselltable.go` is generated. Of the 4995 entries of `all.dat`, 2734 are
the renotation itself, 765 are its extension to values below 1, and the rest are
extrapolated beyond real surface colours:

- S. M. Newhall, D. Nickerson and D. B. Judd, “Final Report of the O.S.A. Subcommittee
  on the Spacing of the Munsell Colors”, *Journal of the Optical Society of America*,
  33(7):385-418 (1943).
- D. B. Judd and G. Wyszecki, “Extension of the Munsell Renotation System to Very Dark
  Colors”, *Journal of the Optical Society of America*, 46(4):281-284 (1956).

Both are codified as Tables 2 and 3 of ASTM D1535-14(2023), Standard Practice for
Specifying Color by the Munsell System, whose value scale `munsell.go` follows.

RIT states no licence for these files, so they are redistributed here without one. They
are published without restriction and are widely redistributed, and the values
themselves are measurements, but no explicit grant has been made.

The conversion in `munsell.go` does not derive from an existing implementation. In
particular it does not derive from Paul Centore’s Munsell and Kubelka-Munk Toolbox,
which is licensed under the GPL, nor from the ports of it.
